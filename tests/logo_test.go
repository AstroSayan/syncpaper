package tests

import (
	"strings"
	"testing"

	"syncpaper/internal/logo"
	"syncpaper/internal/theme"
)

func TestLogoRendering(t *testing.T) {
	start := theme.RGB{R: 0, G: 255, B: 255}
	end := theme.RGB{R: 255, G: 0, B: 255}

	banner := logo.RenderBanner(start, end)
	if banner == "" {
		t.Errorf("expected non-empty rendered banner")
	}

	// Verify ANSI escape sequences are present
	if !strings.Contains(banner, "\033[38;2;") {
		t.Errorf("expected ANSI truecolor escape sequences in rendered banner")
	}

	lines := strings.Split(strings.TrimSpace(banner), "\n")
	if len(lines) < 5 {
		t.Errorf("expected at least 5 lines of ASCII art, got %d", len(lines))
	}
}

func TestLogoDefaultGradientAndPrintHeader(t *testing.T) {
	c1, c2 := logo.DefaultGradient()
	if c1.R != 0 || c2.R != 255 {
		t.Errorf("unexpected default gradient: %+v, %+v", c1, c2)
	}

	// Test PrintHeader with default gradient
	logo.PrintHeader("1.0.0", nil)

	// Test PrintHeader with custom palette
	pal := &theme.Palette{
		Accent:          theme.RGB{R: 100, G: 200, B: 250},
		AccentSecondary: theme.RGB{R: 250, G: 100, B: 200},
	}
	logo.PrintHeader("1.0.0", pal)
}

