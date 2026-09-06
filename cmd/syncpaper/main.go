package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/webp"

	"syncpaper/internal/cache"
	"syncpaper/internal/config"
	"syncpaper/internal/gui"
	"syncpaper/internal/logo"
	"syncpaper/internal/service"
	"syncpaper/internal/setter"
	"syncpaper/internal/source"
	"syncpaper/internal/theme"
)

const version = "1.0.0"

func printUsage() {
	logo.PrintHeader(version, nil)
	fmt.Printf(`Usage:
  syncpaper [command] [options]

Commands:
  gui                Launch modern desktop GUI application
  sync               Pull new wallpapers from configured sources and topics
  rotate             Rotate wallpaper and synchronize device theme
  current            Display active wallpaper and theme palette
  set <path|id>      Apply a specific image and synchronize theme
  list               List cached wallpapers
  theme [image]      Extract color palette and refresh themes without rotating
  favorite [id]      Pin current or specified wallpaper to protect from deletion
  blacklist [id]     Remove wallpaper and permanently block its hash
  daemon             Run background daemon for automatic sync and rotation
  service <action>   Manage systemd user timers (install, enable, status, uninstall)
  config <action>    Manage configuration (path, show, init)
  version            Print version information

Options for 'sync':
  --topic <name>     Fetch wallpapers for a specific topic only
  --count <num>      Number of wallpapers per topic (default from config)

Options for 'rotate':
  --topic <name>     Rotate only to wallpapers matching this topic
  --random           Pick a random wallpaper instead of sequential
  --prev             Rotate to previous wallpaper in history

Options for 'gui':
  --port <num>       Port for local GUI server (default 0 for auto-assign)
  --no-browser       Do not automatically open desktop browser window

Run 'syncpaper <command> --help' for command-specific flags.
`)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	cat, err := cache.LoadCatalog("")
	if err != nil {
		log.Fatalf("Catalog error: %v", err)
	}

	mgr, err := cache.NewManager(cfg, cat)
	if err != nil {
		log.Fatalf("Cache manager error: %v", err)
	}

	ctx := context.Background()

	switch cmd {
	case "gui":
		runGUI(cfg, cat, mgr, args)
	case "sync":
		runSync(ctx, cfg, mgr, args)
	case "rotate":
		runRotate(ctx, cfg, cat, args)
	case "current":
		runCurrent(cat)
	case "set":
		runSet(ctx, cfg, cat, args)
	case "list":
		runList(cat, args)
	case "theme":
		runTheme(cfg, cat, args)
	case "favorite", "fav":
		runFavorite(cat, args)
	case "blacklist":
		runBlacklist(ctx, cfg, cat, args)
	case "daemon":
		runDaemon(ctx, cfg, cat, mgr)
	case "service":
		runService(cfg, args)
	case "config":
		runConfig(args)
	case "version", "--version", "-v":
		fmt.Printf("syncpaper v%s (linux/amd64)\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func runGUI(cfg *config.Config, cat *cache.Catalog, mgr *cache.Manager, args []string) {
	fs := flag.NewFlagSet("gui", flag.ExitOnError)
	portFlag := fs.Int("port", 0, "Port for GUI server (default 0 for auto-assign)")
	noBrowser := fs.Bool("no-browser", false, "Do not automatically launch desktop browser window")
	_ = fs.Parse(args)

	srv, err := gui.NewServer(cfg, cat, mgr, *portFlag)
	if err != nil {
		log.Fatalf("Failed to initialize GUI server: %v", err)
	}

	logo.PrintHeader(version, nil)
	fmt.Printf("🚀 Starting syncpaper GUI at \033[1;36m%s\033[0m\n", srv.URL())

	if !*noBrowser {
		if err := gui.LaunchAppWindow(srv.URL()); err != nil {
			fmt.Printf("⚠️  Could not launch desktop window automatically: %v\n", err)
			fmt.Printf("👉 Please open %s in your browser\n", srv.URL())
		}
	}

	fmt.Println("Press Ctrl+C to stop the GUI server.")
	if err := srv.Start(); err != nil {
		log.Fatalf("GUI server error: %v", err)
	}
}

func runSync(ctx context.Context, cfg *config.Config, mgr *cache.Manager, args []string) {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	topicFlag := fs.String("topic", "", "Fetch wallpapers for a specific topic only")
	countFlag := fs.Int("count", 0, "Override count per topic")
	_ = fs.Parse(args)

	fmt.Println("🚀 Querying online sources for high-resolution wallpapers...")
	items, errs := source.FetchWallpapers(ctx, cfg, *topicFlag, *countFlag)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Printf("  ⚠️  Source notice: %v\n", e)
		}
	}

	if len(items) == 0 {
		fmt.Println("❌ No wallpapers found from active sources.")
		return
	}

	fmt.Printf("📥 Found %d wallpapers. Downloading to local cache...\n", len(items))
	downloaded, dlErrors := mgr.DownloadAll(ctx, items, 4)
	if len(dlErrors) > 0 {
		for _, e := range dlErrors {
			fmt.Printf("  ⚠️  Download error: %v\n", e)
		}
	}

	pruned := mgr.Prune()
	fmt.Printf("✅ Sync complete: %d new wallpapers cached (%d duplicate/cached skipped, %d pruned)\n",
		len(downloaded), len(items)-len(downloaded), pruned)
}

