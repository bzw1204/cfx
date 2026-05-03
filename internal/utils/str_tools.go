package utils

import "fmt"

// Bool2Str 布尔值转字符串
// b: 布尔值，trueStr: true时返回的字符串，falseStr: false时返回的字符串
// 例如：Bool2Str(true, "已连接", "未连接") 返回 "已连接"
// return: 根据布尔值返回对应的字符串
func Bool2Str(b bool, trueStr, falseStr string) string {
	if b {
		return trueStr
	}
	return falseStr
}

// PrintProgress 打印进度
// label: 进度标签，completed: 已完成数量，total: 总数量，extra: 额外的字符串，例如错误信息
// 例如：PrintProgress("下载", 3, 5, "文件已存在")
func PrintProgress(label string, completed, total int, extra string) {
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
