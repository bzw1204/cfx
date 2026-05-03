package network

import (
	"cfx/internal/utils"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// AvailabilityResult 可用性检测结果
type AvailabilityResult struct {
	Node      string
	Available bool
	Stack     string                 // ipv4/ipv6/unknown
	ExitInfo  map[string]interface{} // 出口信息
}

// CheckAvailability 检查单个节点的可用性
func CheckAvailability(nodeStr string, apiURL string, connectTimeout, readTimeout int) (*AvailabilityResult, error) {
	ip := utils.GetIPFromNode(nodeStr)
	port := utils.GetPortFromNode(nodeStr)

	if ip == "" || port == "" {
		return &AvailabilityResult{
			Node:      nodeStr,
			Available: false,
			Stack:     "unknown",
			ExitInfo:  map[string]interface{}{},
		}, fmt.Errorf("无效的节点格式")
	}

	proxyIP := fmt.Sprintf("%s:%s", ip, port)

	client := &http.Client{
		Timeout: time.Duration(readTimeout) * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Duration(connectTimeout) * time.Second,
			}).DialContext,
		},
	}

	resp, err := client.Get(fmt.Sprintf("%s?proxyip=%s", apiURL, proxyIP))
	if err != nil {
		return &AvailabilityResult{
			Node:      nodeStr,
			Available: false,
			Stack:     "unknown",
			ExitInfo:  map[string]interface{}{},
		}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &AvailabilityResult{
			Node:      nodeStr,
			Available: false,
			Stack:     "unknown",
			ExitInfo:  map[string]interface{}{},
		}, fmt.Errorf("API 返回状态码: %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return &AvailabilityResult{
			Node:      nodeStr,
			Available: false,
			Stack:     "unknown",
			ExitInfo:  map[string]interface{}{},
		}, err
	}

	success, _ := data["success"].(bool)
	stack := "unknown"
	if s, ok := data["inferred_stack"].(string); ok {
		stack = s
	}

	exitInfo := map[string]interface{}{}
	if probeResults, ok := data["probe_results"].(map[string]interface{}); ok {
		// 优先使用 IPv6，否则使用 IPv4
		var probe map[string]interface{}
		if ipv6Probe, ok := probeResults["ipv6"].(map[string]interface{}); ok {
			probe = ipv6Probe
		} else if ipv4Probe, ok := probeResults["ipv4"].(map[string]interface{}); ok {
			probe = ipv4Probe
		}

		if probe != nil {
			if exit, ok := probe["exit"].(map[string]interface{}); ok {
				exitInfo = exit
			}
		}
	}

	return &AvailabilityResult{
		Node:      nodeStr,
		Available: success,
		Stack:     stack,
		ExitInfo:  exitInfo,
	}, nil
}

// CheckAvailabilityBatch 批量检查节点可用性
func CheckAvailabilityBatch(nodes []string, apiURL string, connectTimeout, readTimeout int, maxWorkers int, progressCallback func(completed, total int)) []*AvailabilityResult {
	var results []*AvailabilityResult
	total := len(nodes)
	completed := 0

	sem := make(chan struct{}, maxWorkers)
	resultChan := make(chan *AvailabilityResult, total)

	for _, node := range nodes {
		go func(nodeStr string) {
			sem <- struct{}{}
			defer func() { <-sem }()

			result, _ := CheckAvailability(nodeStr, apiURL, connectTimeout, readTimeout)
			resultChan <- result
		}(node)
	}

	go func() {
		for result := range resultChan {
			results = append(results, result)
			completed++
			if progressCallback != nil {
				progressCallback(completed, total)
			}
		}
	}()

	for len(results) < total && completed < total {
		time.Sleep(10 * time.Millisecond)
	}

	close(resultChan)

	return results
}