func runRotate(ctx context.Context, cfg *config.Config, cat *cache.Catalog, args []string) {
	fs := flag.NewFlagSet("rotate", flag.ExitOnError)
	topicFlag := fs.String("topic", "", "Rotate only among wallpapers with this topic")
	randomFlag := fs.Bool("random", false, "Pick a random wallpaper")
	prevFlag := fs.Bool("prev", false, "Pick the previous wallpaper from history")
	_ = fs.Parse(args)

	if len(cat.Wallpapers) == 0 {
		fmt.Println("❌ Local wallpaper cache is empty. Run 'syncpaper sync' first to download wallpapers.")
		os.Exit(1)
	}

	// Filter available wallpapers
	var pool []cache.CachedWallpaper
	for _, w := range cat.Wallpapers {
		if w.Blacklisted {
			continue
		}
		if *topicFlag != "" && !strings.EqualFold(w.Topic, *topicFlag) {
			continue
		}
		// Verify file exists on disk
		if _, err := os.Stat(w.LocalPath); err == nil {
			pool = append(pool, w)
		}
	}

	if len(pool) == 0 {
		fmt.Println("❌ No available wallpapers match the criteria on disk.")
		os.Exit(1)
	}

	var target *cache.CachedWallpaper

	if *prevFlag && len(cat.History) > 1 {
		// Second in history is previous
		prevID := cat.History[1]
		target = cat.GetByID(prevID)
	}

	if target == nil {
		if *randomFlag || len(pool) <= 2 {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			idx := r.Intn(len(pool))
			target = &pool[idx]
		} else {
			// Find first in pool not recently used in history
			historySet := make(map[string]bool)
			for i, h := range cat.History {
				if i < 5 { // Avoid last 5
					historySet[h] = true
				}
			}

			for i := range pool {
				if pool[i].ID != cat.CurrentID && !historySet[pool[i].ID] {
					target = &pool[i]
					break
				}
			}

			// Fallback if all were in recent history
			if target == nil {
				for i := range pool {
					if pool[i].ID != cat.CurrentID {
						target = &pool[i]
						break
					}
				}
			}
			if target == nil {
				target = &pool[0]
			}
		}
	}

	if err := applyWallpaperAndTheme(ctx, cfg, cat, target); err != nil {
		log.Fatalf("Failed to apply wallpaper and theme: %v", err)
	}
}

func applyWallpaperAndTheme(ctx context.Context, cfg *config.Config, cat *cache.Catalog, w *cache.CachedWallpaper) error {
	st, err := setter.DetectSetter(cfg)
	if err != nil {
		return fmt.Errorf("setter error: %w", err)
	}

	fmt.Printf("🎨 Setting wallpaper using [%s]: %s\n", st.Name(), w.LocalPath)
	if err := st.Set(ctx, w.LocalPath); err != nil {
		return fmt.Errorf("failed to set wallpaper: %w", err)
	}

	cat.SetCurrent(w.ID)
	_ = cat.Save()

	if cfg.Theming.Enabled {
		fmt.Println("✨ Extracting color palette and updating device theme...")
		pal, err := theme.ExtractPalette(w.LocalPath, cfg.Theming.Mode)
		if err != nil {
			return fmt.Errorf("theme extraction failed: %w", err)
		}

		if err := theme.ExportTheme(cfg, pal, w.LocalPath); err != nil {
			return fmt.Errorf("theme export failed: %w", err)
		}

		printPalettePreview(pal, w)
	}

	return nil
}

