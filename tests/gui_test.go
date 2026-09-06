package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
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
}
