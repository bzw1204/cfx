package main

import (
	"cfx/internal"
	"cfx/internal/config"
	"cfx/internal/dns"
	"cfx/internal/utils"
	"fmt"
	"os"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("")
	if err != nil {
		fmt.Printf("加载配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger := config.InitLogger(true, "./cfx.log")
	defer logger.Sync()

	// 打印配置概要
	internal.PrintConfigInfo(cfg)

	// ── Phase 1: 获取节点 ──
	allNodes := internal.FetchNodes(cfg)
	if len(allNodes) == 0 {
		logger.Sugar().Fatal("没有获取到任何有效节点")
	}

	// ── Phase 2: 前置过滤 ──
	allNodes = internal.PortFilter(allNodes, cfg)
	allNodes = internal.BlockedFilter(allNodes, cfg)
	allNodes = internal.WhiteListFilter(allNodes, cfg)

	// ── Phase 3: TCP 连通性测试 ──
	tcpResults := internal.TCPTry(allNodes, cfg)

	// ── Phase 4: 选取带宽测速候选节点 ──
	candidates, latencyMap := internal.SelectBandwidthCandidates(tcpResults, cfg)
	if len(candidates) == 0 {
		logger.Sugar().Fatal("没有候选节点")
	}

	// ── Phase 5: 可用性二次检测 ──
	candidatesAfterAvail, availIPInfo := internal.CheckAvailabilityWithRetry(candidates, cfg)

	// ── Phase 6: 带宽测速 ──
	bwResults := internal.MeasureBandwidthWithRetry(candidatesAfterAvail, cfg)

	// ── Phase 7: 选取最终节点 ──
	finalSelected := internal.SelectFinalNodes(tcpResults, bwResults, latencyMap, cfg)

	// ── Phase 8: 写入输出文件 ──
	internal.WriteOutput(finalSelected, cfg.Global.OutputFile)
	logger.Sugar().Infof("结果已保存到 %s（共 %d 个节点）", cfg.Global.OutputFile, len(finalSelected))

	// ── Phase 9: Cloudflare DNS 更新 ──
	ipList := make([]string, 0, len(finalSelected))
	for _, node := range finalSelected {
		ipList = append(ipList, utils.GetIPFromNode(node))
	}
	dns.BatchUpdateCloudflareDNS(cfg, ipList, bwResults, availIPInfo, latencyMap)
}
