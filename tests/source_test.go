package tests

import (
	"testing"

	"syncpaper/internal/config"
	"syncpaper/internal/source"
)

func TestSourceRegistry(t *testing.T) {
	cfg := config.DefaultConfig()

	sources := source.GetEnabledSources(cfg)
	if len(sources) < 2 {
		t.Errorf("expected at least 2 enabled sources by default, got %d", len(sources))
	}

	names := make(map[string]bool)
	for _, s := range sources {
		names[s.Name()] = true
	}

	if !names["wallhaven"] {
		t.Errorf("expected wallhaven to be in enabled sources")
	}
	if !names["bing"] {
		t.Errorf("expected bing to be in enabled sources")
	}

	// Disable wallhaven
	cfg.Sources.Wallhaven.Enabled = false
	sourcesAfter := source.GetEnabledSources(cfg)
	for _, s := range sourcesAfter {
		if s.Name() == "wallhaven" {
			t.Errorf("wallhaven should be disabled")
		}
	}
}
