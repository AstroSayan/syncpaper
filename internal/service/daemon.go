package service

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"syncpaper/internal/config"
)

type DaemonRunner struct {
	cfg      *config.Config
	syncFn   func() error
	rotateFn func() error
}

func NewDaemonRunner(cfg *config.Config, syncFn func() error, rotateFn func() error) *DaemonRunner {
	return &DaemonRunner{
		cfg:      cfg,
		syncFn:   syncFn,
		rotateFn: rotateFn,
	}
}

func (d *DaemonRunner) Run(ctx context.Context) error {
	rotInterval := d.cfg.ParseRotationInterval()
	syncInterval := d.cfg.ParseSyncInterval()

	log.Printf("[syncpaper] Starting daemon (rotate: every %v, sync: every %v)", rotInterval, syncInterval)

	rotTicker := time.NewTicker(rotInterval)
	defer rotTicker.Stop()

	syncTicker := time.NewTicker(syncInterval)
	defer syncTicker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Perform initial rotation on startup
	if d.rotateFn != nil {
		log.Println("[syncpaper] Performing initial rotation...")
		if err := d.rotateFn(); err != nil {
			log.Printf("[syncpaper] Initial rotation error: %v", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("[syncpaper] Daemon shutting down...")
			return nil
		case sig := <-sigChan:
			log.Printf("[syncpaper] Received signal %v, exiting...", sig)
			return nil
		case <-rotTicker.C:
			log.Println("[syncpaper] Timer triggered: rotating wallpaper and theme...")
			if d.rotateFn != nil {
				if err := d.rotateFn(); err != nil {
					log.Printf("[syncpaper] Rotation failed: %v", err)
				}
			}
		case <-syncTicker.C:
			log.Println("[syncpaper] Timer triggered: syncing wallpapers from online sources...")
			if d.syncFn != nil {
				if err := d.syncFn(); err != nil {
					log.Printf("[syncpaper] Sync failed: %v", err)
				}
			}
		}
	}
}
