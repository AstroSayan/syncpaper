package setter

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

type WaylandSetter struct{}

func NewWaylandSetter() *WaylandSetter {
	return &WaylandSetter{}
}

func (w *WaylandSetter) Name() string {
	return "wayland"
}

func (w *WaylandSetter) IsAvailable() bool {
	return hasCommand("swaybg") || hasCommand("swww")
}

func (w *WaylandSetter) Set(ctx context.Context, imagePath string) error {
	if isProcessRunning("swww-daemon") && hasCommand("swww") {
		cmd := exec.CommandContext(ctx, "swww", "img", imagePath, "--transition-type", "wipe")
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

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
		time.Sleep(100 * time.Millisecond)
		for _, pid := range oldPIDs {
			_ = exec.Command("kill", "-9", strconv.Itoa(pid)).Run()
		}
		return nil
	}

	return fmt.Errorf("neither swaybg nor swww found in PATH")
}
