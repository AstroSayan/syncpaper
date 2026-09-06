package setter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"syncpaper/internal/config"
)

// Setter is the interface that sets wallpaper for a specific Desktop Environment or Window Manager
type Setter interface {
	Name() string
	Set(ctx context.Context, imagePath string) error
	IsAvailable() bool
}

// DetectSetter determines the appropriate wallpaper setter for the running environment
func DetectSetter(cfg *config.Config) (Setter, error) {
	configured := strings.ToLower(strings.TrimSpace(cfg.General.DESetter))

	// Check explicit configuration first
	switch configured {
	case "hyprland":
		return NewHyprlandSetter(), nil
	case "sway", "wayland":
		return NewWaylandSetter(), nil
	case "gnome":
		return NewGNOMESetter(), nil
	case "kde", "plasma":
		return NewKDESetter(), nil
	case "x11", "feh", "nitrogen":
		return NewX11Setter(), nil
	case "custom":
		if len(cfg.General.CustomCommand) > 0 {
			return NewCustomSetter(cfg.General.CustomCommand), nil
		}
		return nil, fmt.Errorf("custom setter specified but custom_command is empty")
	}

	// Auto-detection
	de := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	session := strings.ToLower(os.Getenv("DESKTOP_SESSION"))
	hyprSig := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")

	if hyprSig != "" || strings.Contains(de, "hyprland") || strings.Contains(session, "hyprland") {
		return NewHyprlandSetter(), nil
	}

	if strings.Contains(de, "gnome") || strings.Contains(session, "gnome") {
		return NewGNOMESetter(), nil
	}

	if strings.Contains(de, "kde") || strings.Contains(session, "plasma") {
		return NewKDESetter(), nil
	}

	if strings.Contains(de, "sway") || os.Getenv("SWAYSOCK") != "" {
		return NewWaylandSetter(), nil
	}

	// Wayland generic
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if hasCommand("swaybg") || hasCommand("swww") {
			return NewWaylandSetter(), nil
		}
	}

	// X11 fallback
	if os.Getenv("DISPLAY") != "" {
		if hasCommand("feh") || hasCommand("nitrogen") {
			return NewX11Setter(), nil
		}
	}

	// If swaybg is installed, default to it
	if hasCommand("swaybg") {
		return NewWaylandSetter(), nil
	}

	return nil, fmt.Errorf("could not auto-detect desktop environment setter. Please specify 'de_setter' in ~/.config/syncpaper/config.toml")
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
