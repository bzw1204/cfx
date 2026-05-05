package main

import (
	"cfx/internal"
	"cfx/internal/config"
	"cfx/internal/dns"
	"cfx/internal/http"
	"cfx/internal/model"
	"cfx/internal/utils"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("")
	if err != nil {
		fmt.Printf("加载配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化增强日志管理器
	logManager, err := config.NewLogManager(config.LogConfig{
		ConsoleOutput: cfg.Logger.ConsoleOutput,
		FileOutput:    cfg.Logger.FileOutput,
		LogDir:        cfg.Logger.LogDir,
		LogLevel:      cfg.Logger.Level,
		MaxFileSize:   cfg.Logger.MaxFileSize,
		MaxBackups:    cfg.Logger.MaxBackups,
		MaxAge:        cfg.Logger.MaxAge,
		Compress:      cfg.Logger.Compress,
	})
	if err != nil {
		fmt.Printf("初始化日志管理器失败: %v\n", err)
		os.Exit(1)
	}
	defer logManager.Sync()

	// 设置全局logger用于兼容
	logger := logManager.GetLogger()
	config.SetGlobalLogger(logger)

	// 打印配置概要
	internal.PrintConfigInfo(cfg)

	// 启动 HTTP 服务（如果启用）
	if cfg.Http.Enabled {
		go func() {
			if err := http.StartServer(cfg); err != nil {
				logger.Sugar().Fatalf("HTTP 服务启动失败: %v", err)
			}
		}()
	}

	// 初始化进度管理器
	progressMgr := utils.NewProgressManager(logger)

	// 如果启用定时任务，则按间隔执行；否则单次执行
	if cfg.Schedule.Enabled && cfg.Schedule.Interval != "" {
		runWithSchedule(cfg, progressMgr)
	} else {
		runOnce(cfg, progressMgr)
	}

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	logger.Sugar().Info("收到退出信号，正在关闭...")
}

// runOnce 执行一次完整流程
func runOnce(cfg *model.Config, progressMgr *utils.ProgressManager) {
	logger := config.GetLogger()

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
	tcpResults := internal.TCPTry(allNodes, cfg, progressMgr)

	// ── Phase 4: 选取带宽测速候选节点 ──
	candidates, latencyMap := internal.SelectBandwidthCandidates(tcpResults, cfg)
	if len(candidates) == 0 {
		logger.Sugar().Fatal("没有候选节点")
	}

	// ── Phase 5: 可用性二次检测 ──
	candidatesAfterAvail, availIPInfo := internal.CheckAvailabilityWithRetry(candidates, cfg, progressMgr)

	// ── Phase 6: 带宽测速 ──
	bwResults := internal.MeasureBandwidthWithRetry(candidatesAfterAvail, cfg, progressMgr)

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

// runWithSchedule 按定时任务执行
func runWithSchedule(cfg *model.Config, progressMgr *utils.ProgressManager) {
	logger := config.GetLogger()
	logger.Sugar().Infof("定时任务已启用，间隔: %s", cfg.Schedule.Interval)

	// 解析间隔（支持 duration 或 cron 表达式）
	interval, err := parseInterval(cfg.Schedule.Interval)
	if err != nil {
		logger.Sugar().Fatalf("无效的定时任务间隔: %v", err)
	}

	// 立即执行一次
	logger.Sugar().Info("执行首次任务...")
	runOnce(cfg, progressMgr)

	// 定时执行
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Sugar().Infof("定时任务已启动，每 %v 执行一次", interval)
	for range ticker.C {
		logger.Sugar().Info("定时任务触发，开始执行...")
		runOnce(cfg, progressMgr)
	}
}

// parseInterval 解析间隔配置，支持 duration (如 "1h") 和 cron 表达式
func parseInterval(intervalStr string) (time.Duration, error) {
	// 简单解析 duration
	return time.ParseDuration(intervalStr)
}
