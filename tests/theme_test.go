package tests

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"syncpaper/internal/config"
	"syncpaper/internal/theme"
)

func TestColorMath(t *testing.T) {
	c := theme.RGB{R: 255, G: 0, B: 0}
	if c.Hex() != "#ff0000" {
		t.Errorf("expected #ff0000, got %s", c.Hex())
	}
	if c.HyprlandHex() != "rgb(ff0000)" {
		t.Errorf("expected rgb(ff0000), got %s", c.HyprlandHex())
	}
	if c.Saturation() != 1.0 {
		t.Errorf("expected saturation 1.0, got %f", c.Saturation())
	}

	dark := c.Darken(0.5)
	if dark.R > 130 || dark.R < 120 {
		t.Errorf("expected darkened R ~127, got %d", dark.R)
	}

	white := theme.RGB{R: 255, G: 255, B: 255}
	if white.Luminance() < 0.99 {
		t.Errorf("expected white luminance ~1.0, got %f", white.Luminance())
	}

	black := theme.RGB{R: 0, G: 0, B: 0}
	if black.Luminance() > 0.01 {
		t.Errorf("expected black luminance ~0.0, got %f", black.Luminance())
	}
}

func TestExtractPaletteDarkImage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "syncpaper_theme_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	imgPath := filepath.Join(tmpDir, "test_dark.png")
	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	// Create a dark image with a vibrant blue accent
	width, height := 100, 100
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x < 20 && y < 20 {
				img.Set(x, y, color.RGBA{R: 0, G: 120, B: 255, A: 255}) // Vibrant blue patch
			} else {
				img.Set(x, y, color.RGBA{R: 15, G: 18, B: 25, A: 255})  // Deep dark base
			}
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode png: %v", err)
	}
	f.Close()

	pal, err := theme.ExtractPalette(imgPath, "auto")
	if err != nil {
		t.Fatalf("failed to extract palette: %v", err)
	}

	if !pal.IsDark {
		t.Errorf("expected dark palette for predominantly dark image")
	}

	if pal.Background.Luminance() > 0.25 {
		t.Errorf("expected dark background, got luminance %f (%s)", pal.Background.Luminance(), pal.Background.Hex())
	}

	// Foreground should be light and high contrast
	if pal.Foreground.Luminance() < 0.7 {
		t.Errorf("expected light foreground for dark mode, got %f (%s)", pal.Foreground.Luminance(), pal.Foreground.Hex())
	}

	// Pywal schema check
	schema := pal.ToPywalSchema(imgPath)
	if schema.Wallpaper != imgPath {
		t.Errorf("schema wallpaper mismatch")
	}
	if len(schema.Colors) != 16 {
		t.Errorf("expected 16 colors in pywal schema, got %d", len(schema.Colors))
	}
}

func TestExtractPaletteLightImage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "syncpaper_theme_light_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	imgPath := filepath.Join(tmpDir, "test_light.png")
	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	width, height := 80, 80
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x < 15 && y < 15 {
				img.Set(x, y, color.RGBA{R: 220, G: 20, B: 60, A: 255}) // Crimson patch
			} else {
				img.Set(x, y, color.RGBA{R: 245, G: 245, B: 248, A: 255}) // Bright light base
			}
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode png: %v", err)
	}
	f.Close()

	pal, err := theme.ExtractPalette(imgPath, "light")
	if err != nil {
		t.Fatalf("failed to extract palette: %v", err)
	}

	if pal.IsDark {
		t.Errorf("expected light palette for predominantly white image with light mode")
	}
	if pal.Background.Luminance() < 0.7 {
		t.Errorf("expected bright background, got %f", pal.Background.Luminance())
	}
}

func TestThemeExporter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "syncpaper_exporter_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldCache := os.Getenv("XDG_CACHE_HOME")
	defer os.Setenv("XDG_CACHE_HOME", oldCache)
	os.Setenv("XDG_CACHE_HOME", tmpDir)

	cfg := config.DefaultConfig()
	cfg.Theming.ExportPywal = true
	cfg.Theming.ExportHyprland = true
	cfg.Theming.ExportWaybar = true
	cfg.Theming.ExportKitty = true
	cfg.Theming.ExportAlacritty = true
	cfg.Theming.ExportFoot = true
	pal := &theme.Palette{
		Background:      theme.RGB{R: 16, G: 18, B: 24},
		Foreground:      theme.RGB{R: 240, G: 242, B: 248},
		Accent:          theme.RGB{R: 0, G: 229, B: 255},
		AccentSecondary: theme.RGB{R: 255, G: 0, B: 128},
		Surface:         theme.RGB{R: 30, G: 34, B: 46},
		IsDark:          true,
	}
	for i := 0; i < 16; i++ {
		pal.Colors[i] = theme.RGB{R: uint8(i * 15), G: uint8(i * 15), B: uint8(i * 15)}
	}

	if err := theme.ExportTheme(cfg, pal, "/tmp/test.jpg"); err != nil {
		t.Fatalf("ExportTheme failed: %v", err)
	}

	cacheDir := theme.GetCacheDir()
	expectedFiles := []string{
		"colors.json",
		"colors.sh",
		"hyprland-colors.conf",
		"waybar-colors.css",
		"kitty-colors.conf",
		"alacritty-colors.toml",
		"foot-colors.ini",
	}

	for _, ef := range expectedFiles {
		p := filepath.Join(cacheDir, ef)
		data, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("expected exported file %s to exist: %v", ef, err)
		}
		if len(data) == 0 {
			t.Errorf("expected file %s to have content", ef)
		}
	}

	walFile := filepath.Join(tmpDir, "wal", "colors.json")
	if _, err := os.Stat(walFile); err != nil {
		t.Errorf("expected ~/.cache/wal/colors.json to be created: %v", err)
	}
}
