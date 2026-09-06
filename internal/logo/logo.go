package logo

import (
	"fmt"
	"strings"

	"syncpaper/internal/theme"
)

const RawBanner = `  ███████╗██╗   ██╗███╗   ██╗ ██████╗██████╗  █████╗ ██████╗ ███████╗██████╗ 
  ██╔════╝╚██╗ ██╔╝████╗  ██║██╔════╝██╔══██╗██╔══██╗██╔══██╗██╔════╝██╔══██╗
  ███████╗ ╚████╔╝ ██╔██╗ ██║██║     ██████╔╝███████║██████╔╝█████╗  ██████╔╝
  ╚════██║  ╚██╔╝  ██║╚██╗██║██║     ██╔═══╝ ██╔══██║██╔═══╝ ██╔══╝  ██╔══██╗
  ███████║   ██║   ██║ ╚████║╚██████╗██║     ██║  ██║██║     ███████╗██║  ██║
  ╚══════╝   ╚═╝   ╚═╝  ╚═══╝ ╚═════╝╚═╝     ╚═╝  ╚═╝╚═╝     ╚══════╝╚═╝  ╚═╝`

// DefaultGradient returns a cyberpunk cyan-to-magenta gradient
func DefaultGradient() (theme.RGB, theme.RGB) {
	return theme.RGB{R: 0, G: 229, B: 255}, theme.RGB{R: 255, G: 0, B: 127}
}

// RenderBanner renders the ASCII logo with horizontal truecolor gradient
func RenderBanner(start, end theme.RGB) string {
	lines := strings.Split(RawBanner, "\n")
	var result strings.Builder

	maxLen := 0
	for _, l := range lines {
		if len(l) > maxLen {
			maxLen = len(l)
		}
	}
	if maxLen == 0 {
		maxLen = 1
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		runes := []rune(line)
		for col, r := range runes {
			t := float64(col) / float64(maxLen)
			red := uint8(float64(start.R) + t*float64(int(end.R)-int(start.R)))
			green := uint8(float64(start.G) + t*float64(int(end.G)-int(start.G)))
			blue := uint8(float64(start.B) + t*float64(int(end.B)-int(start.B)))

			result.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm%c\033[0m", red, green, blue, r))
		}
		result.WriteString("\n")
	}

	return result.String()
}

// PrintHeader prints the stylized logo, subtitle, and version
func PrintHeader(version string, pal *theme.Palette) {
	start, end := DefaultGradient()
	if pal != nil {
		start = pal.Accent
		end = pal.AccentSecondary
	}

	fmt.Println(RenderBanner(start, end))
	fmt.Printf("   \033[1;37mLinux Wallpaper & Dynamic Theme Synchronizer\033[0m \033[90m•\033[0m \033[36mv%s\033[0m\n\n", version)
}
