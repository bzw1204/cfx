package utils

import (
	"cfx/src/constants"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Node 表示一个节点
type Node struct {
	IP      string
	Port    string
	Country string
	Raw     string // 原始字符串 ip:port#country
}

var (
	nodePattern     = regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+):(\d+)#(.+)$`)
	ipPortPattern   = regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+):(\d+)#`)
	cnNamePattern   = regexp.MustCompile(`^([\x{4e00}-\x{9fff}\x{ff08}\x{ff09}()]+)\d*$`)
	emojiFlagRange  = regexp.MustCompile(`[\x{1F1E6}-\x{1F1FF}]`)
	tokenNoiseRegex = regexp.MustCompile(`^[\d\s\-_.|#]+`)
)

// ExtractCountryCode 从标签中提取国家代码
func ExtractCountryCode(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}

	// 分割标签
	tokens := regexp.MustCompile(`[\s,;|/]+`).Split(label, -1)

	// 1. 优先找标准两位大写字母代码
	for _, token := range tokens {
		tokenCleaned := tokenNoiseRegex.ReplaceAllString(strings.TrimSpace(token), "")
		if matched, _ := regexp.MatchString(`^[A-Z]{2}$`, tokenCleaned); matched {
			return tokenCleaned
		}
	}

	// 2. 尝试提取中文名
	for _, token := range tokens {
		tokenCleaned := tokenNoiseRegex.ReplaceAllString(token, "")
		tokenNoEmoji := emojiFlagRange.ReplaceAllString(tokenCleaned, "")
		tokenNoEmoji = strings.TrimSpace(tokenNoEmoji)

		matches := cnNamePattern.FindStringSubmatch(tokenNoEmoji)
		if len(matches) > 1 {
			cnName := strings.TrimSpace(matches[1])
			if code, ok := constants.CN_TO_CODE[cnName]; ok {
				return code
			}
		}
	}

	// 3. 解码纯 emoji 国旗
	emojiChars := emojiFlagRange.FindAllString(label, -1)
	if len(emojiChars) >= 2 && len(emojiChars)%2 == 0 {
		first := int([]rune(emojiChars[0])[0]) - 0x1F1E6
		second := int([]rune(emojiChars[1])[0]) - 0x1F1E6
		if first >= 0 && first <= 25 && second >= 0 && second <= 25 {
			return string(rune(first+'A')) + string(rune(second+'A'))
		}
	}

	return ""
}

// ParseNode 解析单个节点字符串
func ParseNode(nodeStr string) (*Node, error) {
	matches := nodePattern.FindStringSubmatch(nodeStr)
	if len(matches) != 4 {
		return nil, fmt.Errorf("无效的节点格式: %s", nodeStr)
	}

	return &Node{
		IP:      matches[1],
		Port:    matches[2],
		Country: matches[3],
		Raw:     nodeStr,
	}, nil
}

// ParseTextNodes 从纯文本中解析节点
func ParseTextNodes(text string) []string {
	var nodes []string
	lines := strings.FieldsSeq(text)

	for token := range lines {
		if !strings.Contains(token, "#") {
			continue
		}

		parts := strings.SplitN(token, "#", 2)
		if len(parts) != 2 {
			continue
		}

		ipPort := strings.TrimSpace(parts[0])
		label := strings.TrimSpace(parts[1])

		// 跳过 IPv6
		if strings.HasPrefix(ipPort, "[") {
			continue
		}

		// 验证 IP:Port 格式
		if !regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+:\d+$`).MatchString(ipPort) {
			continue
		}

		// 提取国家代码
		code := ExtractCountryCode(label)
		if code != "" {
			nodes = append(nodes, fmt.Sprintf("%s#%s", ipPort, code))
		}
	}

	return nodes
}

// ParseJSONNodes 从 JSON 结构中递归提取节点
func ParseJSONNodes(data interface{}) []string {
	var nodes []string

	switch v := data.(type) {
	case []any:
		for _, item := range v {
			nodes = append(nodes, ParseJSONNodes(item)...)
		}
	case map[string]interface{}:
		// 查找常见字段
		for _, key := range []string{"nodes", "data", "result", "list"} {
			if val, ok := v[key]; ok {
				if arr, ok := val.([]interface{}); ok {
					nodes = append(nodes, ParseJSONNodes(arr)...)
					break
				}
			}
		}

		// 尝试直接提取节点信息
		ip, _ := v["ip"].(string)
		if ip == "" {
			ip, _ = v["host"].(string)
		}
		portVal, hasPort := v["port"]
		code, _ := v["country"].(string)
		if code == "" {
			code, _ = v["cc"].(string)
		}

		if ip != "" && hasPort && code != "" {
			port := fmt.Sprintf("%v", portVal)
			nodes = append(nodes, fmt.Sprintf("%s:%s#%s", ip, port, strings.ToUpper(code)))
		}
	case string:
		nodes = append(nodes, ParseTextNodes(v)...)
	}

	return nodes
}

// ParseAdaptive 自适应解析任意格式的节点列表
func ParseAdaptive(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	// 尝试 JSON 解析
	if strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
		var data interface{}
		if err := json.Unmarshal([]byte(text), &data); err == nil {
			return ParseJSONNodes(data)
		}
	}

	// 回退到文本解析
	return ParseTextNodes(text)
}

// GetIPFromNode 从节点字符串中提取 IP
func GetIPFromNode(nodeStr string) string {
	if idx := strings.Index(nodeStr, ":"); idx != -1 {
		return nodeStr[:idx]
	}
	return nodeStr
}

// GetPortFromNode 从节点字符串中提取端口
func GetPortFromNode(nodeStr string) string {
	if _, after, ok := strings.Cut(nodeStr, ":"); ok {
		rest := after
		if hashIdx := strings.Index(rest, "#"); hashIdx != -1 {
			return rest[:hashIdx]
		}
		return rest
	}
	return ""
}

// GetCountryFromNode 从节点字符串中提取国家代码
func GetCountryFromNode(nodeStr string) string {
	if idx := strings.LastIndex(nodeStr, "#"); idx != -1 {
		return nodeStr[idx+1:]
	}
	return ""
}
