package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"syncpaper/internal/config"
)

type RedditSource struct {
	cfg config.RedditConfig
}

func NewRedditSource(cfg config.RedditConfig) *RedditSource {
	return &RedditSource{cfg: cfg}
}

func (r *RedditSource) Name() string {
	return "reddit"
}

type redditListing struct {
	Data struct {
		Children []struct {
			Data struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				URL       string `json:"url"`
				PostHint  string `json:"post_hint"`
				Over18    bool   `json:"over_18"`
				Subreddit string `json:"subreddit"`
				Preview   struct {
					Images []struct {
						Source struct {
							URL    string `json:"url"`
							Width  int    `json:"width"`
							Height int    `json:"height"`
						} `json:"source"`
					} `json:"images"`
				} `json:"preview"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func (r *RedditSource) Fetch(ctx context.Context, topic string, count int) ([]WallpaperItem, error) {
	subreddits := r.cfg.Subreddits
	if len(subreddits) == 0 {
		subreddits = []string{"wallpapers", "EarthPorn", "spaceporn"}
	}

	sub := subreddits[0]
	// If topic mentions space, earth, nature, pick matching subreddit if available
	lowerTopic := strings.ToLower(topic)
	for _, s := range subreddits {
		if strings.Contains(strings.ToLower(s), lowerTopic) {
			sub = s
			break
		}
	}

	apiURL := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=%d", sub, count*3)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create reddit request: %w", err)
	}
	// Reddit requires unique custom User-Agent to avoid 429
	req.Header.Set("User-Agent", "linux:syncpaper:v1.0 (by /u/syncpaper_bot)")

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reddit request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit returned status %d", resp.StatusCode)
	}

	var listing redditListing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, fmt.Errorf("failed to decode reddit response: %w", err)
	}

	var items []WallpaperItem
	for _, child := range listing.Data.Children {
		post := child.Data
		if post.Over18 {
			continue
		}

		imageURL := post.URL
		// Clean html escaped &amp;
		imageURL = strings.ReplaceAll(imageURL, "&amp;", "&")

		isDirectImage := strings.HasSuffix(strings.ToLower(imageURL), ".jpg") ||
			strings.HasSuffix(strings.ToLower(imageURL), ".jpeg") ||
			strings.HasSuffix(strings.ToLower(imageURL), ".png") ||
			strings.HasSuffix(strings.ToLower(imageURL), ".webp")

		if !isDirectImage && len(post.Preview.Images) > 0 {
			imageURL = strings.ReplaceAll(post.Preview.Images[0].Source.URL, "&amp;", "&")
			isDirectImage = true
		}

		if !isDirectImage {
			continue
		}

		width := 1920
		height := 1080
		if len(post.Preview.Images) > 0 {
			width = post.Preview.Images[0].Source.Width
			height = post.Preview.Images[0].Source.Height
		}

		items = append(items, WallpaperItem{
			ID:        fmt.Sprintf("reddit_%s", post.ID),
			Source:    "reddit",
			Topic:     topic,
			Title:     post.Title,
			URL:       imageURL,
			Thumbnail: imageURL,
			Width:     width,
			Height:    height,
			Format:    "image/jpeg",
			Extra: map[string]string{
				"subreddit": post.Subreddit,
			},
		})

		if len(items) >= count && count > 0 {
			break
		}
	}

	return items, nil
}
