package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "golang.org/x/image/webp"

	"syncpaper/internal/config"
	"syncpaper/internal/source"
)

type Manager struct {
	cfg     *config.Config
	catalog *Catalog
	dir     string
}

func NewManager(cfg *config.Config, catalog *Catalog) (*Manager, error) {
	storageDir := config.ExpandPath(cfg.General.StorageDir)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory %s: %w", storageDir, err)
	}

	return &Manager{
		cfg:     cfg,
		catalog: catalog,
		dir:     storageDir,
	}, nil
}

// DownloadAll downloads a list of wallpaper items concurrently with deduplication
func (m *Manager) DownloadAll(ctx context.Context, items []source.WallpaperItem, concurrency int) ([]CachedWallpaper, []error) {
	if concurrency <= 0 {
		concurrency = 4
	}

	var (
		wg      sync.WaitGroup
		sem     = make(chan struct{}, concurrency)
		mu      sync.Mutex
		saved   []CachedWallpaper
		errList []error
	)

	for _, item := range items {
		wg.Add(1)
		sem <- struct{}{}

		go func(it source.WallpaperItem) {
			defer wg.Done()
			defer func() { <-sem }()

			cw, err := m.downloadOne(ctx, it)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errList = append(errList, fmt.Errorf("download error [%s]: %w", it.ID, err))
			} else if cw != nil {
				saved = append(saved, *cw)
			}
		}(item)
	}

	wg.Wait()
	_ = m.catalog.Save()
	m.Prune()
	return saved, errList
}

func (m *Manager) downloadOne(ctx context.Context, item source.WallpaperItem) (*CachedWallpaper, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp(m.dir, "syncpaper_dl_*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		_ = os.Remove(tmpPath) // Cleanup if not renamed
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid download URL %s: %w", item.URL, err)
	}
	req.Header.Set("User-Agent", "syncpaper/1.0 (Linux; wallpaper utility)")

	resp, err := source.DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned HTTP %d for %s", resp.StatusCode, item.URL)
	}

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("stream copy failed: %w", err)
	}
	_ = tmpFile.Sync()

	hashStr := hex.EncodeToString(hasher.Sum(nil))

	// Check deduplication
	if m.catalog.HasHash(hashStr) {
		// Already downloaded or blacklisted
		return nil, nil
	}

	// Validate image & get real dimensions
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek temp file: %w", err)
	}

	cfg, format, err := image.DecodeConfig(tmpFile)
	if err != nil {
		// If decoding failed, check format from item or fallback
		format = "jpg"
	}

	width := cfg.Width
	height := cfg.Height
	if width == 0 {
		width = item.Width
	}
	if height == 0 {
		height = item.Height
	}

	// Filter out tiny images/thumbnails: wallpaper should be at least 1280x720
	if width > 0 && height > 0 && (width < 1280 || height < 720) {
		return nil, nil
	}

	ext := "." + format
	if format == "jpeg" {
		ext = ".jpg"
	}
	if ext == "." {
		ext = ".jpg"
	}

	safeID := sanitizeFilename(item.ID)
	finalName := fmt.Sprintf("%s_%s%s", safeID, hashStr[:8], ext)
	finalPath := filepath.Join(m.dir, finalName)

	tmpFile.Close()
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return nil, fmt.Errorf("failed to finalize image %s: %w", finalPath, err)
	}

	cw := CachedWallpaper{
		ID:           item.ID,
		Source:       item.Source,
		Topic:        item.Topic,
		Title:        item.Title,
		URL:          item.URL,
		LocalPath:    finalPath,
		Hash:         hashStr,
		Width:        width,
		Height:       height,
		SizeBytes:    written,
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Time{},
		Favorite:     false,
		Blacklisted:  false,
	}

	m.catalog.AddWallpaper(cw)
	return &cw, nil
}

// Prune cleans up oldest un-favorited wallpapers when over limit
func (m *Manager) Prune() int {
	maxCount := m.cfg.General.MaxCached
	if maxCount <= 0 {
		return 0
	}

	m.catalog.mu.Lock()
	defer m.catalog.mu.Unlock()

	var candidates []CachedWallpaper
	var kept []CachedWallpaper

	for _, w := range m.catalog.Wallpapers {
		if w.Favorite || w.ID == m.catalog.CurrentID {
			kept = append(kept, w)
		} else {
			candidates = append(candidates, w)
		}
	}

	total := len(kept) + len(candidates)
	if total <= maxCount {
		return 0
	}

	excess := total - maxCount

	// Sort candidates by LastUsedAt (ascending), then DownloadedAt (ascending)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].LastUsedAt.Equal(candidates[j].LastUsedAt) {
			return candidates[i].DownloadedAt.Before(candidates[j].DownloadedAt)
		}
		return candidates[i].LastUsedAt.Before(candidates[j].LastUsedAt)
	})

	deleted := 0
	toDelete := candidates[:excess]
	toKeep := candidates[excess:]

	for _, w := range toDelete {
		_ = os.Remove(w.LocalPath)
		deleted++
	}

	m.catalog.Wallpapers = append(kept, toKeep...)
	return deleted
}

func sanitizeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	res := b.String()
	if len(res) > 40 {
		res = res[:40]
	}
	return res
}
