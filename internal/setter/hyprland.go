package setter

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type HyprlandSetter struct{}

func NewHyprlandSetter() *HyprlandSetter {
	return &HyprlandSetter{}
}

func (h *HyprlandSetter) Name() string {
	return "hyprland"
}

func (h *HyprlandSetter) IsAvailable() bool {
	return hasCommand("hyprctl") || hasCommand("swaybg") || hasCommand("swww")
}

func (h *HyprlandSetter) Set(ctx context.Context, imagePath string) error {
	// 1. Check if swww is running
	if isProcessRunning("swww-daemon") && hasCommand("swww") {
		cmd := exec.CommandContext(ctx, "swww", "img", imagePath, "--transition-type", "wipe", "--transition-fps", "60")
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// 2. Check if hyprpaper is running
	if isProcessRunning("hyprpaper") && hasCommand("hyprctl") {
		_ = exec.CommandContext(ctx, "hyprctl", "hyprpaper", "preload", imagePath).Run()
		cmd := exec.CommandContext(ctx, "hyprctl", "hyprpaper", "wallpaper", fmt.Sprintf(",%s", imagePath))
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// 3. Fallback to swaybg
	if hasCommand("swaybg") {
		oldPIDs := getProcessPIDs("swaybg")

		cmd := exec.Command("swaybg", "-i", imagePath, "-m", "fill")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start swaybg: %w", err)
		}

		// Wait briefly so the new wallpaper displays smoothly before killing the old one
		time.Sleep(100 * time.Millisecond)
		for _, pid := range oldPIDs {
			_ = exec.Command("kill", "-9", strconv.Itoa(pid)).Run()
		}

		return nil
	}

	return fmt.Errorf("no Hyprland wallpaper daemon found (install swaybg, swww, or hyprpaper)")
}

func isProcessRunning(name string) bool {
	out, err := exec.Command("pgrep", "-x", name).Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

func getProcessPIDs(name string) []int {
	out, err := exec.Command("pgrep", "-x", name).Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var pids []int
	for _, l := range lines {
		if pid, err := strconv.Atoi(strings.TrimSpace(l)); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}
