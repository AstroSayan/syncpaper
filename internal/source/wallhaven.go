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

type WallhavenSource struct {
	cfg config.WallhavenConfig
}

func NewWallhavenSource(cfg config.WallhavenConfig) *WallhavenSource {
	return &WallhavenSource{cfg: cfg}
}

func (w *WallhavenSource) Name() string {
	return "wallhaven"
}

type wallhavenResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Path       string `json:"path"`
		DimensionX int    `json:"dimension_x"`
		DimensionY int    `json:"dimension_y"`
		FileType   string `json:"file_type"`
		Thumbs     struct {
			Large string `json:"large"`
			Small string `json:"small"`
		} `json:"thumbs"`
	} `json:"data"`
}

func (w *WallhavenSource) Fetch(ctx context.Context, topic string, count int) ([]WallpaperItem, error) {
	baseURL := "https://wallhaven.cc/api/v1/search"
	params := url.Values{}

	if topic != "" {
		params.Set("q", topic)
	}

	categories := w.cfg.Categories
	if categories == "" {
		categories = "110"
	}
	params.Set("categories", categories)

	purity := w.cfg.Purity
	if purity == "" {
		purity = "100"
	}
	params.Set("purity", purity)

	sorting := w.cfg.Sorting
	if sorting == "" {
		sorting = "random"
	}
	params.Set("sorting", sorting)

	if len(w.cfg.Resolutions) > 0 {
		params.Set("resolutions", strings.Join(w.cfg.Resolutions, ","))
	}
	if len(w.cfg.Ratios) > 0 {
		params.Set("ratios", strings.Join(w.cfg.Ratios, ","))
	}
	if w.cfg.APIKey != "" {
		params.Set("apikey", w.cfg.APIKey)
	}

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallhaven request: %w", err)
	}
	req.Header.Set("User-Agent", "syncpaper/1.0 (Linux; wallpaper utility)")

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wallhaven request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wallhaven returned status %d", resp.StatusCode)
	}

	var data wallhavenResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode wallhaven response: %w", err)
	}

	var items []WallpaperItem
	limit := count
	if limit <= 0 || limit > len(data.Data) {
		limit = len(data.Data)
	}

	for i := 0; i < limit; i++ {
		item := data.Data[i]
		items = append(items, WallpaperItem{
			ID:        fmt.Sprintf("wallhaven_%s", item.ID),
			Source:    "wallhaven",
			Topic:     topic,
			Title:     fmt.Sprintf("%s (%dx%d)", topic, item.DimensionX, item.DimensionY),
			URL:       item.Path,
			Thumbnail: item.Thumbs.Large,
			Width:     item.DimensionX,
			Height:    item.DimensionY,
			Format:    item.FileType,
		})
	}

	return items, nil
}
