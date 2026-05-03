package main

import (
	"cfx/src/config"
	"cfx/src/dns"
	"cfx/src/github"
	"cfx/src/network"
	"cfx/src/notifier"
	"cfx/src/utils"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

func main() {
	// 先加载配置
	cfgPath := config.GetConfigFilePath()
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		fmt.Printf("加载配置文件失败: %v\n", err)
		os.Exit(1)
	}
	// 根据配置初始化日志
	logger := config.InitLogger(true, cfg.LogFile)
	defer logger.Sync()

	// 打印配置信息
	modeStr := "全局最优%d个"
	if !cfg.UseGlobalMode {
		modeStr = "每个国家最优%d个"
	}
	logger.Sugar().Infof("当前模式："+modeStr+"，每个节点测试 %d 次 TCP 连接", cfg.GlobalTopN, cfg.TcpProbes)
	logger.Sugar().Infof("最低成功率要求：%.0f%%", cfg.MinSuccessRate*100)
	logger.Sugar().Infof("IP 可用性二次筛选：%s（仅对候选节点）", boolToStr(cfg.TestAvailability, "启用", "禁用"))
	logger.Sugar().Infof("IPv6 客户端 IP 过滤（仅作用于DNS更新环节）：%s", boolToStr(cfg.FilterIpv6Availability, "启用", "禁用"))
	logger.Sugar().Infof("DNS黑名单过滤：%s，黑名单国家：%s", boolToStr(cfg.FilterBlockedCountriesEnabled, "启用", "禁用"), strings.Join(cfg.BlockedCountries, ", "))
	logger.Sugar().Infof("带宽测速候选数：%d，测速文件大小：%.1f MB，超时：%ds", cfg.BandwidthCandidates, cfg.BandwidthSizeMB, cfg.BandwidthTimeout)
	if cfg.FilterCountriesEnabled {
		logger.Sugar().Infof("前置白名单过滤：启用，仅保留：%s", strings.Join(cfg.AllowedCountries, ", "))
	}

	// 从数据源获取节点
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

	logger.Sugar().Infof("合并后总计 %d 个节点", len(allNodes))

	if len(allNodes) == 0 {
		logger.Sugar().Fatal("没有获取到任何有效节点")
	}

	// 前置端口过滤
	if cfg.PreFilterPortEnabled {
		before := len(allNodes)
		var ports []string
		for _, p := range cfg.PreFilterPorts {
			ports = append(ports, fmt.Sprintf("%d", p))
		}
		allNodes = utils.FilterByPort(allNodes, ports)
		after := len(allNodes)
		logger.Sugar().Infof("前置端口过滤（仅保留端口 %s）：%d -> %d 个节点", strings.Join(ports, ", "), before, after)
		if len(allNodes) == 0 {
			logger.Sugar().Fatal("前置端口过滤后无任何节点")
		}
	}

	// 前置黑名单过滤
	if cfg.PreFilterBlockedEnabled && len(cfg.PreFilterBlockedCountries) > 0 {
		before := len(allNodes)
		allNodes = utils.FilterByBlockedCountries(allNodes, cfg.PreFilterBlockedCountries)
		after := len(allNodes)
		logger.Sugar().Infof("前置黑名单过滤：%d -> %d 个节点（已屏蔽：%s）", before, after, strings.Join(cfg.PreFilterBlockedCountries, ", "))
		if len(allNodes) == 0 {
			zap.L().Fatal("前置黑名单过滤后无任何节点")
		}
	}

	// 白名单过滤
	if cfg.FilterCountriesEnabled && len(cfg.AllowedCountries) > 0 {
		before := len(allNodes)
		allNodes = utils.FilterByAllowedCountries(allNodes, cfg.AllowedCountries)
		after := len(allNodes)
		logger.Sugar().Infof("国家过滤（测试前）：%d -> %d 个节点（允许国家：%s）", before, after, strings.Join(cfg.AllowedCountries, ", "))
		if len(allNodes) == 0 {
			zap.L().Fatal("过滤后无任何节点")
		}
	}

	// TCP 测试
	logger.Sugar().Infof("开始 TCP 连接测试（超时 %.1fs，并发 %d）", cfg.Timeout, cfg.MaxWorkers)
	total := len(allNodes)
	completed := 0
	lastPrint := time.Now()

	var tcpResults []*network.TCPResult
	sem := make(chan struct{}, cfg.MaxWorkers)
	resultChan := make(chan *network.TCPResult, total)

	for _, node := range allNodes {
		go func(nodeStr string) {
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := network.TestTCPNode(nodeStr, cfg.Timeout, cfg.TcpProbes)
			if err != nil {
				resultChan <- nil
				return
			}
			resultChan <- result
		}(node)
	}

	for completed < total {
		result := <-resultChan
		if result != nil {
			tcpResults = append(tcpResults, result)
		}

		completed++
		now := time.Now()
		if now.Sub(lastPrint).Seconds() >= float64(cfg.ProgressPrintInterval) || completed == total {
			printProgress("TCP 测试", completed, total, "")
			lastPrint = now
		}
	}
	zap.L().Info("TCP 测试完成")

	if len(tcpResults) == 0 {
		zap.L().Fatal("没有通过成功率筛选的节点，请检查网络或降低 MIN_SUCCESS_RATE")
	}

	// 按成功率和延迟排序
	sort.Slice(tcpResults, func(i, j int) bool {
		if tcpResults[i].Success != tcpResults[j].Success {
			return tcpResults[i].Success > tcpResults[j].Success
		}
		return tcpResults[i].Latency < tcpResults[j].Latency
	})

	// 构建延迟映射
	latencyMap := make(map[string]float64)
	for _, r := range tcpResults {
		latencyMap[r.Node] = r.Latency
	}

	// 选择候选节点
	var candidates []string
	if cfg.UseGlobalMode {
		limit := min(cfg.BandwidthCandidates, len(tcpResults))
		for _, r := range tcpResults[:limit] {
			candidates = append(candidates, r.Node)
		}
		logger.Sugar().Infof("TCP 最优前 %d 个节点进入候选池", len(candidates))
	} else {
		// 分国家模式
		countryNodes := make(map[string][]*network.TCPResult)
		for _, r := range tcpResults {
			countryNodes[r.Country] = append(countryNodes[r.Country], r)
		}

		totalCountries := len(countryNodes)
		baseLimit := cfg.BandwidthCandidates / totalCountries
		if baseLimit < 1 {
			baseLimit = 1
		}

		for _, nodes := range countryNodes {
			// 按成功率和延迟排序
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

	if len(candidates) == 0 {
		zap.L().Fatal("没有候选节点")
	}

	// 可用性检测
	var candidatesAfterAvail []string
	var availIPInfo map[string]string
	for attempt := 1; attempt <= cfg.AvailabilityRetryMax; attempt++ {
		logger.Sugar().Infof("[可用性检测] 第 %d 轮检测", attempt)

		completed := 0
		total := len(candidates)
		lastPrint := time.Now()

		var availResults []*network.AvailabilityResult
		sem := make(chan struct{}, cfg.AvailabilityWorkers)
		resultChan := make(chan *network.AvailabilityResult, total)

		for _, node := range candidates {
			go func(nodeStr string) {
				sem <- struct{}{}
				defer func() { <-sem }()

				result, _ := network.CheckAvailability(nodeStr, cfg.AvailabilityCheckApi, cfg.AvailabilityConnectTimeout, cfg.AvailabilityTimeout)
				resultChan <- result
			}(node)
		}

		passed := 0
		for completed < total {
			result := <-resultChan
			availResults = append(availResults, result)
			if result.Available {
				passed++
			}

			completed++
			now := time.Now()
			if now.Sub(lastPrint).Seconds() >= float64(cfg.ProgressPrintInterval) || completed == total {
				printProgress("[可用性检测]", completed, total, fmt.Sprintf("通过数量：%d", passed))
				lastPrint = now
			}
		}

		// 收集通过的节点
		candidatesAfterAvail = []string{}
		availIPInfo = make(map[string]string)
		for _, r := range availResults {
			if r.Available {
				candidatesAfterAvail = append(candidatesAfterAvail, r.Node)
				availIPInfo[r.Node] = r.Stack
			}
		}
		if len(candidatesAfterAvail) > 0 {
			logger.Sugar().Infof("可用性检测通过 %d 个节点", len(candidatesAfterAvail))
			break
		}

		if attempt < cfg.AvailabilityRetryMax {
			zap.L().Warn(fmt.Sprintf("本轮可用性检测通过率为 0%%，等待 %d 秒后重试", cfg.AvailabilityRetryDelay))
			time.Sleep(time.Duration(cfg.AvailabilityRetryDelay) * time.Second)
		}
	}

	if len(candidatesAfterAvail) == 0 {
		zap.L().Error(fmt.Sprintf("可用性检测经 %d 轮重试后仍无节点通过", cfg.AvailabilityRetryMax))
		notifier.SendWxPusherNotification(cfg,
			fmt.Sprintf("IP 可用性检测经 %d 轮重试后仍无节点通过，已跳过过滤，使用原候选列表继续", cfg.AvailabilityRetryMax),
			"可用性检测全部失败")
		candidatesAfterAvail = candidates
	}

	// 带宽测速
	var bwResults []*network.BandwidthResult
	bandwidthURL := fmt.Sprintf("https://speed.cloudflare.com/__down?bytes=%d", int(cfg.BandwidthSizeMB*1024*1024))

	for attempt := 1; attempt <= cfg.BandwidthRetryMax; attempt++ {
		logger.Sugar().Infof("[带宽测速] 第 %d 轮测试", attempt)

		completed := 0
		total := len(candidatesAfterAvail)
		lastPrint := time.Now()

		sem := make(chan struct{}, cfg.BandwidthWorkers)
		resultChan := make(chan *network.BandwidthResult, total)

		for _, node := range candidatesAfterAvail {
			go func(nodeStr string) {
				sem <- struct{}{}
				defer func() { <-sem }()

				result, _ := network.MeasureBandwidth(nodeStr, bandwidthURL, cfg.BandwidthConnectTimeout, cfg.BandwidthTimeout)
				resultChan <- result
			}(node)
		}

		for completed < total {
			result := <-resultChan
			if result.Speed > 0 {
				bwResults = append(bwResults, result)
			}

			completed++
			now := time.Now()
			if now.Sub(lastPrint).Seconds() >= float64(cfg.ProgressPrintInterval) || completed == total {
				printProgress("[带宽测速]", completed, total, "")
				lastPrint = now
			}
		}

		if len(bwResults) > 0 {
			break
		}

		if attempt < cfg.BandwidthRetryMax {
			zap.L().Warn(fmt.Sprintf("本轮测速无有效结果，等待 %d 秒后重试", cfg.BandwidthRetryDelay))
			time.Sleep(time.Duration(cfg.BandwidthRetryDelay) * time.Second)
		}
	}

	// 选择最终节点
	var finalSelected []string
	if len(bwResults) == 0 {
		zap.L().Warn("带宽测速多次重试仍无有效结果，将使用 TCP 筛选结果作为最终节点")
		notifier.SendWxPusherNotification(cfg,
			fmt.Sprintf("带宽测速经 %d 轮尝试后仍无有效结果，已降级使用 TCP 排序节点", cfg.BandwidthRetryMax),
			"带宽测速全部失败")

		if cfg.UseGlobalMode {
			limit := cfg.GlobalTopN
			if limit > len(tcpResults) {
				limit = len(tcpResults)
			}
			for _, r := range tcpResults[:limit] {
				finalSelected = append(finalSelected, r.Node)
			}
		} else {
			countryNodes := make(map[string][]*network.TCPResult)
			for _, r := range tcpResults {
				countryNodes[r.Country] = append(countryNodes[r.Country], r)
			}
			for _, nodes := range countryNodes {
				sort.Slice(nodes, func(i, j int) bool {
					if nodes[i].Success != nodes[j].Success {
						return nodes[i].Success > nodes[j].Success
					}
					return nodes[i].Latency < nodes[j].Latency
				})
				limit := cfg.PerCountryTopN
				if limit > len(nodes) {
					limit = len(nodes)
				}
				for _, r := range nodes[:limit] {
					finalSelected = append(finalSelected, r.Node)
				}
			}
		}
	} else {
		if cfg.UseGlobalMode {
			limit := cfg.GlobalTopN
			if limit > len(bwResults) {
				limit = len(bwResults)
			}
			for _, r := range bwResults[:limit] {
				finalSelected = append(finalSelected, r.Node)
			}
		} else {
			countrySpeedNodes := make(map[string][]*network.BandwidthResult)
			for _, r := range bwResults {
				country := utils.GetCountryFromNode(r.Node)
				if country != "" {
					countrySpeedNodes[country] = append(countrySpeedNodes[country], r)
				}
			}
			for _, nodes := range countrySpeedNodes {
				limit := cfg.PerCountryTopN
				if limit > len(nodes) {
					limit = len(nodes)
				}
				for _, r := range nodes[:limit] {
					finalSelected = append(finalSelected, r.Node)
				}
			}
			// 按速度排序
			speedMap := make(map[string]float64)
			for _, r := range bwResults {
				speedMap[r.Node] = r.Speed
			}
			sort.Slice(finalSelected, func(i, j int) bool {
				return speedMap[finalSelected[i]] > speedMap[finalSelected[j]]
			})
		}

		zap.L().Info("================ 最终优选节点 ================")
		speedMap := make(map[string]float64)
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

	// 写入 ip.txt
	writeIPTxt(finalSelected, cfg)
	logger.Sugar().Infof("结果已保存到 %s（共 %d 个节点）", cfg.OutputFile, len(finalSelected))

	// 提取 IP 列表
	var ipList []string
	for _, node := range finalSelected {
		ipList = append(ipList, utils.GetIPFromNode(node))
	}

	// 更新 Cloudflare DNS
	dns.BatchUpdateCloudflareDNS(cfg, ipList, bwResults, availIPInfo, latencyMap)

	// 同步到 GitHub
	github.SyncToGitHub(cfg)
}

// writeIPTxt 写入 ip.txt 文件
func writeIPTxt(nodes []string, cfg *config.Config) {
	file, err := os.Create(cfg.OutputFile)
	if err != nil {
		zap.L().Error(fmt.Sprintf("无法创建文件 %s: %v", cfg.OutputFile, err))
		return
	}
	defer file.Close()

	// 写入头部广告
	if cfg.AdHeaderEnabled {
		for _, line := range cfg.AdHeaderLines {
			file.WriteString(line + "\n")
		}
	}

	// 写入节点
	for _, node := range nodes {
		if cfg.AdPerlineEnabled && cfg.AdPerlineText != "" {
			file.WriteString(fmt.Sprintf("%s%s\n", node, cfg.AdPerlineText))
		} else {
			file.WriteString(node + "\n")
		}
	}

	// 写入尾部广告
	if cfg.AdFooterEnabled {
		for _, line := range cfg.AdFooterLines {
			file.WriteString(line + "\n")
		}
	}
}

// boolToStr 布尔值转字符串
func boolToStr(b bool, trueStr, falseStr string) string {
	if b {
		return trueStr
	}
	return falseStr
}

func printProgress(label string, completed, total int, extra string) {
	percent := 0.0
	if total > 0 {
		percent = float64(completed) / float64(total) * 100
	}

	line := fmt.Sprintf("%s 进度：%d/%d (%.1f%%)", label, completed, total, percent)
	if extra != "" {
		line += " " + extra
	}

	// 使用 \r 回车 + \033[2K 清除整行，实现同一行覆盖输出
	fmt.Printf("\r\033[2K\033[36m%s\033[0m", line)
}
