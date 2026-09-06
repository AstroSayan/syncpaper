package theme

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"math/rand"
	"os"
	"sort"

	_ "golang.org/x/image/webp"
)

type Cluster struct {
	Center RGB
	Count  int
}

// ExtractPalette extracts a balanced color palette and theme from an image file
func ExtractPalette(imagePath string, mode string) (*Palette, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid image bounds")
	}

	// Subsample to 80x80 max grid for sub-10ms performance
	sampleStepX := int(math.Max(1, float64(w)/80.0))
	sampleStepY := int(math.Max(1, float64(h)/80.0))

	var pixels []RGB
	var totalLum float64

	for y := bounds.Min.Y; y < bounds.Max.Y; y += sampleStepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += sampleStepX {
			r, g, b, _ := img.At(x, y).RGBA()
			c := RGB{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
			}
			pixels = append(pixels, c)
			totalLum += c.Luminance()
		}
	}

	if len(pixels) == 0 {
		return nil, fmt.Errorf("no pixels sampled")
	}

	avgLuminance := totalLum / float64(len(pixels))
	isDark := avgLuminance < 0.55
	if mode == "dark" {
		isDark = true
	} else if mode == "light" {
		isDark = false
	}

	// Run K-means clustering (k=8)
	clusters := kMeans(pixels, 8, 8)

	// Sort clusters by count (dominance)
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Count > clusters[j].Count
	})

	// Find most vibrant accent color
	accent := findBestAccent(clusters, isDark)
	accentSecondary := findSecondaryAccent(clusters, accent, isDark)

	// Determine background & surface
	var bg, surface, fg, cursor RGB
	if isDark {
		// Dark background: deep tint of dominant or darkest cluster
		darkest := findDarkest(clusters)
		// Ensure background is suitably dark (luminance <= 0.08)
		if darkest.Luminance() > 0.08 {
			bg = darkest.Darken(0.7)
		} else {
			bg = darkest
		}
		// Clamp minimum darkness for clean OLED/dark rice
		if bg.R > 35 {
			bg.R = 20
		}
		if bg.G > 35 {
			bg.G = 22
		}
		if bg.B > 45 {
			bg.B = 30
		}

		surface = bg.Lighten(0.12)
		fg = RGB{R: 232, G: 235, B: 240} // Crisp soft white
		cursor = accent
	} else {
		// Light background
		lightest := findLightest(clusters)
		if lightest.Luminance() < 0.88 {
			bg = lightest.Lighten(0.7)
		} else {
			bg = lightest
		}
		surface = bg.Darken(0.08)
		fg = RGB{R: 30, G: 32, B: 42} // Deep slate
		cursor = accent
	}

	activeBorder := accent
	inactiveBorder := surface.Lighten(0.1)
	if !isDark {
		inactiveBorder = surface.Darken(0.15)
	}

	ansiColors := generateANSI16(bg, fg, accent, clusters, isDark)

	return &Palette{
		IsDark:          isDark,
		Background:      bg,
		Foreground:      fg,
		Surface:         surface,
		Accent:          accent,
		AccentSecondary: accentSecondary,
		Cursor:          cursor,
		ActiveBorder:    activeBorder,
		InactiveBorder:  inactiveBorder,
		Colors:          ansiColors,
	}, nil
}

func kMeans(pixels []RGB, k int, iterations int) []Cluster {
	if len(pixels) == 0 {
		return nil
	}
	if k > len(pixels) {
		k = len(pixels)
	}

	// Deterministic seed based on sample count for reproducible theme per image
	r := rand.New(rand.NewSource(int64(len(pixels))))

	// Initialize centers with k-means++-like spread
	centers := make([]RGB, k)
	centers[0] = pixels[r.Intn(len(pixels))]

	for i := 1; i < k; i++ {
		var maxDist float64
		bestIdx := 0
		for step := 0; step < 20; step++ {
			p := pixels[r.Intn(len(pixels))]
			minDist := math.MaxFloat64
			for j := 0; j < i; j++ {
				d := colorDistanceSq(p, centers[j])
				if d < minDist {
					minDist = d
				}
			}
			if minDist > maxDist {
				maxDist = minDist
				bestIdx = step
				centers[i] = p
			}
		}
		if maxDist == 0 {
			centers[i] = pixels[r.Intn(len(pixels))]
		}
		_ = bestIdx
	}

	clusters := make([]Cluster, k)

	for iter := 0; iter < iterations; iter++ {
		counts := make([]int, k)
		sumR := make([]uint64, k)
		sumG := make([]uint64, k)
		sumB := make([]uint64, k)

		for _, p := range pixels {
			bestCluster := 0
			minDist := math.MaxFloat64
			for i := 0; i < k; i++ {
				d := colorDistanceSq(p, centers[i])
				if d < minDist {
					minDist = d
					bestCluster = i
				}
			}
			counts[bestCluster]++
			sumR[bestCluster] += uint64(p.R)
			sumG[bestCluster] += uint64(p.G)
			sumB[bestCluster] += uint64(p.B)
		}

		for i := 0; i < k; i++ {
			if counts[i] > 0 {
				centers[i] = RGB{
					R: uint8(sumR[i] / uint64(counts[i])),
					G: uint8(sumG[i] / uint64(counts[i])),
					B: uint8(sumB[i] / uint64(counts[i])),
				}
			}
		}
	}

	for i := 0; i < k; i++ {
		clusters[i] = Cluster{
			Center: centers[i],
			Count:  0,
		}
	}

	for _, p := range pixels {
		bestCluster := 0
		minDist := math.MaxFloat64
		for i := 0; i < k; i++ {
			d := colorDistanceSq(p, centers[i])
			if d < minDist {
				minDist = d
				bestCluster = i
			}
		}
		clusters[bestCluster].Count++
	}

	return clusters
}

