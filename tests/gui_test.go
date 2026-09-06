package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"syncpaper/internal/cache"
	"syncpaper/internal/config"
	"syncpaper/internal/gui"
)

func TestGUIServerEndpoints(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "syncpaper_gui_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.General.StorageDir = tmpDir

	catPath := filepath.Join(tmpDir, "catalog.json")
	cat, err := cache.LoadCatalog(catPath)
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	mgr, err := cache.NewManager(cfg, cat)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	srv, err := gui.NewServer(cfg, cat, mgr, 0)
	if err != nil {
		t.Fatalf("failed to create gui server: %v", err)
	}

	go func() {
		_ = srv.Start()
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// 1. Test Static Index HTML
	resp, err := http.Get(srv.URL() + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for GET /, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Test /api/status
	resp, err = http.Get(srv.URL() + "/api/status")
	if err != nil {
		t.Fatalf("GET /api/status failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for /api/status, got %d", resp.StatusCode)
	}
	var statusData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&statusData); err != nil {
		t.Errorf("failed to parse /api/status json: %v", err)
	}
	resp.Body.Close()

	// 3. Test /api/wallpapers
	resp, err = http.Get(srv.URL() + "/api/wallpapers")
	if err != nil {
		t.Fatalf("GET /api/wallpapers failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for /api/wallpapers, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 4. Test /api/config GET
	resp, err = http.Get(srv.URL() + "/api/config")
	if err != nil {
		t.Fatalf("GET /api/config failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for /api/config, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	if srv.Port() <= 0 {
		t.Errorf("expected valid port, got %d", srv.Port())
	}

	// 5. Add a dummy wallpaper to catalog and test /api/favorite & /api/blacklist
	dummyPath := filepath.Join(tmpDir, "dummy.jpg")
	_ = os.WriteFile(dummyPath, []byte("fake"), 0644)
	cat.AddWallpaper(cache.CachedWallpaper{
		ID:        "dummy_wp",
		LocalPath: dummyPath,
		Title:     "Dummy",
	})

	// Test favorite POST
	favBody := `{"id":"dummy_wp"}`
	resp, err = http.Post(srv.URL()+"/api/favorite", "application/json", strings.NewReader(favBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Errorf("POST /api/favorite failed: status %v, err %v", resp.StatusCode, err)
	}
	resp.Body.Close()

	// Test blacklist POST
	blBody := `{"id":"dummy_wp"}`
	resp, err = http.Post(srv.URL()+"/api/blacklist", "application/json", strings.NewReader(blBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Errorf("POST /api/blacklist failed: status %v, err %v", resp.StatusCode, err)
	}
	resp.Body.Close()

	// Test wallpaper image 404 for missing ID
	resp, err = http.Get(srv.URL() + "/api/wallpaper/image?id=not_found")
	if err != nil || resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for missing image, got %v", resp.StatusCode)
	}
	resp.Body.Close()

	// Test config POST
	cfgBody := `{"topics":{"list":["cyberpunk","minimal"]}}`
	resp, err = http.Post(srv.URL()+"/api/config", "application/json", strings.NewReader(cfgBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Errorf("POST /api/config failed: status %v, err %v", resp.StatusCode, err)
	}
	resp.Body.Close()

	// 6. Test /api/close triggers srv.Done()
	postResp, err := http.Post(srv.URL()+"/api/close", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST /api/close failed: %v", err)
	}
	if postResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for /api/close, got %d", postResp.StatusCode)
	}
	postResp.Body.Close()

	select {
	case <-srv.Done():
		// Success!
	case <-time.After(1 * time.Second):
		t.Errorf("expected srv.Done() to be closed after POST /api/close")
	}
}
