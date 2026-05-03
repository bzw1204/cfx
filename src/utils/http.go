package utils

import (
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"resty.dev/v3"
)

type FetchConfig struct {
	Timeout        int
	ConnectTimeout int
	MaxRetries     int
	RetryDelay     int
	Logger         *zap.Logger
}

func FetchSource(remotes []string, cfg *FetchConfig) ([]string, error) {
	if len(remotes) == 0 {
		return []string{}, nil
	}

	if cfg == nil {
		cfg = &FetchConfig{
			Timeout:        10,
			ConnectTimeout: 5,
			MaxRetries:     3,
			RetryDelay:     30,
		}
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 10
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 2
	}

	type result struct {
		nodes []string
		err   error
		url   string
	}

	results := make(chan result, len(remotes))
	var wg sync.WaitGroup

	for _, url := range remotes {
		wg.Add(1)
		go func(remoteURL string) {
			defer wg.Done()

			var nodes []string
			var lastErr error

			for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
				client := resty.New()
				client.SetTransport(&http.Transport{
					Proxy: http.ProxyFromEnvironment,
				})
				client.SetTimeout(time.Duration(cfg.Timeout) * time.Second)

				resp, err := client.R().Get(remoteURL)
				if err != nil {
					lastErr = err
					client.Close()
					if attempt < cfg.MaxRetries {
						time.Sleep(time.Duration(cfg.RetryDelay) * time.Second)
						continue
					}
					break
				}

				nodes = ParseAdaptive(resp.String())
				client.Close()
				break
			}

			results <- result{nodes: nodes, err: lastErr, url: remoteURL}
		}(url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	seen := make(map[string]struct{})
	var allNodes []string

	for res := range results {
		if res.err != nil {
			if cfg.Logger != nil {
				cfg.Logger.Warn("获取节点失败", zap.String("url", res.url), zap.Error(res.err))
			}
			continue
		}
		for _, node := range res.nodes {
			if _, exists := seen[node]; !exists {
				seen[node] = struct{}{}
				allNodes = append(allNodes, node)
			}
		}
	}

	return allNodes, nil
}
