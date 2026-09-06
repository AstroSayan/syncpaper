package gui

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"syncpaper/internal/cache"
	"syncpaper/internal/config"
	"syncpaper/internal/setter"
	"syncpaper/internal/source"
	"syncpaper/internal/theme"
)

type Server struct {
	cfg        *config.Config
	cat        *cache.Catalog
	mgr        *cache.Manager
	httpServer *http.Server
	listener   net.Listener
	port       int
}

func NewServer(cfg *config.Config, cat *cache.Catalog, mgr *cache.Manager, port int) (*Server, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	actualPort := ln.Addr().(*net.TCPAddr).Port

	s := &Server{
		cfg:      cfg,
		cat:      cat,
		mgr:      mgr,
		listener: ln,
		port:     actualPort,
	}

	mux := http.NewServeMux()

	// Static web assets
	webFS, err := GetFileSystem()
	if err != nil {
		return nil, fmt.Errorf("failed to get embedded web fs: %w", err)
	}
	fileServer := http.FileServer(webFS)
	mux.Handle("/", fileServer)

	// API Endpoints
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/wallpapers", s.handleWallpapers)
	mux.HandleFunc("/api/wallpaper/image", s.handleWallpaperImage)
	mux.HandleFunc("/api/rotate", s.handleRotate)
	mux.HandleFunc("/api/set", s.handleSet)
	mux.HandleFunc("/api/sync", s.handleSync)
	mux.HandleFunc("/api/favorite", s.handleFavorite)
	mux.HandleFunc("/api/blacklist", s.handleBlacklist)
	mux.HandleFunc("/api/config", s.handleConfig)

	s.httpServer = &http.Server{
		Handler: mux,
	}

	return s, nil
}

func (s *Server) Port() int {
	return s.port
}

func (s *Server) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", s.port)
}

func (s *Server) Start() error {
	return s.httpServer.Serve(s.listener)
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	curr := s.cat.GetCurrent()

	var palData interface{}
	if curr != nil {
		pal, err := theme.ExtractPalette(curr.LocalPath, s.cfg.Theming.Mode)
		if err == nil {
			var ansiHex [16]string
			for i := 0; i < 16; i++ {
				ansiHex[i] = pal.Colors[i].Hex()
			}
			palData = map[string]interface{}{
				"is_dark":          pal.IsDark,
				"background":       pal.Background.Hex(),
				"foreground":       pal.Foreground.Hex(),
				"surface":          pal.Surface.Hex(),
				"accent":           pal.Accent.Hex(),
				"accent_secondary": pal.AccentSecondary.Hex(),
				"active_border":    pal.ActiveBorder.Hex(),
				"inactive_border":  pal.InactiveBorder.Hex(),
				"colors":           ansiHex,
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"current": curr,
		"palette": palData,
	})
}

func (s *Server) handleWallpapers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"wallpapers": s.cat.Wallpapers,
	})
}

func (s *Server) handleWallpaperImage(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	path := r.URL.Query().Get("path")

	var targetPath string
	if id != "" {
		item := s.cat.GetByID(id)
		if item != nil {
			targetPath = item.LocalPath
		}
	} else if path != "" {
		targetPath = path
	}

	if targetPath == "" {
		http.NotFound(w, r)
		return
	}

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	ext := strings.ToLower(filepath.Ext(targetPath))
	switch ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	http.ServeFile(w, r, targetPath)
}

func (s *Server) handleRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(s.cat.Wallpapers) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "no wallpapers in cache",
		})
		return
	}

	// Pick next wallpaper avoiding recent history
	var target *cache.CachedWallpaper
	for i := range s.cat.Wallpapers {
		cand := &s.cat.Wallpapers[i]
		if cand.Blacklisted {
			continue
		}
		if cand.ID != s.cat.CurrentID {
			target = cand
			break
		}
	}
	if target == nil {
		target = &s.cat.Wallpapers[0]
	}

	st, err := setter.DetectSetter(s.cfg)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if err := st.Set(r.Context(), target.LocalPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	s.cat.SetCurrent(target.ID)
	_ = s.cat.Save()

	pal, _ := theme.ExtractPalette(target.LocalPath, s.cfg.Theming.Mode)
	if pal != nil {
		_ = theme.ExportTheme(s.cfg, pal, target.LocalPath)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"current": target,
	})
}

