# syncpaper 🖼️🎨

<p align="center">
  <a href="https://github.com/AstroSayan/syncpaper/releases"><img src="https://img.shields.io/github/v/release/AstroSayan/syncpaper?style=for-the-badge&logo=github&color=3b82f6&logoColor=white" alt="Release Version"></a>
  <a href="https://github.com/AstroSayan/syncpaper/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/AstroSayan/syncpaper/ci.yml?branch=main&style=for-the-badge&logo=githubactions&logoColor=white&label=CI%20Build" alt="CI Build Status"></a>
  <a href="https://github.com/AstroSayan/syncpaper/actions"><img src="https://img.shields.io/badge/Tests-Passing-10b981?style=for-the-badge&logo=go&logoColor=white" alt="Tests Status"></a>
  <a href="https://github.com/AstroSayan/syncpaper/actions"><img src="https://img.shields.io/badge/Coverage-36%25-06b6d4?style=for-the-badge&logo=codecov&logoColor=white" alt="Code Coverage"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/AstroSayan/syncpaper?style=for-the-badge&logo=go&logoColor=white&color=00ADD8" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-8b5cf6?style=for-the-badge" alt="License: MIT"></a>
</p>

<p align="center">
  <strong>High-performance, zero-bloat Linux wallpaper synchronizer and dynamic desktop theming utility with embedded glassmorphism GUI.</strong>
</p>

---

## ✨ Features

- **Blazing Performance**: Single static Go binary (~8.5MB stripped), instant startup (<2ms), sub-10ms color extraction, and tiny memory footprint (<8MB RAM in daemon mode).
- **Multi-Source Online Fetching**:
  - **Wallhaven**: Query search API with topics, resolution filters (1080p, 1440p, 4K UHD), aspect ratios, and purity filters.
  - **Bing Daily**: Curated high-resolution daily landscape and photography with captions.
  - **NASA APOD**: Astronomy Picture of the Day for celestial and space aesthetics.
  - **Reddit**: Subreddits like `r/wallpapers`, `r/EarthPorn`, `r/spaceporn` (configurable).
  - **Unsplash**: Direct high-res search (optional API key).
- **Intelligent Local Cache & Storage**:
  - SHA-256 content deduplication prevents duplicate downloads across sources.
  - Retention pruning automatically purges oldest unpinned wallpapers when cache limit is exceeded.
  - Pin favorites (`syncpaper favorite`) to protect them from pruning forever.
  - Blacklist unwanted wallpapers (`syncpaper blacklist`) to remove them and block their hash.
- **Universal Desktop Environment (DE) & WM Support**:
  - **Hyprland**: Native support via `swaybg`, `hyprpaper`, or `swww` with smooth zero-flicker transitions.
  - **Sway / Wayland wlroots**: Manages `swaybg` or `swww`.
  - **GNOME**: Toggles `gsettings` background URIs (light and dark).
  - **KDE Plasma 5 & 6**: DBus evaluation script on `org.kde.plasmashell`.
  - **X11**: Automatic fallback to `feh` or `nitrogen`.
  - **Custom**: User-defined command templates (`setter = "custom"`).
- **Dynamic Device Theming Engine**:
  - Fast K-means color quantization in pure Go (no external C/Python dependencies).
  - Dominant color, vibrant accent, secondary accent, surface, and high-contrast foreground text (WCAG AA compliant).
  - Automatic dark/light mode detection based on wallpaper luminance.
  - Generates theme configurations into `~/.cache/syncpaper/`:
    - `colors.json`: 100% pywal / wallust / matugen compatible format.
    - `colors.sh`: Shell script environment variables (`$BACKGROUND`, `$ACCENT`, `$COLOR0`..`$COLOR15`).
    - `hyprland-colors.conf`: Borders and accent variables for Hyprland.
    - `waybar-colors.css`: CSS variables for Waybar (`@define-color accent ...;`).
    - `kitty-colors.conf`: Kitty terminal colors (hot-reloads dynamically).
    - `alacritty-colors.toml`: Alacritty terminal theme.
    - `foot-colors.ini`: Foot terminal theme.
  - Live desktop reload: Dispatches `hyprctl reload`, `pkill -SIGUSR2 waybar`, GTK interface color scheme preference, and custom hooks.
  - Custom hook: Dispatches `~/.config/syncpaper/on_theme.sh <wallpaper_path> <colors_json_path>` after each rotation.
- **Automation Out-of-the-Box**:
  - **Systemd User Timers**: One-command installation (`syncpaper service install` + `syncpaper service enable`) for daily wallpaper sync and hourly rotation.
  - **Standalone Daemon**: Built-in background daemon (`syncpaper daemon`) for simple autostart in `hyprland.conf` or window manager scripts.

---

## 🚀 Installation

### Option 1: Pre-built Static Binaries (Recommended)

