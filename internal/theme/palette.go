package theme

import (
	"fmt"
	"math"
)

// RGB represents an 8-bit RGB color
type RGB struct {
	R, G, B uint8
}

func (c RGB) Hex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func (c RGB) HyprlandHex() string {
	return fmt.Sprintf("rgb(%02x%02x%02x)", c.R, c.G, c.B)
}

func (c RGB) Luminance() float64 {
	// Relative luminance per ITU-R BT.709
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func (c RGB) Saturation() float64 {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min
	if max == 0 {
		return 0
	}
	return delta / max
}

// Lighten blends towards white by factor [0, 1]
func (c RGB) Lighten(factor float64) RGB {
	return RGB{
		R: uint8(float64(c.R) + (255.0-float64(c.R))*factor),
		G: uint8(float64(c.G) + (255.0-float64(c.G))*factor),
		B: uint8(float64(c.B) + (255.0-float64(c.B))*factor),
	}
}

// Darken blends towards black by factor [0, 1]
func (c RGB) Darken(factor float64) RGB {
	f := 1.0 - factor
	return RGB{
		R: uint8(float64(c.R) * f),
		G: uint8(float64(c.G) * f),
		B: uint8(float64(c.B) * f),
	}
}

// Palette holds the extracted dynamic theme colors
type Palette struct {
	IsDark          bool   `json:"is_dark"`
	Background      RGB    `json:"background"`
	Foreground      RGB    `json:"foreground"`
	Surface         RGB    `json:"surface"`
	Accent          RGB    `json:"accent"`
	AccentSecondary RGB    `json:"accent_secondary"`
	Cursor          RGB    `json:"cursor"`
	ActiveBorder    RGB    `json:"active_border"`
	InactiveBorder  RGB    `json:"inactive_border"`
	Colors          [16]RGB `json:"colors"` // ANSI 0-15
}

// PywalColorsSchema matches pywal ~/.cache/wal/colors.json format
type PywalColorsSchema struct {
	Wallpaper string            `json:"wallpaper"`
	Alpha     string            `json:"alpha"`
	Special   map[string]string `json:"special"`
	Colors    map[string]string `json:"colors"`
}

// ToPywalSchema converts Palette to standard pywal format
func (p *Palette) ToPywalSchema(wallpaperPath string) PywalColorsSchema {
	special := map[string]string{
		"background": p.Background.Hex(),
		"foreground": p.Foreground.Hex(),
		"cursor":     p.Cursor.Hex(),
		"accent":     p.Accent.Hex(),
	}

	colors := make(map[string]string)
	for i := 0; i < 16; i++ {
		colors[fmt.Sprintf("color%d", i)] = p.Colors[i].Hex()
	}

	return PywalColorsSchema{
		Wallpaper: wallpaperPath,
		Alpha:     "100",
		Special:   special,
		Colors:    colors,
	}
}
