package source

import (
	"context"
	"net/http"
	"time"
)

// WallpaperItem represents metadata for a wallpaper available to download
type WallpaperItem struct {
	ID        string            `json:"id"`
	Source    string            `json:"source"`
	Topic     string            `json:"topic"`
	Title     string            `json:"title"`
	URL       string            `json:"url"`
	Thumbnail string            `json:"thumbnail"`
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	Format    string            `json:"format"`
	Extra     map[string]string `json:"extra,omitempty"`
}

// Source defines the interface that each wallpaper provider implements
type Source interface {
	Name() string
	Fetch(ctx context.Context, topic string, count int) ([]WallpaperItem, error)
}

// DefaultHTTPClient provides a fast, timeout-safe client with connection pooling
var DefaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     60 * time.Second,
	},
}
