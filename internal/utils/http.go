package utils

import (
	"net/http"
	"strings"
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

			// 在重试循环外创建一次 resty 客户端，避免每次重试重建 transport
			client := resty.New()
			client.SetTransport(&http.Transport{
				Proxy: http.ProxyFromEnvironment,
			})
			client.SetTimeout(time.Duration(cfg.Timeout) * time.Second)
			defer client.Close()

			for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
				resp, err := client.R().Get(remoteURL)
				if err != nil {
					lastErr = err
					if attempt < cfg.MaxRetries {
						time.Sleep(time.Duration(cfg.RetryDelay) * time.Second)
						continue
					}
					break
				}

				nodes = ParseAdaptive(resp.String())
				break
			}

			results <- result{nodes: nodes, err: lastErr, url: remoteURL}
		}(url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// 基于 ip:port 去重（忽略国家标签差异）
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
			key := node
			if before, _, ok := strings.Cut(node, "#"); ok {
				key = before // ip:port
			}
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				allNodes = append(allNodes, node)
			}
		}
	}

	return allNodes, nil
}
