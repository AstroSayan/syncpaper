package gui

import (
	"fmt"
	"os/exec"
)

// LaunchAppWindow opens the GUI in desktop application mode or default browser
func LaunchAppWindow(targetURL string) error {
	browsers := []string{"chromium", "brave", "google-chrome-stable", "google-chrome"}

	for _, b := range browsers {
		if path, err := exec.LookPath(b); err == nil {
			cmd := exec.Command(path,
				fmt.Sprintf("--app=%s", targetURL),
				"--user-data-dir=/tmp/syncpaper-gui-profile",
				"--class=syncpaper",
				"--name=syncpaper",
			)
			if err := cmd.Start(); err == nil {
				return nil
			}
		}
	}

	// Fallback to xdg-open
	if path, err := exec.LookPath("xdg-open"); err == nil {
		return exec.Command(path, targetURL).Start()
	}

	return fmt.Errorf("no suitable browser or xdg-open found to launch GUI")
}
