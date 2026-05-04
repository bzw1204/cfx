package internal

import (
	"cfx/internal/config"
	"cfx/internal/model"
	"cfx/internal/network"
	"cfx/internal/utils"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
)

// 前置端口过滤
// nodes: 所有节点
// cfg: 配置信息
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

// 前置黑名单过滤
// nodes: 所有节点
// cfg: 配置信息
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

// 白名单过滤
// nodes: 所有节点
// cfg: 配置信息
// 返回值: 过滤后的节点
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

// TCP 测试
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
