package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"syncpaper/internal/config"
)

func getSystemdUserDir() (string, error) {
	configDir := config.GetConfigDir()
	userDir := filepath.Join(filepath.Dir(configDir), "systemd", "user")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create systemd user dir %s: %w", userDir, err)
	}
	return userDir, nil
}

func getBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// InstallSystemdUnits writes service and timer units to ~/.config/systemd/user/
func InstallSystemdUnits(cfg *config.Config) error {
	userDir, err := getSystemdUserDir()
	if err != nil {
		return err
	}

	bin, err := getBinaryPath()
	if err != nil {
		bin = "syncpaper"
	}

	// 1. syncpaper-sync.service
	syncService := fmt.Sprintf(`[Unit]
Description=Syncpaper Daily Wallpaper Fetch Service
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=%s sync

[Install]
WantedBy=default.target
`, bin)

	// 2. syncpaper-sync.timer
	syncTimer := fmt.Sprintf(`[Unit]
Description=Daily timer for syncpaper wallpaper download

[Timer]
OnCalendar=*-*-* 06:00:00
Persistent=true

[Install]
WantedBy=timers.target
`)

	// 3. syncpaper-rotate.service
	rotateService := fmt.Sprintf(`[Unit]
Description=Syncpaper Wallpaper & Theme Rotation Service
PartOf=graphical-session.target
After=graphical-session.target

[Service]
Type=oneshot
ExecStart=%s rotate

[Install]
WantedBy=graphical-session.target
`, bin)

	// 4. syncpaper-rotate.timer
	interval := cfg.General.RotationInterval
	if interval == "" {
		interval = "1h"
	}
	rotateTimer := fmt.Sprintf(`[Unit]
Description=Timer for syncpaper wallpaper rotation

[Timer]
OnBootSec=1m
OnUnitActiveSec=%s

[Install]
WantedBy=timers.target
`, interval)

	files := map[string]string{
		"syncpaper-sync.service":   syncService,
		"syncpaper-sync.timer":     syncTimer,
		"syncpaper-rotate.service": rotateService,
		"syncpaper-rotate.timer":   rotateTimer,
	}

	for name, content := range files {
		p := filepath.Join(userDir, name)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write unit %s: %w", p, err)
		}
	}

	// Reload systemd daemon
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}

// EnableSystemdUnits enables and starts the timers
func EnableSystemdUnits() error {
	cmd := exec.Command("systemctl", "--user", "enable", "--now", "syncpaper-sync.timer", "syncpaper-rotate.timer")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable timers (%v): %s", err, string(out))
	}
	return nil
}

// StatusSystemdUnits returns status of syncpaper user units
func StatusSystemdUnits() string {
	cmd := exec.Command("systemctl", "--user", "list-timers", "--all")
	out, err := cmd.Output()
	if err != nil {
		return "systemctl --user list-timers failed or not running"
	}

	var filtered []string
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		if strings.Contains(l, "syncpaper") || strings.Contains(l, "NEXT") {
			filtered = append(filtered, l)
		}
	}

	if len(filtered) <= 1 {
		return "No active syncpaper timers found. Run 'syncpaper service install' then 'syncpaper service enable'."
	}
	return strings.Join(filtered, "\n")
}

// UninstallSystemdUnits disables timers and removes files
func UninstallSystemdUnits() error {
	_ = exec.Command("systemctl", "--user", "disable", "--now", "syncpaper-sync.timer", "syncpaper-rotate.timer").Run()

	userDir, err := getSystemdUserDir()
	if err != nil {
		return err
	}

	files := []string{
		"syncpaper-sync.service",
		"syncpaper-sync.timer",
		"syncpaper-rotate.service",
		"syncpaper-rotate.timer",
	}

	for _, f := range files {
		_ = os.Remove(filepath.Join(userDir, f))
	}

	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}