Download the latest pre-compiled, self-contained binary for your Linux architecture from [GitHub Releases](https://github.com/AstroSayan/syncpaper/releases/latest):

```bash
# For x86_64 (amd64)
curl -sL https://github.com/AstroSayan/syncpaper/releases/latest/download/syncpaper-linux-amd64.tar.gz | tar -xz
sudo install -m 755 syncpaper-linux-amd64/syncpaper /usr/local/bin/syncpaper

# For ARM64 (aarch64 / Raspberry Pi)
curl -sL https://github.com/AstroSayan/syncpaper/releases/latest/download/syncpaper-linux-arm64.tar.gz | tar -xz
sudo install -m 755 syncpaper-linux-arm64/syncpaper /usr/local/bin/syncpaper
```

### Option 2: Building from Source

Ensure Go (>= 1.22) is installed:

```bash
git clone https://github.com/AstroSayan/syncpaper.git
cd syncpaper
make build
sudo make install
```

This installs the binary to `/usr/local/bin/syncpaper`.

---

## 📖 Quick Start

### 1. Initialize Configuration
```bash
syncpaper config init
```
This generates `~/.config/syncpaper/config.toml`.

### 2. Fetch Wallpapers
```bash
# Pull wallpapers across your configured topics and sources
syncpaper sync

# Or fetch wallpapers for a specific topic
syncpaper sync --topic "cyberpunk" --count 4
```

### 3. Rotate Wallpaper and Apply Theme
```bash
syncpaper rotate
```
This updates your desktop wallpaper, extracts the color palette, emits all config files, triggers live component reloads, and displays an ANSI color preview in your terminal.

### 4. Enable Daily Refresh & Periodic Rotation
```bash
syncpaper service install
syncpaper service enable
```
Check timer status anytime:
```bash
syncpaper service status
```

---

## 🛠️ Command Reference

| Command | Description |
| :--- | :--- |
| `syncpaper gui [--port <p>] [--no-browser]` | Launch modern desktop GUI application window |
| `syncpaper sync [--topic <t>] [--count <n>]` | Pull wallpapers from configured online sources into cache |
| `syncpaper rotate [--topic <t>] [--random] [--prev]` | Rotate to the next wallpaper and update device theme |
| `syncpaper current` | Display details and color palette of currently set wallpaper |
| `syncpaper set <path\|id>` | Set a specific wallpaper by catalog ID or local file path |
| `syncpaper list` | List cached wallpapers, topics, dimensions, and favorite status |
| `syncpaper favorite [id]` | Pin current or specified wallpaper as favorite (prevents pruning) |
| `syncpaper blacklist [id]` | Remove wallpaper and permanently block its image hash |
| `syncpaper theme [image-path]` | Extract color palette and update themes without changing wallpaper |
| `syncpaper daemon` | Run standalone daemon with internal tickers for rotation and sync |
| `syncpaper service <install\|enable\|status\|uninstall>` | Manage systemd user services and timers |
| `syncpaper config <path\|show\|init>` | View or initialize configuration file |
| `syncpaper version` | Print version information |

---

## ⚙️ Configuration (`~/.config/syncpaper/config.toml`)

```toml
[general]
  # Directory where wallpapers are cached
  storage_dir = "~/.local/share/syncpaper/wallpapers"
  # Maximum number of wallpapers to retain (favorites are never pruned)
  max_cached = 60
  # Rotation interval (e.g. 30m, 1h, 2h)
  rotation_interval = "1h"
  # Online synchronization interval
  sync_interval = "24h"
  # Desktop setter: auto | hyprland | sway | gnome | kde | x11 | custom
  de_setter = "auto"
  # Optional custom command template:
  # custom_command = ["swaybg", "-i", "{path}", "-m", "fill"]

[topics]
  # Topics to search online
  list = [
    "cyberpunk",
    "minimalist nature",
    "deep space",
    "dark aesthetic",
    "abstract architecture"
  ]
  # Number of wallpapers to fetch per topic
  count_per_topic = 4

[sources]
  [sources.wallhaven]
    enabled = true
    categories = "110" # General, Anime, People
    purity = "100"     # SFW
    resolutions = ["1920x1080", "2560x1440", "3840x2160"]
    ratios = ["16x9", "21x9", "16x10"]
    sorting = "random" # random | toplist | views
    # api_key = "optional_api_key_here"

  [sources.bing]
    enabled = true
    market = "en-US"

  [sources.nasa]
    enabled = true
    api_key = "DEMO_KEY"

  [sources.reddit]
    enabled = false
    subreddits = ["wallpapers", "EarthPorn", "spaceporn"]

  [sources.unsplash]
    enabled = false
    # access_key = "your_unsplash_access_key"

[theming]
  enabled = true
  # Mode: auto (adapts to wallpaper luminance) | dark | light
  mode = "auto"
  export_hyprland = true
  export_waybar = true
  export_kitty = true
  export_alacritty = true
  export_foot = true
  export_pywal = true
  update_gtk_theme = true
  hook_script = "~/.config/syncpaper/on_theme.sh"
```

---

## 🖥️ Desktop Integration Examples

### Hyprland (`~/.config/hypr/hyprland.conf`)
Include generated border and accent colors:
```ini
source = ~/.cache/syncpaper/hyprland-colors.conf

general {
    col.active_border = $active_border_1 $active_border_2 45deg
    col.inactive_border = $inactive_border
}

# Autostart daemon (if not using systemd timers):
# exec-once = syncpaper daemon
```

### Waybar (`~/.config/waybar/style.css`)
Include generated CSS variables:
```css
@import url("../../.cache/syncpaper/waybar-colors.css");

window#waybar {
    background-color: @background;
    color: @foreground;
    border-bottom: 2px solid @accent;
}

#workspaces button.active {
    background-color: @accent;
    color: @background;
}
```

### Kitty Terminal (`~/.config/kitty/kitty.conf`)
Include generated colors:
```ini
include ~/.cache/syncpaper/kitty-colors.conf
```

### Custom Shell Hook (`~/.config/syncpaper/on_theme.sh`)
```bash
#!/usr/bin/env bash
WALLPAPER="$1"
COLORS_JSON="$2"

# Example: notify user with Dunst / SwayNC
notify-send "Wallpaper & Theme Updated" "New theme applied from $(basename "$WALLPAPER")" -i "$WALLPAPER"
```

---

## 🧪 Running Tests

```bash
make test
```

Unit tests cover configuration serialization, path expansion, API parsing, color math, K-means clustering, WCAG contrast verification, catalog operations, and retention pruning.

---

## 📜 License

MIT License. Designed for high performance and clean desktop aesthetics.