func printPalettePreview(p *theme.Palette, w *cache.CachedWallpaper) {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("🖼️  Wallpaper: %s\n", w.Title)
	fmt.Printf("📌 Topic:     %s (%s)\n", w.Topic, w.Source)
	fmt.Printf("📐 Dim:       %dx%d\n", w.Width, w.Height)
	fmt.Printf("🌓 Mode:      %s\n\n", map[bool]string{true: "Dark", false: "Light"}[p.IsDark])

	fmt.Println("Color Palette:")
	printColorBlock("Background", p.Background)
	printColorBlock("Surface   ", p.Surface)
	printColorBlock("Accent    ", p.Accent)
	printColorBlock("Accent 2  ", p.AccentSecondary)
	printColorBlock("Border    ", p.ActiveBorder)
	printColorBlock("Foreground", p.Foreground)

	fmt.Print("\nANSI: ")
	for i := 0; i < 8; i++ {
		c := p.Colors[i]
		fmt.Printf("\033[48;2;%d;%d;%dm  \033[0m", c.R, c.G, c.B)
	}
	fmt.Print("\n      ")
	for i := 8; i < 16; i++ {
		c := p.Colors[i]
		fmt.Printf("\033[48;2;%d;%d;%dm  \033[0m", c.R, c.G, c.B)
	}
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

func printColorBlock(name string, c theme.RGB) {
	fmt.Printf("  %s: \033[48;2;%d;%d;%dm    \033[0m %s\n", name, c.R, c.G, c.B, c.Hex())
}

func runCurrent(cat *cache.Catalog) {
	current := cat.GetCurrent()
	if current == nil {
		fmt.Println("No current wallpaper tracked. Run 'syncpaper rotate' or 'syncpaper set <path>'.")
		return
	}

	pal, err := theme.ExtractPalette(current.LocalPath, "auto")
	if err != nil {
		fmt.Printf("Wallpaper: %s (Path: %s)\n", current.Title, current.LocalPath)
		return
	}

	logo.PrintHeader(version, pal)
	printPalettePreview(pal, current)
}

func runSet(ctx context.Context, cfg *config.Config, cat *cache.Catalog, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: syncpaper set <path-or-id>")
		os.Exit(1)
	}

	target := args[0]
	// Check if target is an ID in catalog
	item := cat.GetByID(target)
	if item != nil {
		if err := applyWallpaperAndTheme(ctx, cfg, cat, item); err != nil {
			log.Fatalf("Failed to apply wallpaper: %v", err)
		}
		return
	}

	// Check if target is a file path
	absPath, err := filepath.Abs(config.ExpandPath(target))
	if err != nil || os.IsNotExist(err) {
		log.Fatalf("Target not found as catalog ID or local file: %s", target)
	}

	width, height := 0, 0
	if f, err := os.Open(absPath); err == nil {
		if imgCfg, _, err := image.DecodeConfig(f); err == nil {
			width = imgCfg.Width
			height = imgCfg.Height
		}
		f.Close()
	}

	customItem := &cache.CachedWallpaper{
		ID:           filepath.Base(absPath),
		Source:       "local",
		Topic:        "custom",
		Title:        filepath.Base(absPath),
		LocalPath:    absPath,
		Width:        width,
		Height:       height,
		DownloadedAt: time.Now(),
		Favorite:     true,
	}
	cat.AddWallpaper(*customItem)

	if err := applyWallpaperAndTheme(ctx, cfg, cat, customItem); err != nil {
		log.Fatalf("Failed to apply wallpaper: %v", err)
	}
}

func runList(cat *cache.Catalog, args []string) {
	if len(cat.Wallpapers) == 0 {
		fmt.Println("No wallpapers cached. Run 'syncpaper sync' first.")
		return
	}

	fmt.Printf("%-24s %-16s %-10s %-12s %-4s %s\n", "ID", "TOPIC", "SOURCE", "DIMENSIONS", "FAV", "TITLE")
	fmt.Println(strings.Repeat("─", 80))

	for _, w := range cat.Wallpapers {
		fav := " "
		if w.Favorite {
			fav = "★"
		}
		if w.ID == cat.CurrentID {
			fav += "▶"
		}
		title := w.Title
		if len(title) > 28 {
			title = title[:25] + "..."
		}
		fmt.Printf("%-24s %-16s %-10s %-12s %-4s %s\n",
			w.ID, w.Topic, w.Source, fmt.Sprintf("%dx%d", w.Width, w.Height), fav, title)
	}
	fmt.Printf("\nTotal: %d wallpapers cached\n", len(cat.Wallpapers))
}

