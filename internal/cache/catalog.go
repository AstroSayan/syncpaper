package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"syncpaper/internal/config"
)

// CachedWallpaper contains metadata about an image stored in the local cache
type CachedWallpaper struct {
	ID           string    `json:"id"`
	Source       string    `json:"source"`
	Topic        string    `json:"topic"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	LocalPath    string    `json:"local_path"`
	Hash         string    `json:"hash"` // SHA-256
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	SizeBytes    int64     `json:"size_bytes"`
	DownloadedAt time.Time `json:"downloaded_at"`
	LastUsedAt   time.Time `json:"last_used_at"`
	Favorite     bool      `json:"favorite"`
	Blacklisted  bool      `json:"blacklisted"`
}

// Catalog holds the collection of cached wallpapers and rotation history
type Catalog struct {
	CurrentID   string            `json:"current_id"`
	History     []string          `json:"history"` // Last N shown IDs
	Wallpapers  []CachedWallpaper `json:"wallpapers"`
	Blacklisted []string          `json:"blacklisted_hashes"`

	filePath string
	mu       sync.Mutex
}

// GetCatalogPath returns ~/.local/share/syncpaper/catalog.json
func GetCatalogPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".", ".local", "share", "syncpaper", "catalog.json")
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "syncpaper", "catalog.json")
}

// LoadCatalog loads catalog from disk or initializes a new one
func LoadCatalog(customPath string) (*Catalog, error) {
	path := customPath
	if path == "" {
		path = GetCatalogPath()
	}
	path = config.ExpandPath(path)

	c := &Catalog{
		History:     []string{},
		Wallpapers:  []CachedWallpaper{},
		Blacklisted: []string{},
		filePath:    path,
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return c, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog %s: %w", path, err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("failed to parse catalog JSON: %w", err)
	}
	c.filePath = path
	return c, nil
}

// Save writes catalog to disk atomically
func (c *Catalog) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for catalog: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode catalog: %w", err)
	}

	tmpPath := c.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write tmp catalog: %w", err)
	}

	return os.Rename(tmpPath, c.filePath)
}

// HasHash checks if a file hash is already in the catalog
func (c *Catalog) HasHash(hash string) bool {
	if hash == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, b := range c.Blacklisted {
		if b == hash {
			return true
		}
	}
	for _, w := range c.Wallpapers {
		if w.Hash == hash {
			return true
		}
	}
	return false
}

// AddWallpaper adds a new wallpaper entry
func (c *Catalog) AddWallpaper(w CachedWallpaper) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing if present
	for i, item := range c.Wallpapers {
		if item.ID == w.ID || (w.Hash != "" && item.Hash == w.Hash) {
			c.Wallpapers[i] = w
			return
		}
	}
	c.Wallpapers = append(c.Wallpapers, w)
}

// GetByID finds wallpaper by ID
func (c *Catalog) GetByID(id string) *CachedWallpaper {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.Wallpapers {
		if c.Wallpapers[i].ID == id {
			return &c.Wallpapers[i]
		}
	}
	return nil
}

// GetCurrent returns the currently set wallpaper
func (c *Catalog) GetCurrent() *CachedWallpaper {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.CurrentID == "" {
		return nil
	}
	for i := range c.Wallpapers {
		if c.Wallpapers[i].ID == c.CurrentID {
			return &c.Wallpapers[i]
		}
	}
	return nil
}

// SetCurrent marks the current wallpaper and records history
func (c *Catalog) SetCurrent(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.CurrentID = id
	for i := range c.Wallpapers {
		if c.Wallpapers[i].ID == id {
			c.Wallpapers[i].LastUsedAt = time.Now()
			break
		}
	}

	// Keep history up to last 20
	c.History = append([]string{id}, c.History...)
	if len(c.History) > 20 {
		c.History = c.History[:20]
	}
}

// ToggleFavorite toggles the favorite status of a wallpaper
func (c *Catalog) ToggleFavorite(id string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.Wallpapers {
		if c.Wallpapers[i].ID == id {
			c.Wallpapers[i].Favorite = !c.Wallpapers[i].Favorite
			return c.Wallpapers[i].Favorite, nil
		}
	}
	return false, fmt.Errorf("wallpaper with id %s not found", id)
}

// Blacklist adds wallpaper to blacklist and removes from catalog
func (c *Catalog) Blacklist(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var newWallpapers []CachedWallpaper
	var found *CachedWallpaper
	for _, w := range c.Wallpapers {
		if w.ID == id {
			wCopy := w
			found = &wCopy
		} else {
			newWallpapers = append(newWallpapers, w)
		}
	}

	if found == nil {
		return fmt.Errorf("wallpaper with id %s not found", id)
	}

	c.Wallpapers = newWallpapers
	c.Blacklisted = append(c.Blacklisted, found.Hash)
	if c.CurrentID == id {
		c.CurrentID = ""
	}

	// Remove file if exists
	_ = os.Remove(found.LocalPath)
	return nil
}
