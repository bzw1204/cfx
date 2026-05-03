package network

import (
	"cfx/src/utils"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// BandwidthResult 带宽测速结果
type BandwidthResult struct {
	Node  string
	Speed float64 // Mbps
}

// MeasureBandwidth 测量单个节点的带宽
func MeasureBandwidth(nodeStr string, bandwidthURL string, connectTimeout, readTimeout int) (*BandwidthResult, error) {
	ip := utils.GetIPFromNode(nodeStr)
	port := utils.GetPortFromNode(nodeStr)

	if ip == "" || port == "" {
		return &BandwidthResult{Node: nodeStr, Speed: 0}, fmt.Errorf("无效的节点格式")
	}

	// 创建自定义 Transport，使用 --resolve 类似的功能
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// 将 speed.cloudflare.com 解析到目标 IP
			if strings.HasPrefix(addr, "speed.cloudflare.com:") {
				addr = fmt.Sprintf("%s:%s", ip, port)
			}
			var d net.Dialer
			d.Timeout = time.Duration(connectTimeout) * time.Second
			return d.DialContext(ctx, network, addr)
		},
		TLSHandshakeTimeout: time.Duration(connectTimeout) * time.Second,
	}

	client := &http.Client{
		Timeout:   time.Duration(readTimeout) * time.Second,
		Transport: transport,
	}

	start := time.Now()
	resp, err := client.Get(bandwidthURL)
	if err != nil {
		return &BandwidthResult{Node: nodeStr, Speed: 0}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &BandwidthResult{Node: nodeStr, Speed: 0}, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &BandwidthResult{Node: nodeStr, Speed: 0}, err
	}

	elapsed := time.Since(start).Seconds()
	sizeBytes := float64(len(body))

	if elapsed > 0 && sizeBytes > 0 {
		speedMbps := (sizeBytes * 8) / (elapsed * 1000 * 1000)
		return &BandwidthResult{
			Node:  nodeStr,
			Speed: speedMbps,
		}, nil
	}

	return &BandwidthResult{Node: nodeStr, Speed: 0}, nil
}

// MeasureBandwidthBatch 批量测量节点带宽
func MeasureBandwidthBatch(nodes []string, bandwidthURL string, connectTimeout, readTimeout int, maxWorkers int, progressCallback func(completed, total int)) []*BandwidthResult {
	var results []*BandwidthResult
	total := len(nodes)
	completed := 0

	sem := make(chan struct{}, maxWorkers)
	resultChan := make(chan *BandwidthResult, total)

	for _, node := range nodes {
		go func(nodeStr string) {
			sem <- struct{}{}
			defer func() { <-sem }()

			result, _ := MeasureBandwidth(nodeStr, bandwidthURL, connectTimeout, readTimeout)
			resultChan <- result
		}(node)
	}

	go func() {
		for result := range resultChan {
			if result.Speed > 0 {
				results = append(results, result)
			}
			completed++
			if progressCallback != nil {
				progressCallback(completed, total)
			}
		}
	}()

	for completed < total {
		time.Sleep(10 * time.Millisecond)
	}

	close(resultChan)

	// 按速度降序排序
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Speed > results[i].Speed {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}
