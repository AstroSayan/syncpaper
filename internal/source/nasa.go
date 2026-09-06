package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"syncpaper/internal/config"
)

type NASASource struct {
	cfg config.NASAConfig
}

func NewNASASource(cfg config.NASAConfig) *NASASource {
	return &NASASource{cfg: cfg}
}

func (n *NASASource) Name() string {
	return "nasa"
}

type nasaItem struct {
	Date        string `json:"date"`
	Explanation string `json:"explanation"`
	HDURL       string `json:"hdurl"`
	URL         string `json:"url"`
	MediaType   string `json:"media_type"`
	Title       string `json:"title"`
}

func (n *NASASource) Fetch(ctx context.Context, topic string, count int) ([]WallpaperItem, error) {
	apiKey := n.cfg.APIKey
	if apiKey == "" {
		apiKey = "DEMO_KEY"
	}

	limit := count
	if limit <= 0 || limit > 10 {
		limit = 5
	}

	apiURL := fmt.Sprintf("https://api.nasa.gov/planetary/apod?api_key=%s&count=%d", url.QueryEscape(apiKey), limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create nasa request: %w", err)
	}
	req.Header.Set("User-Agent", "syncpaper/1.0 (Linux; wallpaper utility)")

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nasa request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nasa returned status %d", resp.StatusCode)
	}

	var data []nasaItem
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode nasa response: %w", err)
	}

	var items []WallpaperItem
	for _, item := range data {
		if item.MediaType != "image" {
			continue
		}

		targetURL := item.HDURL
		if targetURL == "" {
			targetURL = item.URL
		}
		if targetURL == "" {
			continue
		}

		items = append(items, WallpaperItem{
			ID:        fmt.Sprintf("nasa_%s", item.Date),
			Source:    "nasa",
			Topic:     topic,
			Title:     item.Title,
			URL:       targetURL,
			Thumbnail: item.URL,
			Width:     1920, // default estimate if not provided
			Height:    1080,
			Format:    "image/jpeg",
			Extra: map[string]string{
				"explanation": item.Explanation,
				"date":        item.Date,
			},
		})
	}

	return items, nil
}
