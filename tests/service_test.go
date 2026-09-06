package tests

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"syncpaper/internal/config"
	"syncpaper/internal/service"
)

func TestDaemonRunnerLifecycle(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.General.RotationInterval = "100ms"
	cfg.General.SyncInterval = "100ms"

	var rotated atomic.Bool
	var synced atomic.Bool

	rotateFn := func() error {
		rotated.Store(true)
		return nil
	}

	syncFn := func() error {
		synced.Store(true)
		return nil
	}

	d := service.NewDaemonRunner(cfg, syncFn, rotateFn)
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	if err := d.Run(ctx); err != nil {
		t.Fatalf("daemon Run failed: %v", err)
	}

	if !rotated.Load() {
		t.Errorf("expected initial rotation to be executed in daemon")
	}
}

func TestInstallSystemdUnits(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "syncpaper_systemd_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldConfig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldConfig)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := config.DefaultConfig()
	if err := service.InstallSystemdUnits(cfg); err != nil {
		t.Fatalf("InstallSystemdUnits failed: %v", err)
	}

	userDir := filepath.Join(tmpDir, "systemd", "user")
	expectedFiles := []string{
		"syncpaper-sync.service",
		"syncpaper-sync.timer",
		"syncpaper-rotate.service",
		"syncpaper-rotate.timer",
	}

	for _, ef := range expectedFiles {
		p := filepath.Join(userDir, ef)
		data, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("expected unit file %s to exist: %v", ef, err)
		}
		if len(data) == 0 {
			t.Errorf("unit file %s was empty", ef)
		}
	}
}
