package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"syncpaper/internal/config"
)

type UnsplashSource struct {
	cfg config.UnsplashConfig
}

func NewUnsplashSource(cfg config.UnsplashConfig) *UnsplashSource {
	return &UnsplashSource{cfg: cfg}
}

func (u *UnsplashSource) Name() string {
	return "unsplash"
}

type unsplashSearchResponse struct {
	Results []struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		AltDesc     string `json:"alt_description"`
		Width       int    `json:"width"`
		Height      int    `json:"height"`
		URLs        struct {
			Raw     string `json:"raw"`
			Full    string `json:"full"`
			Regular string `json:"regular"`
			Thumb   string `json:"thumb"`
		} `json:"urls"`
	} `json:"results"`
}

func (u *UnsplashSource) Fetch(ctx context.Context, topic string, count int) ([]WallpaperItem, error) {
	if u.cfg.AccessKey == "" {
		return nil, errors.New("unsplash access_key is required in config to fetch from unsplash")
	}

	limit := count
	if limit <= 0 || limit > 30 {
		limit = 10
	}

	query := topic
	if query == "" {
		query = "wallpaper"
	}

	apiURL := fmt.Sprintf("https://api.unsplash.com/search/photos?query=%s&per_page=%d&orientation=landscape",
		url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create unsplash request: %w", err)
	}
	req.Header.Set("Authorization", "Client-ID "+u.cfg.AccessKey)
	req.Header.Set("User-Agent", "syncpaper/1.0 (Linux; wallpaper utility)")

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unsplash request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unsplash returned status %d", resp.StatusCode)
	}

	var data unsplashSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode unsplash response: %w", err)
	}

	var items []WallpaperItem
	for _, item := range data.Results {
		title := item.Description
		if title == "" {
			title = item.AltDesc
		}
		if title == "" {
			title = fmt.Sprintf("Unsplash %s", topic)
		}

		// Use raw with standard UHD parameters or full
		imgURL := item.URLs.Full
		if imgURL == "" {
			imgURL = item.URLs.Regular
		}

		items = append(items, WallpaperItem{
			ID:        fmt.Sprintf("unsplash_%s", item.ID),
			Source:    "unsplash",
			Topic:     topic,
			Title:     title,
			URL:       imgURL,
			Thumbnail: item.URLs.Thumb,
			Width:     item.Width,
			Height:    item.Height,
			Format:    "image/jpeg",
		})
	}

	return items, nil
}
