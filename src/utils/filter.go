package utils

import (
	"strings"
)

// FilterByPort 按端口过滤节点
func FilterByPort(nodes []string, allowedPorts []string) []string {
	var filtered []string
	portMap := make(map[string]bool)
	for _, port := range allowedPorts {
		portMap[port] = true
	}

	for _, node := range nodes {
		port := GetPortFromNode(node)
		if portMap[port] {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// FilterByBlockedCountries 按黑名单国家过滤节点
func FilterByBlockedCountries(nodes []string, blockedCountries []string) []string {
	blockedMap := make(map[string]bool)
	for _, country := range blockedCountries {
		blockedMap[strings.ToUpper(country)] = true
	}

	var filtered []string
	for _, node := range nodes {
		country := GetCountryFromNode(node)
		if !blockedMap[strings.ToUpper(country)] {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// FilterByAllowedCountries 按白名单国家过滤节点
func FilterByAllowedCountries(nodes []string, allowedCountries []string) []string {
	allowedMap := make(map[string]bool)
	for _, country := range allowedCountries {
		allowedMap[strings.ToUpper(country)] = true
	}

	var filtered []string
	for _, node := range nodes {
		country := GetCountryFromNode(node)
		if allowedMap[strings.ToUpper(country)] {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// DeduplicateNodes 去重节点（基于 IP:Port）
func DeduplicateNodes(nodes []string) []string {
	seen := make(map[string]bool)
	var unique []string

	for _, node := range nodes {
		key := strings.Split(node, "#")[0] // ip:port
		if !seen[key] {
			seen[key] = true
			unique = append(unique, node)
		}
	}

	return unique
}
