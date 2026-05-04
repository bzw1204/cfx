package internal

import (
	"cfx/internal/config"
	"cfx/internal/model"
	"cfx/internal/network"
	"cfx/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ────────── 配置打印 ──────────

// PrintConfigInfo 打印配置概要信息
func PrintConfigInfo(cfg *model.Config) {
	logger := config.GetLogger()
	modeStr := "全局最优%d个"
	if !cfg.Global.Enable {
		modeStr = "每个国家最优%d个"
	}
	logger.Sugar().Infof("当前模式："+modeStr+"，每个节点测试 %d 次 TCP 连接", cfg.Global.TopN, cfg.Tcp.Probes)
	logger.Sugar().Infof("最低成功率要求：%.0f%%", cfg.Tcp.MinSuccessRate*100)
	logger.Sugar().Infof("IP 可用性二次筛选：%s (仅对候选节点)", utils.Bool2Str(cfg.Availability.Enabled, "启用", "禁用"))
	logger.Sugar().Infof("IPv6 客户端 IP 过滤(仅作用于DNS更新环节): %s", utils.Bool2Str(cfg.Availability.IPV6Availability, "启用", "禁用"))
	logger.Sugar().Infof("DNS黑名单过滤: %s, 黑名单国家：%s", utils.Bool2Str(cfg.Filter.BlockedEnabled, "启用", "禁用"), strings.Join(cfg.Filter.BlockedCountries, ", "))
	logger.Sugar().Infof("带宽测速候选数: %d, 测速文件大小：%.1f MB,超时：%ds", cfg.Global.BandwidthCandidates, cfg.Bandwidth.SizeMB, cfg.Bandwidth.Timeout)
	if cfg.Filter.CountriesEnabled {
		logger.Sugar().Infof("前置白名单过滤：启用，仅保留：%s", strings.Join(cfg.Filter.AllowedCountries, ", "))
	}
}

// ────────── 节点获取 ──────────

// FetchNodes 从数据源获取并合并节点
func FetchNodes(cfg *model.Config) []string {
	logger := config.GetLogger()
	remotes := make([]string, 0, len(cfg.AdditionalSources))
	for _, s := range cfg.AdditionalSources {
		if s.Enabled {
			remotes = append(remotes, s.Url)
		}
	}
	allNodes, err := utils.FetchSource(remotes, &utils.FetchConfig{
		MaxRetries:     3,
		RetryDelay:     5,
		Timeout:        30,
		ConnectTimeout: 5,
		Logger:         logger,
	})
	if err != nil {
		logger.Sugar().Warnf("获取节点时部分源出错: %v", err)
	}
	logger.Sugar().Infof("合并后总计 %d 个节点", len(allNodes))
	return allNodes
}

// ────────── 前置过滤 ──────────

// PortFilter 前置端口过滤
func PortFilter(nodes []string, cfg *model.Config) []string {
	logger := config.GetLogger()
	if cfg.Filter.PortEnabled {
		before := len(nodes)
		var ports []string
		for _, p := range cfg.Filter.Ports {
			ports = append(ports, fmt.Sprintf("%d", p))
		}
		nodes = utils.FilterByPort(nodes, ports)
		if after := len(nodes); after > 0 {
			logger.Sugar().Infof("前置端口过滤 (仅保留端口 %s) :%d -> %d 个节点", strings.Join(ports, ", "), before, after)
		} else {
			logger.Sugar().Fatal("前置端口过滤后无任何节点")
		}
	}
	return nodes
}

// BlockedFilter 前置黑名单过滤
func BlockedFilter(nodes []string, cfg *model.Config) []string {
	logger := config.GetLogger()
	if cfg.Filter.BlockedEnabled && len(cfg.Filter.BlockedCountries) > 0 {
		before := len(nodes)
		nodes = utils.FilterByBlockedCountries(nodes, cfg.Filter.BlockedCountries)
		if after := len(nodes); after > 0 {
			logger.Sugar().Infof("前置黑名单过滤：%d -> %d 个节点（已屏蔽：%s)", before, after, strings.Join(cfg.Filter.BlockedCountries, ", "))
		} else {
			logger.Sugar().Fatal("前置黑名单过滤后无任何节点")
		}
	}
	return nodes
}

// WhiteListFilter 白名单过滤
func WhiteListFilter(nodes []string, cfg *model.Config) []string {
	logger := config.GetLogger()
	if cfg.Filter.CountriesEnabled && len(cfg.Filter.AllowedCountries) > 0 {
		before := len(nodes)
		nodes = utils.FilterByAllowedCountries(nodes, cfg.Filter.AllowedCountries)
		if after := len(nodes); after > 0 {
			logger.Sugar().Infof("国家过滤（测试前）：%d -> %d 个节点（允许国家：%s)", before, after, strings.Join(cfg.Filter.AllowedCountries, ", "))
		} else {
			logger.Sugar().Fatal("过滤后无任何节点")
		}
	}
	return nodes
}

// ────────── TCP 测试 ──────────

// TCPTry 执行 TCP 连通性测试，返回按成功率和延迟排序的结果
func TCPTry(nodes []string, cfg *model.Config) []*network.ProbeResult {
	logger := config.GetLogger()
	logger.Sugar().Infof("开始 TCP 连接测试（超时 %.1fs, 并发 %d)", cfg.Tcp.Timeout, cfg.Tcp.MaxWorkers)
	tcpResults, err := network.ProbeTCPNodes(nodes, float64(cfg.Tcp.Timeout), cfg.Tcp.Probes, cfg.Tcp.MaxWorkers, func(completed, total int) {
		utils.PrintProgress("TCP 测试", completed, total, "")
	})
	if err != nil {
		logger.Sugar().Fatal("TCP 测试出错", zap.Error(err))
		return nil
	}
	logger.Sugar().Info("TCP 测试完成")

	if len(tcpResults) == 0 {
		logger.Sugar().Fatal("没有通过成功率筛选的节点，请检查网络或降低 MIN_SUCCESS_RATE")
	}
	// 按成功率和延迟排序
	sort.Slice(tcpResults, func(i, j int) bool {
		if tcpResults[i].Success != tcpResults[j].Success {
			return tcpResults[i].Success > tcpResults[j].Success
		}
		return tcpResults[i].Latency < tcpResults[j].Latency
	})
	return tcpResults
}

// ────────── 带宽测速候选选取 ──────────

// SelectBandwidthCandidates 从 TCP 测试结果中选择进入带宽测速的候选节点
// 返回值: candidates 节点列表, latencyMap 节点→延迟映射
func SelectBandwidthCandidates(tcpResults []*network.ProbeResult, cfg *model.Config) (candidates []string, latencyMap map[string]float64) {
	logger := config.GetLogger()

	// 构建延迟映射
	latencyMap = make(map[string]float64, len(tcpResults))
	for _, r := range tcpResults {
		latencyMap[r.Node] = r.Latency
	}

	if cfg.Global.Enable {
		limit := min(cfg.Global.BandwidthCandidates, len(tcpResults))
		candidates = make([]string, 0, limit)
		for _, r := range tcpResults[:limit] {
			candidates = append(candidates, r.Node)
		}
		logger.Sugar().Infof("TCP 最优前 %d 个节点进入候选池", len(candidates))
	} else {
		countryNodes := make(map[string][]*network.ProbeResult)
		for _, r := range tcpResults {
			countryNodes[r.Country] = append(countryNodes[r.Country], r)
		}
		totalCountries := len(countryNodes)
		baseLimit := max(cfg.Global.BandwidthCandidates/totalCountries, 1)

		for _, nodes := range countryNodes {
			sort.Slice(nodes, func(i, j int) bool {
				if nodes[i].Success != nodes[j].Success {
					return nodes[i].Success > nodes[j].Success
				}
				return nodes[i].Latency < nodes[j].Latency
			})
			limit := baseLimit
			if limit > len(nodes) {
				limit = len(nodes)
			}
			for _, r := range nodes[:limit] {
				candidates = append(candidates, r.Node)
			}
		}
		logger.Sugar().Infof("各国家候选池分配：共 %d 个国家，每国最多 %d 个候选，总计 %d 个节点进入候选池", totalCountries, baseLimit, len(candidates))
	}
	return
}

// ────────── 可用性检测 ──────────

// CheckAvailabilityWithRetry 对候选节点进行可用性二次检测，支持自动重试
// 返回值: passed 通过节点列表, ipInfo 节点→IP协议栈映射
func CheckAvailabilityWithRetry(candidates []string, cfg *model.Config) (passed []string, ipInfo map[string]string) {
	logger := config.GetLogger()
	if !cfg.Availability.Enabled {
		return candidates, make(map[string]string)
	}

	for attempt := 1; attempt <= cfg.Availability.Retry; attempt++ {
		logger.Sugar().Infof("[可用性检测] 第 %d 轮检测", attempt)

		total := len(candidates)
		completed := 0
		lastPrint := time.Now()

		sem := make(chan struct{}, cfg.Availability.MaxWorkers)
		resultChan := make(chan *network.AvailabilityResult, total)

		for _, node := range candidates {
			go func(nodeStr string) {
				sem <- struct{}{}
				defer func() { <-sem }()
				result, _ := network.CheckAvailability(nodeStr, cfg.Availability.CheckApi, cfg.Availability.ConnectTimeout, cfg.Availability.Timeout)
				resultChan <- result
			}(node)
		}

		var availResults []*network.AvailabilityResult
		passedCount := 0
		for completed < total {
			result := <-resultChan
			availResults = append(availResults, result)
			if result.Available {
				passedCount++
			}
			completed++
			now := time.Now()
			if now.Sub(lastPrint).Seconds() >= 1 || completed == total {
				utils.PrintProgress("[可用性检测]", completed, total, fmt.Sprintf("通过数量：%d", passedCount))
				lastPrint = now
			}
		}

		passed = make([]string, 0, passedCount)
		ipInfo = make(map[string]string, passedCount)
		for _, r := range availResults {
			if r.Available {
				passed = append(passed, r.Node)
				ipInfo[r.Node] = r.Stack
			}
		}
		if len(passed) > 0 {
			logger.Sugar().Infof("可用性检测通过 %d 个节点", len(passed))
			return
		}

		if attempt < cfg.Availability.Retry {
			logger.Sugar().Warnf("本轮可用性检测通过率为 0%%，等待 %d 秒后重试", cfg.Availability.RetryDelay)
			time.Sleep(time.Duration(cfg.Availability.RetryDelay) * time.Second)
		}
	}

	logger.Sugar().Errorf("可用性检测经 %d 轮重试后仍无节点通过，降级使用原始候选列表", cfg.Availability.Retry)
	return candidates, make(map[string]string)
}

// ────────── 带宽测速 ──────────

// MeasureBandwidthWithRetry 对候选节点进行带宽测速，支持自动重试
// 返回值: 测速结果列表（按速度降序），全部失败时返回 nil
func MeasureBandwidthWithRetry(candidates []string, cfg *model.Config) []*network.BandwidthResult {
	logger := config.GetLogger()
	if !cfg.Bandwidth.Enabled {
		return nil
	}

	bandwidthURL := fmt.Sprintf("https://speed.cloudflare.com/__down?bytes=%d", int(cfg.Bandwidth.SizeMB*1024*1024))

	for attempt := 1; attempt <= cfg.Bandwidth.Retry; attempt++ {
		logger.Sugar().Infof("[带宽测速] 第 %d 轮测试", attempt)

		total := len(candidates)
		completed := 0
		lastPrint := time.Now()

		sem := make(chan struct{}, cfg.Bandwidth.MaxWorkers)
		resultChan := make(chan *network.BandwidthResult, total)

		for _, node := range candidates {
			go func(nodeStr string) {
				sem <- struct{}{}
				defer func() { <-sem }()
				result, _ := network.MeasureBandwidth(nodeStr, bandwidthURL, cfg.Bandwidth.ConnectTimeout, cfg.Bandwidth.Timeout)
				resultChan <- result
			}(node)
		}

		var bwResults []*network.BandwidthResult
		for completed < total {
			result := <-resultChan
			if result.Speed > 0 {
				bwResults = append(bwResults, result)
			}
			completed++
			now := time.Now()
			if now.Sub(lastPrint).Seconds() >= 1 || completed == total {
				utils.PrintProgress("[带宽测速]", completed, total, "")
				lastPrint = now
			}
		}

		if len(bwResults) > 0 {
			return bwResults
		}

		if attempt < cfg.Bandwidth.Retry {
			logger.Sugar().Warnf("本轮测速无有效结果，等待 %d 秒后重试", cfg.Bandwidth.RetryDelay)
			time.Sleep(time.Duration(cfg.Bandwidth.RetryDelay) * time.Second)
		}
	}
	return nil
}

// ────────── 最终节点选取 ──────────

// SelectFinalNodes 从测速结果中选取最终节点
// 有带宽结果时优先按带宽选取；带宽全部失败则降级到 TCP 排序
func SelectFinalNodes(tcpResults []*network.ProbeResult, bwResults []*network.BandwidthResult, latencyMap map[string]float64, cfg *model.Config) []string {
	logger := config.GetLogger()

	if len(bwResults) == 0 {
		logger.Sugar().Warnf("带宽测速经 %d 轮尝试后仍无有效结果，已降级使用 TCP 排序节点", cfg.Bandwidth.Retry)
		return selectByTCP(tcpResults, cfg)
	}

	selected := selectByBandwidth(bwResults, cfg)
	printFinalNodes(selected, bwResults, latencyMap, logger)
	return selected
}

// selectByTCP 按 TCP 排序结果选取最终节点（带宽测速失败时的降级方案）
func selectByTCP(tcpResults []*network.ProbeResult, cfg *model.Config) []string {
	if cfg.Global.Enable {
		limit := min(cfg.Global.TopN, len(tcpResults))
		selected := make([]string, 0, limit)
		for _, r := range tcpResults[:limit] {
			selected = append(selected, r.Node)
		}
		return selected
	}

	countryNodes := make(map[string][]*network.ProbeResult)
	for _, r := range tcpResults {
		countryNodes[r.Country] = append(countryNodes[r.Country], r)
	}
	var selected []string
	for _, nodes := range countryNodes {
		sort.Slice(nodes, func(i, j int) bool {
			if nodes[i].Success != nodes[j].Success {
				return nodes[i].Success > nodes[j].Success
			}
			return nodes[i].Latency < nodes[j].Latency
		})
		limit := min(cfg.Global.PerCountryTopN, len(nodes))
		for _, r := range nodes[:limit] {
			selected = append(selected, r.Node)
		}
	}
	return selected
}

// selectByBandwidth 按带宽测速结果选取最终节点
func selectByBandwidth(bwResults []*network.BandwidthResult, cfg *model.Config) []string {
	if cfg.Global.Enable {
		limit := min(cfg.Global.TopN, len(bwResults))
		selected := make([]string, 0, limit)
		for _, r := range bwResults[:limit] {
			selected = append(selected, r.Node)
		}
		return selected
	}

	// 分国家模式
	countrySpeedNodes := make(map[string][]*network.BandwidthResult)
	for _, r := range bwResults {
		country := utils.GetCountryFromNode(r.Node)
		if country != "" {
			countrySpeedNodes[country] = append(countrySpeedNodes[country], r)
		}
	}
	var selected []string
	for _, nodes := range countrySpeedNodes {
		limit := min(cfg.Global.PerCountryTopN, len(nodes))
		for _, r := range nodes[:limit] {
			selected = append(selected, r.Node)
		}
	}
	// 跨国家按速度降序
	speedMap := make(map[string]float64, len(bwResults))
	for _, r := range bwResults {
		speedMap[r.Node] = r.Speed
	}
	sort.Slice(selected, func(i, j int) bool {
		return speedMap[selected[i]] > speedMap[selected[j]]
	})
	return selected
}

// printFinalNodes 打印最终优选节点
func printFinalNodes(finalSelected []string, bwResults []*network.BandwidthResult, latencyMap map[string]float64, logger *zap.Logger) {
	logger.Sugar().Info("================ 最终优选节点 ================")
	speedMap := make(map[string]float64, len(bwResults))
	for _, r := range bwResults {
		speedMap[r.Node] = r.Speed
	}
	for i, node := range finalSelected {
		speed := speedMap[node]
		lat := latencyMap[node]
		if lat > 0 {
			logger.Sugar().Infof("%d. %s 速度 %.2f Mbps 延迟 %.2f ms", i+1, node, speed, lat*1000)
		} else {
			logger.Sugar().Infof("%d. %s 速度 %.2f Mbps", i+1, node, speed)
		}
	}
}

// ────────── 文件输出 ──────────

// WriteOutput 将最终节点写入输出文件
func WriteOutput(nodes []string, outputPath string) {
	logger := config.GetLogger()
	if outputPath == "" {
		logger.Warn("输出文件路径为空，跳过写入")
		return
	}

	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			logger.Sugar().Errorf("无法创建输出目录 %s: %v", outputDir, err)
			return
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		logger.Sugar().Errorf("无法创建文件 %s: %v", outputPath, err)
		return
	}
	defer file.Close()

	for _, node := range nodes {
		file.WriteString(node + "\n")
	}
}