func runFavorite(cat *cache.Catalog, args []string) {
	targetID := cat.CurrentID
	if len(args) > 0 {
		targetID = args[0]
	}
	if targetID == "" {
		fmt.Println("No wallpaper specified or currently active.")
		return
	}

	fav, err := cat.ToggleFavorite(targetID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	_ = cat.Save()

	if fav {
		fmt.Printf("★ Pinned %s as favorite (will not be pruned).\n", targetID)
	} else {
		fmt.Printf("☆ Removed %s from favorites.\n", targetID)
	}
}

func runBlacklist(ctx context.Context, cfg *config.Config, cat *cache.Catalog, args []string) {
	targetID := cat.CurrentID
	if len(args) > 0 {
		targetID = args[0]
	}
	if targetID == "" {
		fmt.Println("No wallpaper specified or currently active.")
		return
	}

	if err := cat.Blacklist(targetID); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	_ = cat.Save()

	fmt.Printf("🚫 Blacklisted and removed %s.\n", targetID)
	// If current was blacklisted, rotate immediately to next
	if len(cat.Wallpapers) > 0 {
		runRotate(ctx, cfg, cat, nil)
	}
}

func runTheme(cfg *config.Config, cat *cache.Catalog, args []string) {
	var imagePath string
	var title string

	if len(args) > 0 {
		imagePath = config.ExpandPath(args[0])
		title = filepath.Base(imagePath)
	} else {
		curr := cat.GetCurrent()
		if curr == nil {
			fmt.Println("No active wallpaper. Provide an image path: syncpaper theme <path>")
			return
		}
		imagePath = curr.LocalPath
		title = curr.Title
	}

	pal, err := theme.ExtractPalette(imagePath, cfg.Theming.Mode)
	if err != nil {
		log.Fatalf("Palette extraction failed: %v", err)
	}

	if err := theme.ExportTheme(cfg, pal, imagePath); err != nil {
		log.Fatalf("Theme export failed: %v", err)
	}

	mockWallpaper := &cache.CachedWallpaper{
		Title:     title,
		Topic:     "manual",
		Source:    "file",
		LocalPath: imagePath,
	}
	printPalettePreview(pal, mockWallpaper)
	fmt.Printf("Theme files generated in %s\n", theme.GetCacheDir())
}

func runDaemon(ctx context.Context, cfg *config.Config, cat *cache.Catalog, mgr *cache.Manager) {
	syncFn := func() error {
		items, _ := source.FetchWallpapers(ctx, cfg, "", 0)
		if len(items) > 0 {
			_, _ = mgr.DownloadAll(ctx, items, 4)
		}
		return nil
	}

	rotateFn := func() error {
		runRotate(ctx, cfg, cat, nil)
		return nil
	}

	d := service.NewDaemonRunner(cfg, syncFn, rotateFn)
	if err := d.Run(ctx); err != nil {
		log.Fatalf("Daemon error: %v", err)
	}
}

func runService(cfg *config.Config, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: syncpaper service <install|enable|status|uninstall>")
		return
	}

	action := args[0]
	switch action {
	case "install":
		if err := service.InstallSystemdUnits(cfg); err != nil {
			log.Fatalf("Installation failed: %v", err)
		}
		fmt.Println("✅ Installed systemd user units to ~/.config/systemd/user/")
		fmt.Println("👉 To activate, run: syncpaper service enable")
	case "enable":
		if err := service.EnableSystemdUnits(); err != nil {
			log.Fatalf("Enable failed: %v", err)
		}
		fmt.Println("✅ Enabled and started syncpaper user timers:")
		fmt.Println("   - syncpaper-sync.timer   (Daily refresh)")
		fmt.Println("   - syncpaper-rotate.timer (Hourly wallpaper & theme rotation)")
	case "status":
		fmt.Println(service.StatusSystemdUnits())
	case "uninstall":
		if err := service.UninstallSystemdUnits(); err != nil {
			log.Fatalf("Uninstall failed: %v", err)
		}
		fmt.Println("✅ Disabled and removed syncpaper systemd user units.")
	default:
		fmt.Printf("Unknown service action: %s. Use install, enable, status, or uninstall.\n", action)
	}
}

func runConfig(args []string) {
	if len(args) == 0 || args[0] == "path" {
		fmt.Println(config.DefaultConfigPath())
		return
	}

	switch args[0] {
	case "show":
		p := config.DefaultConfigPath()
		data, err := os.ReadFile(p)
		if err != nil {
			log.Fatalf("Failed to read config: %v", err)
		}
		fmt.Println(string(data))
	case "init":
		p := config.DefaultConfigPath()
		cfg := config.DefaultConfig()
		if err := config.SaveConfig(cfg, p); err != nil {
			log.Fatalf("Failed to init config: %v", err)
		}
		fmt.Printf("✅ Initialized default configuration at %s\n", p)
	default:
		fmt.Printf("Unknown config action: %s. Use path, show, or init.\n", args[0])
	}
}