func colorDistanceSq(c1, c2 RGB) float64 {
	dr := float64(c1.R) - float64(c2.R)
	dg := float64(c1.G) - float64(c2.G)
	db := float64(c1.B) - float64(c2.B)
	return dr*dr + dg*dg + db*db
}

func findBestAccent(clusters []Cluster, isDark bool) RGB {
	var best RGB
	var highestScore float64 = -1.0

	for _, cl := range clusters {
		c := cl.Center
		sat := c.Saturation()
		lum := c.Luminance()

		// Score prioritizes rich saturation and readable luminance
		lumTarget := 0.55
		if !isDark {
			lumTarget = 0.40
		}
		lumDiff := math.Abs(lum - lumTarget)
		score := sat*1.5 + (1.0 - lumDiff)

		if score > highestScore {
			highestScore = score
			best = c
		}
	}

	if highestScore < 0.4 {
		// Fallback vibrant cyan/blue if image is near monochrome
		if isDark {
			return RGB{R: 114, G: 135, B: 253}
		}
		return RGB{R: 30, G: 102, B: 245}
	}

	return best
}

func findSecondaryAccent(clusters []Cluster, primary RGB, isDark bool) RGB {
	var best RGB
	var highestScore float64 = -1.0

	for _, cl := range clusters {
		c := cl.Center
		dist := colorDistanceSq(c, primary)
		if dist < 1200 { // too close to primary
			continue
		}

		sat := c.Saturation()
		lum := c.Luminance()
		score := sat + (1.0 - math.Abs(lum-0.5))

		if score > highestScore {
			highestScore = score
			best = c
		}
	}

	if highestScore < 0 {
		return primary.Lighten(0.25)
	}

	return best
}

func findDarkest(clusters []Cluster) RGB {
	minLum := math.MaxFloat64
	best := clusters[0].Center
	for _, cl := range clusters {
		lum := cl.Center.Luminance()
		if lum < minLum {
			minLum = lum
			best = cl.Center
		}
	}
	return best
}

func findLightest(clusters []Cluster) RGB {
	maxLum := -1.0
	best := clusters[0].Center
	for _, cl := range clusters {
		lum := cl.Center.Luminance()
		if lum > maxLum {
			maxLum = lum
			best = cl.Center
		}
	}
	return best
}

func generateANSI16(bg, fg, accent RGB, clusters []Cluster, isDark bool) [16]RGB {
	var c [16]RGB

	// 0: Black / Base Background
	c[0] = bg.Lighten(0.05)
	// 7: Foreground / Light
	c[7] = fg
	// 8: Bright Black (Muted Grey)
	c[8] = bg.Lighten(0.25)
	if !isDark {
		c[8] = bg.Darken(0.35)
	}
	// 15: Bright White
	c[15] = fg.Lighten(0.1)

	// Pick colors from clusters or construct harmonic base
	c[1] = RGB{R: 235, G: 111, B: 146} // Red
	c[2] = RGB{R: 156, G: 207, B: 154} // Green
	c[3] = RGB{R: 246, G: 193, B: 119} // Yellow
	c[4] = accent                      // Blue / Accent
	c[5] = RGB{R: 196, G: 167, B: 231} // Magenta
	c[6] = RGB{R: 140, G: 216, B: 224} // Cyan

	// Bright variants (lighten or saturate slightly)
	for i := 1; i <= 6; i++ {
		c[i+8] = c[i].Lighten(0.15)
	}

	return c
}
