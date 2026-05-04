package network

import (
	"cfx/internal/utils"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type ProbeResult struct {
	Node     string  // 节点
	Protocol string  // 协议
	Latency  float64 // 最小延迟（秒）
	Country  string  // 国家
	Success  int     // 成功次数
}

// 探测 TCP 节点
// node 节点，timeout 超时时间，probes 探测次数
// 返回探测结果，错误信息
func ProbeTCPNode(nodeStr string, timeout float64, probes int) (*ProbeResult, error) {
	node, err := utils.ParseNode(nodeStr)
	if err != nil {
		return nil, err
	}

	minLatency := float64(1<<63 - 1) // 最大值
	success := 0
	consecutiveFails := 0

	for range probes {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", node.IP, node.Port), time.Duration(timeout*float64(time.Second)))
		if err != nil {
			consecutiveFails++
			// 连续失败 2 次且从未成功过 → 提前终止该节点
			if consecutiveFails >= 2 && success == 0 {
				break
			}
			continue
		}
		consecutiveFails = 0

		// SetLinger(0) → 关闭时发 RST 替代 FIN，跳过 TIME_WAIT
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetLinger(0)
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

	return &ProbeResult{
		Node:    nodeStr,
		Latency: minLatency,
		Country: node.Country,
		Success: success,
	}, nil
}

// 批量探测 TCP 节点
// nodes 节点列表，timeout 超时时间，probes 探测次数，maxWorkers 最大并发数，progressCallback 进度回调
// 返回结果列表，错误信息
func ProbeTCPNodes(nodes []string, timeout float64, probes int, maxWorkers int, progressCallback func(completed, total int)) ([]*ProbeResult, error) {
	total := len(nodes)
	results := make([]*ProbeResult, 0, total)

	sem := make(chan struct{}, maxWorkers)
	resultChan := make(chan *ProbeResult, total)

	var wg sync.WaitGroup
	var completed int32

	for _, node := range nodes {
		wg.Add(1)

		go func(nodeStr string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := ProbeTCPNode(nodeStr, timeout, probes)
			if err == nil {
				resultChan <- result
			}

			// 无论成功失败，都算完成
			cur := atomic.AddInt32(&completed, 1)
			if progressCallback != nil {
				progressCallback(int(cur), total)
			}
		}(node)
	}

	// 关闭 channel 的唯一正确方式
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		results = append(results, result)
	}

	return results, nil
}
