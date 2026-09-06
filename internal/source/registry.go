package source

import (
	"context"
	"fmt"
	"sync"

	"syncpaper/internal/config"
)

// GetEnabledSources returns all sources that have been enabled in config
func GetEnabledSources(cfg *config.Config) []Source {
	var sources []Source
	if cfg.Sources.Wallhaven.Enabled {
		sources = append(sources, NewWallhavenSource(cfg.Sources.Wallhaven))
	}
	if cfg.Sources.Bing.Enabled {
		sources = append(sources, NewBingSource(cfg.Sources.Bing))
	}
	if cfg.Sources.NASA.Enabled {
		sources = append(sources, NewNASASource(cfg.Sources.NASA))
	}
	if cfg.Sources.Reddit.Enabled {
		sources = append(sources, NewRedditSource(cfg.Sources.Reddit))
	}
	if cfg.Sources.Unsplash.Enabled && cfg.Sources.Unsplash.AccessKey != "" {
		sources = append(sources, NewUnsplashSource(cfg.Sources.Unsplash))
	}
	return sources
}

// FetchWallpapers queries all enabled sources across topics concurrently
func FetchWallpapers(ctx context.Context, cfg *config.Config, filterTopic string, countOverride int) ([]WallpaperItem, []error) {
	sources := GetEnabledSources(cfg)
	if len(sources) == 0 {
		return nil, []error{fmt.Errorf("no wallpaper sources enabled in configuration")}
	}

	topics := cfg.Topics.List
	if filterTopic != "" {
		topics = []string{filterTopic}
	}

	countPerTopic := cfg.Topics.CountPerTopic
	if countOverride > 0 {
		countPerTopic = countOverride
	}

	type fetchTask struct {
		src   Source
		topic string
		count int
	}

	var tasks []fetchTask
	for _, topic := range topics {
		for _, src := range sources {
			// Allocate count evenly or per source
			c := countPerTopic / len(sources)
			if c < 1 {
				c = 1
			}
			tasks = append(tasks, fetchTask{src: src, topic: topic, count: c})
		}
	}

	var (
		mu       sync.Mutex
		results  []WallpaperItem
		errs     []error
		wg       sync.WaitGroup
		poolSize = 4
		sem      = make(chan struct{}, poolSize)
	)

	for _, t := range tasks {
		wg.Add(1)
		sem <- struct{}{}

		go func(task fetchTask) {
			defer wg.Done()
			defer func() { <-sem }()

			items, err := task.src.Fetch(ctx, task.topic, task.count)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s [%s]: %w", task.src.Name(), task.topic, err))
			} else {
				results = append(results, items...)
			}
		}(t)
	}

	wg.Wait()
	return results, errs
}
