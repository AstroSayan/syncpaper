package tests

import (
	"context"
	"testing"

	"syncpaper/internal/config"
	"syncpaper/internal/setter"
)

func TestSetterDetectionExplicit(t *testing.T) {
	cases := []struct {
		deSetting string
		wantName  string
	}{
		{"hyprland", "hyprland"},
		{"sway", "wayland"},
		{"wayland", "wayland"},
		{"gnome", "gnome"},
		{"kde", "kde"},
		{"plasma", "kde"},
		{"x11", "x11"},
		{"feh", "x11"},
	}

	for _, tc := range cases {
		cfg := config.DefaultConfig()
		cfg.General.DESetter = tc.deSetting

		s, err := setter.DetectSetter(cfg)
		if err != nil {
			t.Errorf("DetectSetter(%q) unexpected error: %v", tc.deSetting, err)
			continue
		}
		if s.Name() != tc.wantName {
			t.Errorf("DetectSetter(%q) got name %q, want %q", tc.deSetting, s.Name(), tc.wantName)
		}
		_ = s.IsAvailable()
	}
}

func TestCustomSetter(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.General.DESetter = "custom"
	cfg.General.CustomCommand = []string{"echo", "{path}"}

	s, err := setter.DetectSetter(cfg)
	if err != nil {
		t.Fatalf("DetectSetter(custom) failed: %v", err)
	}

	if s.Name() != "custom" {
		t.Errorf("expected name custom, got %s", s.Name())
	}
	if !s.IsAvailable() {
		t.Errorf("expected custom setter with echo command to be available")
	}

	ctx := context.Background()
	if err := s.Set(ctx, "/tmp/sample_wallpaper.jpg"); err != nil {
		t.Errorf("custom.Set failed: %v", err)
	}

	// Empty custom command error case
	cfgEmpty := config.DefaultConfig()
	cfgEmpty.General.DESetter = "custom"
	cfgEmpty.General.CustomCommand = nil
	_, errEmpty := setter.DetectSetter(cfgEmpty)
	if errEmpty == nil {
		t.Errorf("expected error for empty custom command, got nil")
	}
}

func TestSetterAutoDetection(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.General.DESetter = ""

	// 1. Test Hyprland signature
	t.Run("HyprlandSig", func(t *testing.T) {
		t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "test_sig_123")
		s, err := setter.DetectSetter(cfg)
		if err != nil {
			t.Fatalf("expected hyprland detected, got error: %v", err)
		}
		if s.Name() != "hyprland" {
			t.Errorf("expected hyprland, got %s", s.Name())
		}
	})

	// 2. Test GNOME desktop
	t.Run("GNOMEDesktop", func(t *testing.T) {
		t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
		t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
		s, err := setter.DetectSetter(cfg)
		if err != nil {
			t.Fatalf("expected gnome detected, got error: %v", err)
		}
		if s.Name() != "gnome" {
			t.Errorf("expected gnome, got %s", s.Name())
		}
	})

	// 3. Test KDE desktop
	t.Run("KDEDesktop", func(t *testing.T) {
		t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
		t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
		s, err := setter.DetectSetter(cfg)
		if err != nil {
			t.Fatalf("expected kde detected, got error: %v", err)
		}
		if s.Name() != "kde" {
			t.Errorf("expected kde, got %s", s.Name())
		}
	})

	// 4. Test Sway desktop
	t.Run("SwayDesktop", func(t *testing.T) {
		t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
		t.Setenv("XDG_CURRENT_DESKTOP", "sway")
		s, err := setter.DetectSetter(cfg)
		if err != nil {
			t.Fatalf("expected sway/wayland detected, got error: %v", err)
		}
		if s.Name() != "wayland" {
			t.Errorf("expected wayland, got %s", s.Name())
		}
	})

	// 5. Direct constructors
	_ = setter.NewGNOMESetter().Name()
	_ = setter.NewKDESetter().Name()
	_ = setter.NewWaylandSetter().Name()
	_ = setter.NewX11Setter().Name()
	_ = setter.NewHyprlandSetter().Name()
}
