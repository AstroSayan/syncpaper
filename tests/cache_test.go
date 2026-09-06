package tests

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"syncpaper/internal/cache"
	"syncpaper/internal/config"
	"syncpaper/internal/source"
)

func TestCatalogOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "syncpaper_cat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	catPath := filepath.Join(tmpDir, "catalog.json")
	cat, err := cache.LoadCatalog(catPath)
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	w1 := cache.CachedWallpaper{
		ID:           "test_1",
		Source:       "wallhaven",
		Topic:        "space",
		Title:        "Nebula",
		LocalPath:    "/tmp/nebula.jpg",
		Hash:         "hash123",
		Width:        1920,
		Height:       1080,
		DownloadedAt: time.Now(),
	}

	cat.AddWallpaper(w1)
	if !cat.HasHash("hash123") {
		t.Errorf("expected catalog to have hash123")
	}

	cat.SetCurrent("test_1")
	if cat.CurrentID != "test_1" {
		t.Errorf("expected CurrentID=test_1, got %s", cat.CurrentID)
	}
	if len(cat.History) != 1 || cat.History[0] != "test_1" {
		t.Errorf("history error: %v", cat.History)
	}

	// Toggle favorite
	fav, err := cat.ToggleFavorite("test_1")
	if err != nil || !fav {
		t.Errorf("expected favorite to be true, got %v, err %v", fav, err)
	}

	// Save and reload
	if err := cat.Save(); err != nil {
		t.Fatalf("failed to save catalog: %v", err)
	}

	reloaded, err := cache.LoadCatalog(catPath)
	if err != nil {
		t.Fatalf("failed to reload catalog: %v", err)
	}

	if len(reloaded.Wallpapers) != 1 {
		t.Errorf("expected 1 wallpaper, got %d", len(reloaded.Wallpapers))
	}
	if !reloaded.Wallpapers[0].Favorite {
		t.Errorf("expected wallpaper favorite to be true")
	}

	// Blacklist
	if err := reloaded.Blacklist("test_1"); err != nil {
		t.Errorf("blacklist failed: %v", err)
	}
	if len(reloaded.Wallpapers) != 0 {
		t.Errorf("expected 0 wallpapers after blacklist, got %d", len(reloaded.Wallpapers))
	}
	if !reloaded.HasHash("hash123") {
		t.Errorf("expected blacklisted hash to still be recorded")
	}
}

func TestCachePruning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "syncpaper_prune_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	catPath := filepath.Join(tmpDir, "catalog.json")
	cat, _ := cache.LoadCatalog(catPath)

	cfg := config.DefaultConfig()
	cfg.General.StorageDir = tmpDir
	cfg.General.MaxCached = 2

	mgr, err := cache.NewManager(cfg, cat)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Create 3 dummy wallpaper files
	for i := 1; i <= 3; i++ {
		p := filepath.Join(tmpDir, "wall_"+string(rune('0'+i))+".jpg")
		_ = os.WriteFile(p, []byte("fake image data"), 0644)

		cat.AddWallpaper(cache.CachedWallpaper{
			ID:           "id_" + string(rune('0'+i)),
			LocalPath:    p,
			DownloadedAt: time.Now().Add(time.Duration(i) * time.Minute),
			Favorite:     i == 1, // id_1 is favorite!
		})
	}

	// With max 2 and 3 items (where 1 is favorite), prune should delete 1 non-favorite
	deleted := mgr.Prune()
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	if len(cat.Wallpapers) != 2 {
		t.Errorf("expected 2 wallpapers remaining, got %d", len(cat.Wallpapers))
	}

	// Ensure id_1 (favorite) was preserved
	foundFav := false
	for _, w := range cat.Wallpapers {
		if w.ID == "id_1" {
			foundFav = true
		}
	}
	if !foundFav {
		t.Errorf("expected favorite wallpaper to be preserved during pruning")
	}
}

func TestDownloadAndCache(t *testing.T) {
	// Create mock image server serving 1920x1080 image
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 50, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode mock image: %v", err)
	}
	imgBytes := buf.Bytes()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imgBytes)
	}))
	defer ts.Close()

	tmpDir, err := os.MkdirTemp("", "syncpaper_download_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.General.StorageDir = tmpDir
	cfg.General.MaxCached = 10

	catPath := filepath.Join(tmpDir, "catalog.json")
	cat, err := cache.LoadCatalog(catPath)
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	mgr, err := cache.NewManager(cfg, cat)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	items := []source.WallpaperItem{
		{
			ID:     "mock_1",
			Source: "wallhaven",
			Topic:  "nature",
			Title:  "Test Wallpaper",
			URL:    ts.URL + "/image1.png",
			Format: "png",
		},
	}

	ctx := context.Background()
	saved, errs := mgr.DownloadAll(ctx, items, 2)
	if len(errs) > 0 {
		t.Fatalf("unexpected download errors: %v", errs)
	}
	if len(saved) != 1 {
		t.Fatalf("expected 1 downloaded wallpaper, got %d", len(saved))
	}
	if saved[0].Width != 1920 || saved[0].Height != 1080 {
		t.Errorf("expected 1920x1080, got %dx%d", saved[0].Width, saved[0].Height)
	}

	// Test Deduplication: second download of same image should skip
	saved2, _ := mgr.DownloadAll(ctx, items, 2)
	if len(saved2) != 0 {
		t.Errorf("expected 0 downloaded for duplicate hash, got %d", len(saved2))
	}
}

func TestCatalogLookups(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "syncpaper_lookup_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cat, _ := cache.LoadCatalog(filepath.Join(tmpDir, "catalog.json"))
	w := cache.CachedWallpaper{
		ID:        "find_me",
		Title:     "Target",
		LocalPath: "/tmp/target.jpg",
	}
	cat.AddWallpaper(w)

	found := cat.GetByID("find_me")
	if found == nil || found.Title != "Target" {
		t.Errorf("expected to find wallpaper by ID, got %+v", found)
	}

	missing := cat.GetByID("does_not_exist")
	if missing != nil {
		t.Errorf("expected nil for non-existent ID, got %+v", missing)
	}

	_ = cache.GetCatalogPath()
}