func (s *Server) handleSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var target *cache.CachedWallpaper
	if req.ID != "" {
		target = s.cat.GetByID(req.ID)
	}

	if target == nil && req.Path != "" {
		target = &cache.CachedWallpaper{
			ID:        filepath.Base(req.Path),
			LocalPath: req.Path,
			Title:     filepath.Base(req.Path),
			Topic:     "manual",
		}
	}

	if target == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"success": false,
			"error":   "wallpaper not found",
		})
		return
	}

	st, err := setter.DetectSetter(s.cfg)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if err := st.Set(r.Context(), target.LocalPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	s.cat.SetCurrent(target.ID)
	_ = s.cat.Save()

	pal, _ := theme.ExtractPalette(target.LocalPath, s.cfg.Theming.Mode)
	if pal != nil {
		_ = theme.ExportTheme(s.cfg, pal, target.LocalPath)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"current": target,
	})
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	items, errs := source.FetchWallpapers(r.Context(), s.cfg, "", 0)
	var errStrs []string
	for _, e := range errs {
		errStrs = append(errStrs, e.Error())
	}

	downloaded, dlErrors := s.mgr.DownloadAll(r.Context(), items, 4)
	for _, e := range dlErrors {
		errStrs = append(errStrs, e.Error())
	}

	pruned := s.mgr.Prune()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"downloaded": len(downloaded),
		"pruned":     pruned,
		"error":      strings.Join(errStrs, "; "),
	})
}

func (s *Server) handleFavorite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	fav, err := s.cat.ToggleFavorite(req.ID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	_ = s.cat.Save()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"favorite": fav,
	})
}

func (s *Server) handleBlacklist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.cat.Blacklist(req.ID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	_ = s.cat.Save()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, s.cfg)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Topics struct {
				List          []string `json:"list"`
				CountPerTopic int      `json:"count_per_topic"`
			} `json:"topics"`
			Sources struct {
				Wallhaven struct{ Enabled bool `json:"enabled"` } `json:"wallhaven"`
				Bing      struct{ Enabled bool `json:"enabled"` } `json:"bing"`
				NASA      struct{ Enabled bool `json:"enabled"` } `json:"nasa"`
				Reddit    struct{ Enabled bool `json:"enabled"` } `json:"reddit"`
			} `json:"sources"`
			General struct {
				RotationInterval string `json:"rotation_interval"`
				MaxCached        int    `json:"max_cached"`
				DESetter         string `json:"de_setter"`
			} `json:"general"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if len(req.Topics.List) > 0 {
			s.cfg.Topics.List = req.Topics.List
		}
		if req.Topics.CountPerTopic > 0 {
			s.cfg.Topics.CountPerTopic = req.Topics.CountPerTopic
		}
		s.cfg.Sources.Wallhaven.Enabled = req.Sources.Wallhaven.Enabled
		s.cfg.Sources.Bing.Enabled = req.Sources.Bing.Enabled
		s.cfg.Sources.NASA.Enabled = req.Sources.NASA.Enabled
		s.cfg.Sources.Reddit.Enabled = req.Sources.Reddit.Enabled

		if req.General.RotationInterval != "" {
			s.cfg.General.RotationInterval = req.General.RotationInterval
		}
		if req.General.MaxCached > 0 {
			s.cfg.General.MaxCached = req.General.MaxCached
		}
		if req.General.DESetter != "" {
			s.cfg.General.DESetter = req.General.DESetter
		}

		_ = config.SaveConfig(s.cfg, config.DefaultConfigPath())

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"config":  s.cfg,
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func parseQueryInt(val string, fallback int) int {
	if n, err := strconv.Atoi(val); err == nil {
		return n
	}
	return fallback
}
