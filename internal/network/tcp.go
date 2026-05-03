package network

import (
	"cfx/internal/utils"
	"fmt"
	"net"
	"time"
)

// TCPResult TCP 测试结果（用于存储TCP连接测试的结果）
type TCPResult struct {
	Node    string
	Latency float64 // 最小延迟（秒）
	Country string
	Success int // 成功次数
}

// TestTCPNode 测试单个节点的 TCP 连通性
func TestTCPNode(nodeStr string, timeout float64, probes int) (*TCPResult, error) {
	node, err := utils.ParseNode(nodeStr)
	if err != nil {
		return nil, err
	}

	minLatency := float64(1<<63 - 1) // 最大值
	success := 0

	for range probes {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", node.IP, node.Port), time.Duration(timeout*float64(time.Second)))
		if err != nil {
			continue
		}
		conn.Close()

		latency := time.Since(start).Seconds()
		if latency < minLatency {
			minLatency = latency
		}
		success++
	}

	if success == 0 {
		return nil, fmt.Errorf("所有探测均失败")
	}

	return &TCPResult{
		Node:    nodeStr,
		Latency: minLatency,
		Country: node.Country,
		Success: success,
	}, nil
}

// TestTCPNodes 并发测试多个节点
func TestTCPNodes(nodes []string, timeout float64, probes int, maxWorkers int, progressCallback func(completed, total int)) []*TCPResult {
	var results []*TCPResult
	total := len(nodes)
	completed := 0

	// 使用 channel 控制并发
	sem := make(chan struct{}, maxWorkers)
	resultChan := make(chan *TCPResult, total)

	for _, node := range nodes {
		go func(nodeStr string) {
			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量

			result, err := TestTCPNode(nodeStr, timeout, probes)
			if err == nil {
				resultChan <- result
			}
		}(node)
	}

	// 收集结果
	go func() {
		for result := range resultChan {
			results = append(results, result)
			completed++
			if progressCallback != nil {
				progressCallback(completed, total)
			}
		}
	}()

	// 等待所有任务完成
	for len(results) < total && completed < total {
		time.Sleep(10 * time.Millisecond)
	}

	close(resultChan)

	return results
}
