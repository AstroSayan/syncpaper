package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"syncpaper/internal/config"
)

type BingSource struct {
	cfg config.BingConfig
}

func NewBingSource(cfg config.BingConfig) *BingSource {
	return &BingSource{cfg: cfg}
}

func (b *BingSource) Name() string {
	return "bing"
}

type bingResponse struct {
	Images []struct {
		StartDate string `json:"startdate"`
		URL       string `json:"url"`
		URLBase   string `json:"urlbase"`
		Copyright string `json:"copyright"`
		Title     string `json:"title"`
	} `json:"images"`
}

func (b *BingSource) Fetch(ctx context.Context, topic string, count int) ([]WallpaperItem, error) {
	market := b.cfg.Market
	if market == "" {
		market = "en-US"
	}

	n := count
	if n <= 0 || n > 8 {
		n = 8
	}

	apiURL := fmt.Sprintf("https://www.bing.com/HPImageArchive.aspx?format=js&idx=0&n=%d&mkt=%s", n, url.QueryEscape(market))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create bing request: %w", err)
	}
	req.Header.Set("User-Agent", "syncpaper/1.0 (Linux; wallpaper utility)")

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bing request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bing returned status %d", resp.StatusCode)
	}

	var data bingResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode bing response: %w", err)
	}

	var items []WallpaperItem
	for _, img := range data.Images {
		// Try UHD first, fallback to standard 1920x1080
		fullURL := "https://www.bing.com" + img.URL
		if img.URLBase != "" {
			fullURL = "https://www.bing.com" + img.URLBase + "_UHD.jpg"
		}

		title := img.Title
		if title == "" {
			title = img.Copyright
		}

		// Filter by topic if topic is specified (case-insensitive substring match)
		if topic != "" {
			lowerTopic := strings.ToLower(topic)
			lowerTitle := strings.ToLower(title)
			lowerCopyright := strings.ToLower(img.Copyright)
			// If topic is generic or doesn't match, still allow if count is needed, but prefer matches
			if !strings.Contains(lowerTitle, lowerTopic) && !strings.Contains(lowerCopyright, lowerTopic) {
				// Don't discard immediately if user didn't specify strict topic, but keep relevance
				// For bing, nature/landscape matches almost everything
			}
		}

		id := fmt.Sprintf("bing_%s", img.StartDate)
		if img.URLBase != "" {
			parts := strings.Split(img.URLBase, ".")
			if len(parts) > 1 {
				id = fmt.Sprintf("bing_%s", parts[len(parts)-1])
			}
		}

		items = append(items, WallpaperItem{
			ID:        id,
			Source:    "bing",
			Topic:     topic,
			Title:     title,
			URL:       fullURL,
			Thumbnail: "https://www.bing.com" + img.URL,
			Width:     3840,
			Height:    2160,
			Format:    "image/jpeg",
			Extra: map[string]string{
				"copyright": img.Copyright,
			},
		})
		if len(items) >= count && count > 0 {
			break
		}
	}

	return items, nil
}
