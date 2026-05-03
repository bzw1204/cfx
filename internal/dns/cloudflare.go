package dns

import (
	"bytes"
	"cfx/internal/config"
	"cfx/internal/network"
	"cfx/internal/notifier"
	"cfx/internal/utils"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// CloudflareDNSRecord Cloudflare DNS 记录
type CloudflareDNSRecord struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

// CloudflareBatchUpdate Cloudflare 批量更新请求
type CloudflareBatchUpdate struct {
	Deletes []any                 `json:"deletes"`
	Posts   []CloudflareDNSRecord `json:"posts"`
}

// CloudflareResponse Cloudflare API 响应
type CloudflareResponse struct {
	Success bool  `json:"success"`
	Errors  []any `json:"errors"`
	Result  any   `json:"result"`
}

// BatchUpdateCloudflareDNS 批量更新 Cloudflare DNS
func BatchUpdateCloudflareDNS(cfg *config.Config, ipList []string, bwResults []*network.BandwidthResult, ipInfo map[string]string, latencyMap map[string]float64) error {
	if !cfg.CfEnabled {
		zap.L().Debug("Cloudflare DNS 批量更新未启用")
		return nil
	}

	targetCount := cfg.DnsUpdateTargetCount
	if targetCount == 0 {
		targetCount = 15
	}

	// 筛选用于 DNS 更新的 IP
	var dnsIPList []string
	var dnsNodeList []string
	filteredByPort := 0
	filteredByIPv6 := 0
	filteredByCountry := 0

	if len(bwResults) > 0 && len(ipInfo) > 0 {
		blockedSet := make(map[string]bool)
		if cfg.FilterBlockedCountriesEnabled {
			for _, c := range cfg.BlockedCountries {
				blockedSet[strings.ToUpper(c)] = true
			}
		}

		for _, br := range bwResults {
			nodeStr := br.Node

			// 端口过滤
			port := utils.GetPortFromNode(nodeStr)
			if port != "443" {
				filteredByPort++
				continue
			}

			// IPv6 过滤
			if cfg.FilterIpv6Availability {
				stack := ipInfo[nodeStr]
				if stack == "ipv6_only" {
					filteredByIPv6++
					continue
				}
			}

			// 国家黑名单过滤
			if len(blockedSet) > 0 {
				country := utils.GetCountryFromNode(nodeStr)
				if blockedSet[strings.ToUpper(country)] {
					filteredByCountry++
					continue
				}
			}

			pureIP := utils.GetIPFromNode(nodeStr)
			dnsIPList = append(dnsIPList, pureIP)
			dnsNodeList = append(dnsNodeList, nodeStr)

			if len(dnsIPList) >= targetCount {
				break
			}
		}

		// 打印过滤信息
		var filterParts []string
		if filteredByPort > 0 {
			filterParts = append(filterParts, fmt.Sprintf("非443端口过滤(%d个)", filteredByPort))
		}
		if cfg.FilterIpv6Availability {
			filterParts = append(filterParts, fmt.Sprintf("IPv6落地过滤(%d个)", filteredByIPv6))
		}
		if cfg.FilterBlockedCountriesEnabled {
			filterParts = append(filterParts, fmt.Sprintf("DNS黑名单过滤(%d个)", filteredByCountry))
		}
		filterStr := "无过滤"
		if len(filterParts) > 0 {
			filterStr = strings.Join(filterParts, " + ")
		}
		zap.L().Info(fmt.Sprintf("从 %d 个测速节点中筛选出 %d 个节点用于 DNS 更新（%s）", len(bwResults), len(dnsIPList), filterStr))
	}

	// 降级处理
	if len(dnsIPList) == 0 {
		if len(ipList) > 0 {
			zap.L().Warn("未能从完整测速结果构建 DNS 列表，降级使用 ip.txt 中的 IP")
			dnsIPList = ipList
			dnsNodeList = ipList
		} else {
			msg := "没有可用的 IP 用于 DNS 更新，跳过"
			zap.L().Info(msg)
			notifier.SendWxPusherNotification(cfg, msg, "DNS 更新跳过")
			return nil
		}
	}

	// 去重
	seen := make(map[string]bool)
	var uniqueIPs []string
	var uniqueNodes []string
	for i, ip := range dnsIPList {
		if !seen[ip] {
			seen[ip] = true
			uniqueIPs = append(uniqueIPs, ip)
			uniqueNodes = append(uniqueNodes, dnsNodeList[i])
		}
	}
	dnsIPList = uniqueIPs
	dnsNodeList = uniqueNodes

	zap.L().Info(fmt.Sprintf("准备将以下 %d 个 IP 批量更新到 Cloudflare DNS", len(dnsIPList)))
	speedMap := make(map[string]float64)
	for _, br := range bwResults {
		speedMap[br.Node] = br.Speed
	}
	for i, node := range dnsNodeList {
		speed := speedMap[node]
		latency := latencyMap[node]
		if latency > 0 {
			zap.L().Info(fmt.Sprintf("%d. %s 速度 %.2f Mbps 延迟 %.2f ms", i+1, node, speed, latency*1000))
		} else {
			zap.L().Info(fmt.Sprintf("%d. %s 速度 %.2f Mbps", i+1, dnsIPList[i], speed))
		}
	}

	// 构建 HTTP 客户端
	client := &http.Client{
		Timeout: time.Duration(cfg.CfDnsReadTimeout) * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Duration(cfg.CfDnsConnectTimeout) * time.Second,
			}).DialContext,
		},
	}

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", cfg.CfApiToken),
		"Content-Type":  "application/json",
	}

	maxRetries := cfg.DnsUpdateMaxRetries
	retryDelay := cfg.DnsUpdateRetryDelay

	for attempt := 1; attempt <= maxRetries; attempt++ {
		zap.L().Info(fmt.Sprintf("[DNS 更新] 尝试 %d/%d", attempt, maxRetries))

		// 查询现有记录
		listURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=A&name=%s",
			cfg.CfZoneId, cfg.CfDnsRecordName)

		req, _ := http.NewRequest("GET", listURL, nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			zap.L().Error(fmt.Sprintf("[尝试 %d/%d] DNS 更新出错: %v", attempt, maxRetries, err))
			if attempt < maxRetries {
				time.Sleep(time.Duration(retryDelay) * time.Second)
				continue
			}
			finalError := fmt.Sprintf("Cloudflare DNS 更新失败，已重试 %d 次，错误：%v", maxRetries, err)
			zap.L().Error(finalError)
			notifier.SendWxPusherNotification(cfg, finalError, "DNS 更新失败")
			return err
		}

		var listResp CloudflareResponse
		json.NewDecoder(resp.Body).Decode(&listResp)
		resp.Body.Close()

		if !listResp.Success {
			err := fmt.Errorf("查询 DNS 记录失败: %v", listResp.Errors)
			zap.L().Error(fmt.Sprintf("[尝试 %d/%d] DNS 更新出错: %v", attempt, maxRetries, err))
			if attempt < maxRetries {
				time.Sleep(time.Duration(retryDelay) * time.Second)
				continue
			}
			finalError := fmt.Sprintf("Cloudflare DNS 更新失败，已重试 %d 次，错误：%v", maxRetries, err)
			zap.L().Error(finalError)
			notifier.SendWxPusherNotification(cfg, finalError, "DNS 更新失败")
			return err
		}

		// 构建批量更新请求
		existingRecords, _ := listResp.Result.([]interface{})
		deletes := make([]interface{}, len(existingRecords))
		for i, rec := range existingRecords {
			if r, ok := rec.(map[string]interface{}); ok {
				deletes[i] = map[string]string{"id": r["id"].(string)}
			}
		}

		posts := make([]CloudflareDNSRecord, len(dnsIPList))
		for i, ip := range dnsIPList {
			posts[i] = CloudflareDNSRecord{
				Name:    cfg.CfDnsRecordName,
				Type:    "A",
				Content: ip,
				TTL:     cfg.CfTtl,
				Proxied: cfg.CfProxied,
			}
		}

		batchPayload := CloudflareBatchUpdate{
			Deletes: deletes,
			Posts:   posts,
		}

		jsonData, _ := json.Marshal(batchPayload)
		batchURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/batch", cfg.CfZoneId)

		req, _ = http.NewRequest("POST", batchURL, bytes.NewBuffer(jsonData))
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err = client.Do(req)
		if err != nil {
			zap.L().Error(fmt.Sprintf("[尝试 %d/%d] DNS 更新出错: %v", attempt, maxRetries, err))
			if attempt < maxRetries {
				time.Sleep(time.Duration(retryDelay) * time.Second)
				continue
			}
			finalError := fmt.Sprintf("Cloudflare DNS 更新失败，已重试 %d 次，错误：%v", maxRetries, err)
			zap.L().Error(finalError)
			notifier.SendWxPusherNotification(cfg, finalError, "DNS 更新失败")
			return err
		}

		var batchResp CloudflareResponse
		json.NewDecoder(resp.Body).Decode(&batchResp)
		resp.Body.Close()

		if !batchResp.Success {
			err := fmt.Errorf("批量更新失败: %v", batchResp.Errors)
			zap.L().Error(fmt.Sprintf("[尝试 %d/%d] DNS 更新出错: %v", attempt, maxRetries, err))
			if attempt < maxRetries {
				time.Sleep(time.Duration(retryDelay) * time.Second)
				continue
			}
			finalError := fmt.Sprintf("Cloudflare DNS 更新失败，已重试 %d 次，错误：%v", maxRetries, err)
			zap.L().Error(finalError)
			notifier.SendWxPusherNotification(cfg, finalError, "DNS 更新失败")
			return err
		}

		successMsg := fmt.Sprintf("Cloudflare DNS 批量更新成功！已将 %s 指向 %d 个 IP", cfg.CfDnsRecordName, len(dnsIPList))
		zap.L().Info(successMsg)
		zap.L().Info("注意：DNS 解析将随机返回这些 IP 中的一个，实现负载均衡")
		return nil
	}

	return nil
}
