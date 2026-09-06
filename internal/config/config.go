package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	General GeneralConfig `toml:"general"`
	Topics  TopicsConfig  `toml:"topics"`
	Sources SourcesConfig `toml:"sources"`
	Theming ThemingConfig `toml:"theming"`
}

type GeneralConfig struct {
	StorageDir       string   `toml:"storage_dir"`
	MaxCached        int      `toml:"max_cached"`
	RotationInterval string   `toml:"rotation_interval"`
	SyncInterval     string   `toml:"sync_interval"`
	DESetter         string   `toml:"de_setter"` // auto, hyprland, sway, gnome, kde, x11, custom
	CustomCommand    []string `toml:"custom_command,omitempty"`
}

type TopicsConfig struct {
	List          []string `toml:"list"`
	CountPerTopic int      `toml:"count_per_topic"`
}

type SourcesConfig struct {
	Wallhaven WallhavenConfig `toml:"wallhaven"`
	Bing      BingConfig      `toml:"bing"`
	NASA      NASAConfig      `toml:"nasa"`
	Reddit    RedditConfig    `toml:"reddit"`
	Unsplash  UnsplashConfig  `toml:"unsplash"`
}

type WallhavenConfig struct {
	Enabled     bool     `toml:"enabled"`
	APIKey      string   `toml:"api_key,omitempty"`
	Categories  string   `toml:"categories"` // 100/110/111 (General, Anime, People)
	Purity      string   `toml:"purity"`     // 100 (SFW), 110 (Sketchy), 111 (NSFW)
	Resolutions []string `toml:"resolutions"`
	Ratios      []string `toml:"ratios"`
	Sorting     string   `toml:"sorting"` // random, toplist, views
}

type BingConfig struct {
	Enabled bool   `toml:"enabled"`
	Market  string `toml:"market"`
}

type NASAConfig struct {
	Enabled bool   `toml:"enabled"`
	APIKey  string `toml:"api_key"`
}

type RedditConfig struct {
	Enabled    bool     `toml:"enabled"`
	Subreddits []string `toml:"subreddits"`
}

type UnsplashConfig struct {
	Enabled   bool   `toml:"enabled"`
	AccessKey string `toml:"access_key,omitempty"`
}

type ThemingConfig struct {
	Enabled         bool   `toml:"enabled"`
	Mode            string `toml:"mode"` // auto, dark, light
	ExportHyprland  bool   `toml:"export_hyprland"`
	ExportWaybar    bool   `toml:"export_waybar"`
	ExportKitty     bool   `toml:"export_kitty"`
	ExportAlacritty bool   `toml:"export_alacritty"`
	ExportFoot      bool   `toml:"export_foot"`
	ExportPywal     bool   `toml:"export_pywal"`
	UpdateGTKTheme  bool   `toml:"update_gtk_theme"`
	HookScript      string `toml:"hook_script"`
}

// ExpandPath expands ~ and environment variables in path
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				path = home
			} else {
				path = filepath.Join(home, path[2:])
			}
		}
	}
	return os.ExpandEnv(path)
}

// DefaultConfig returns safe and optimized defaults
func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			StorageDir:       "~/.local/share/syncpaper/wallpapers",
			MaxCached:        60,
			RotationInterval: "1h",
			SyncInterval:     "24h",
			DESetter:         "auto",
			CustomCommand:    []string{},
		},
		Topics: TopicsConfig{
			List: []string{
				"cyberpunk",
				"minimalist nature",
				"deep space",
				"dark aesthetic",
				"abstract architecture",
			},
			CountPerTopic: 4,
		},
		Sources: SourcesConfig{
			Wallhaven: WallhavenConfig{
				Enabled:     true,
				APIKey:      "",
				Categories:  "110",
				Purity:      "100", // SFW
				Resolutions: []string{"1920x1080", "2560x1440", "3840x2160"},
				Ratios:      []string{"16x9", "21x9", "16x10"},
				Sorting:     "random",
			},
			Bing: BingConfig{
				Enabled: true,
				Market:  "en-US",
			},
			NASA: NASAConfig{
				Enabled: true,
				APIKey:  "DEMO_KEY",
			},
			Reddit: RedditConfig{
				Enabled:    false,
				Subreddits: []string{"wallpapers", "EarthPorn", "spaceporn"},
			},
			Unsplash: UnsplashConfig{
				Enabled:   false,
				AccessKey: "",
			},
		},
		Theming: ThemingConfig{
			Enabled:         true,
			Mode:            "auto",
			ExportHyprland:  true,
			ExportWaybar:    true,
			ExportKitty:     true,
			ExportAlacritty: true,
			ExportFoot:      true,
			ExportPywal:     true,
			UpdateGTKTheme:  true,
			HookScript:      "~/.config/syncpaper/on_theme.sh",
		},
	}
}

// GetConfigDir returns ~/.config/syncpaper
func GetConfigDir() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".", ".config", "syncpaper")
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "syncpaper")
}

// DefaultConfigPath returns ~/.config/syncpaper/config.toml
func DefaultConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.toml")
}

// LoadConfig loads from the given path or returns default
func LoadConfig(customPath string) (*Config, error) {
	path := customPath
	if path == "" {
		path = DefaultConfigPath()
	}
	path = ExpandPath(path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := SaveConfig(cfg, path); err != nil {
			return cfg, nil // fallback in memory if save fails
		}
		return cfg, nil
	}

	cfg := DefaultConfig()
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	return cfg, nil
}

// SaveConfig writes the configuration to a file
func SaveConfig(cfg *Config, path string) error {
	path = ExpandPath(path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	var buf bytes.Buffer
	buf.WriteString("# syncpaper configuration\n# High-performance Linux wallpaper synchronizer and dynamic theme generator\n\n")

	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// ParseRotationInterval parses the rotation interval duration
func (c *Config) ParseRotationInterval() time.Duration {
	d, err := time.ParseDuration(c.General.RotationInterval)
	if err != nil || d < time.Minute {
		return time.Hour
	}
	return d
}

// ParseSyncInterval parses the sync interval duration
func (c *Config) ParseSyncInterval() time.Duration {
	d, err := time.ParseDuration(c.General.SyncInterval)
	if err != nil || d < time.Minute {
		return 24 * time.Hour
	}
	return d
}
