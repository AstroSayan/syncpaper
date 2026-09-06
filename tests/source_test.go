package tests

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"syncpaper/internal/config"
	"syncpaper/internal/source"
)

func TestSourceRegistry(t *testing.T) {
	cfg := config.DefaultConfig()

	sources := source.GetEnabledSources(cfg)
	if len(sources) < 2 {
		t.Errorf("expected at least 2 enabled sources by default, got %d", len(sources))
	}

	names := make(map[string]bool)
	for _, s := range sources {
		names[s.Name()] = true
	}

	if !names["wallhaven"] {
		t.Errorf("expected wallhaven to be in enabled sources")
	}
	if !names["bing"] {
		t.Errorf("expected bing to be in enabled sources")
	}

	// Disable wallhaven
	cfg.Sources.Wallhaven.Enabled = false
	sourcesAfter := source.GetEnabledSources(cfg)
	for _, s := range sourcesAfter {
		if s.Name() == "wallhaven" {
			t.Errorf("wallhaven should be disabled")
		}
	}
}

type mockTransport struct {
	fn func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.fn(req)
}

func TestSourceFetches(t *testing.T) {
	oldTransport := source.DefaultHTTPClient.Transport
	defer func() { source.DefaultHTTPClient.Transport = oldTransport }()

	source.DefaultHTTPClient.Transport = &mockTransport{
		fn: func(req *http.Request) (*http.Response, error) {
			urlStr := req.URL.String()
			var body string

			if strings.Contains(urlStr, "bing.com") {
				body = `{"images":[{"startdate":"20260906","urlbase":"/th?id=OHR.Test","title":"Lake","copyright":"Photo"}]}`
			} else if strings.Contains(urlStr, "wallhaven.cc") {
				body = `{"data":[{"id":"wh1","path":"https://w.wallhaven.cc/test.jpg","thumbs":{"large":"https://th.wallhaven.cc/test.jpg"},"dimension_x":3840,"dimension_y":2160,"file_type":"image/jpeg"}]}`
			} else if strings.Contains(urlStr, "api.nasa.gov") {
				body = `[{"date":"2026-09-06","title":"Galaxy","hdurl":"https://apod.nasa.gov/test.jpg","media_type":"image"}]`
			} else if strings.Contains(urlStr, "reddit.com") {
				body = `{"data":{"children":[{"data":{"id":"rd1","title":"Forest","url":"https://i.redd.it/forest.jpg","post_hint":"image"}}]}}`
			} else if strings.Contains(urlStr, "unsplash.com") {
				body = `{"results":[{"id":"un1","description":"Desert","urls":{"raw":"https://images.unsplash.com/desert","thumb":"https://images.unsplash.com/thumb"},"width":3000,"height":2000}]}`
			} else {
				body = `{}`
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		},
	}

	ctx := context.Background()

	// 1. Bing
	bing := source.NewBingSource(config.BingConfig{Enabled: true, Market: "en-US"})
	bItems, err := bing.Fetch(ctx, "nature", 1)
	if err != nil || len(bItems) == 0 {
		t.Fatalf("Bing Fetch failed: %v", err)
	}
	if bItems[0].Source != "bing" || bItems[0].Title != "Lake" {
		t.Errorf("unexpected bing item: %+v", bItems[0])
	}

	// 2. Wallhaven
	wh := source.NewWallhavenSource(config.WallhavenConfig{Enabled: true, Resolutions: []string{"1920x1080"}})
	wItems, err := wh.Fetch(ctx, "nature", 1)
	if err != nil || len(wItems) == 0 {
		t.Fatalf("Wallhaven Fetch failed: %v", err)
	}
	if wItems[0].Source != "wallhaven" || wItems[0].Width != 3840 {
		t.Errorf("unexpected wallhaven item: %+v", wItems[0])
	}

	// 3. NASA
	nasa := source.NewNASASource(config.NASAConfig{Enabled: true})
	nItems, err := nasa.Fetch(ctx, "space", 1)
	if err != nil || len(nItems) == 0 {
		t.Fatalf("NASA Fetch failed: %v", err)
	}
	if nItems[0].Source != "nasa" || nItems[0].Title != "Galaxy" {
		t.Errorf("unexpected NASA item: %+v", nItems[0])
	}

	// 4. Reddit
	reddit := source.NewRedditSource(config.RedditConfig{Enabled: true, Subreddits: []string{"wallpapers"}})
	rItems, err := reddit.Fetch(ctx, "nature", 1)
	if err != nil || len(rItems) == 0 {
		t.Fatalf("Reddit Fetch failed: %v", err)
	}
	if rItems[0].Source != "reddit" || rItems[0].Title != "Forest" {
		t.Errorf("unexpected Reddit item: %+v", rItems[0])
	}

	// 5. Unsplash
	unsplash := source.NewUnsplashSource(config.UnsplashConfig{Enabled: true, AccessKey: "test_key"})
	uItems, err := unsplash.Fetch(ctx, "nature", 1)
	if err != nil || len(uItems) == 0 {
		t.Fatalf("Unsplash Fetch failed: %v", err)
	}
	if uItems[0].Source != "unsplash" || uItems[0].Width != 3000 {
		t.Errorf("unexpected Unsplash item: %+v", uItems[0])
	}

	// 6. FetchWallpapers concurrent registry
	cfg := config.DefaultConfig()
	cfg.Sources.Wallhaven.Enabled = true
	cfg.Sources.Bing.Enabled = true
	cfg.Sources.NASA.Enabled = true
	all, errs := source.FetchWallpapers(ctx, cfg, "nature", 1)
	if len(errs) > 0 {
		t.Errorf("unexpected errors during FetchWallpapers: %v", errs)
	}
	if len(all) < 2 {
		t.Errorf("expected at least 2 items from multiple sources, got %d", len(all))
	}
}
