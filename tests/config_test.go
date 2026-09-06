package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"syncpaper/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.General.MaxCached != 60 {
		t.Errorf("expected MaxCached=60, got %d", cfg.General.MaxCached)
	}
	if len(cfg.Topics.List) == 0 {
		t.Errorf("expected default topics list not to be empty")
	}
	if !cfg.Sources.Wallhaven.Enabled {
		t.Errorf("expected Wallhaven to be enabled by default")
	}
	if !cfg.Sources.Bing.Enabled {
		t.Errorf("expected Bing to be enabled by default")
	}
	if !cfg.Theming.Enabled {
		t.Errorf("expected theming to be enabled by default")
	}

	interval := cfg.ParseRotationInterval()
	if interval != time.Hour {
		t.Errorf("expected rotation interval 1h, got %v", interval)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	expanded := config.ExpandPath("~/Pictures")
	expected := filepath.Join(home, "Pictures")
	if expanded != expected {
		t.Errorf("expected %s, got %s", expected, expanded)
	}

	raw := "/tmp/test"
	if config.ExpandPath(raw) != raw {
		t.Errorf("expected %s, got %s", raw, config.ExpandPath(raw))
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "syncpaper_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, "config.toml")
	cfg := config.DefaultConfig()
	cfg.General.MaxCached = 123
	cfg.Topics.List = []string{"space", "cyberpunk"}

	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.General.MaxCached != 123 {
		t.Errorf("expected MaxCached=123, got %d", loaded.General.MaxCached)
	}
	if len(loaded.Topics.List) != 2 || loaded.Topics.List[0] != "space" {
		t.Errorf("topics not preserved: %v", loaded.Topics.List)
	}
}
