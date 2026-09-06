package gui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// LaunchAppWindow opens the GUI in desktop application mode or default browser
func LaunchAppWindow(targetURL string) error {
	browsers := []string{"chromium", "brave", "google-chrome-stable", "google-chrome"}

	profileDir := filepath.Join(os.TempDir(), "syncpaper-gui-profile")
	_ = os.MkdirAll(profileDir, 0755)

	// Create "First Run" sentinel to suppress any welcome/first-run wizard
	_ = os.WriteFile(filepath.Join(profileDir, "First Run"), []byte{}, 0644)

	args := []string{
		fmt.Sprintf("--app=%s", targetURL),
		fmt.Sprintf("--user-data-dir=%s", profileDir),
		"--class=syncpaper",
		"--name=syncpaper",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-search-engine-choice-screen",
		"--disable-first-run-ui",
		"--disable-default-apps",
		"--disable-features=Translate,OptimizationHints,MediaRouter",
		"--disable-sync",
		"--password-store=basic",
	}

	for _, b := range browsers {
		if path, err := exec.LookPath(b); err == nil {
			cmd := exec.Command(path, args...)
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
