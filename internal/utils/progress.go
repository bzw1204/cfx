package utils

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ProgressManager 管理多个任务的进度显示
// ProgressManager manages progress display for multiple tasks
type ProgressManager struct {
	logger     *zap.Logger
	isTerminal bool
	tasks      map[string]*ProgressTask
	mu         sync.RWMutex
	lastRender time.Time
}

// ProgressTask 表示单个任务的进度
// ProgressTask represents progress for a single task
type ProgressTask struct {
	Label     string        // 任务标签 / Task label
	Completed int           // 已完成数量 / Completed count
	Total     int           // 总数量 / Total count
	Extra     string        // 额外信息 / Extra information
	StartTime time.Time     // 开始时间 / Start time
	TaskID    string        // 任务ID / Task ID
}

// NewProgressManager 创建新的进度管理器
// NewProgressManager creates a new progress manager
func NewProgressManager(logger *zap.Logger) *ProgressManager {
	return &ProgressManager{
		logger:     logger,
		isTerminal: isTerminal(),
		tasks:      make(map[string]*ProgressTask),
		lastRender: time.Now(),
	}
}

// UpdateProgress 更新任务进度
// UpdateProgress updates task progress
func (pm *ProgressManager) UpdateProgress(taskID, label string, completed, total int, extra string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	task := &ProgressTask{
		Label:     label,
		Completed: completed,
		Total:     total,
		Extra:     extra,
		StartTime: time.Now(),
		TaskID:    taskID,
	}

	// 如果任务已存在，保持原有的开始时间
	// If task exists, keep the original start time
	if existing, exists := pm.tasks[taskID]; exists {
		task.StartTime = existing.StartTime
	}

	pm.tasks[taskID] = task

	// 记录到日志
	// Log to file
	pm.logger.Info("progress_update",
		zap.String("task_id", taskID),
		zap.String("label", label),
		zap.Int("completed", completed),
		zap.Int("total", total),
		zap.Float64("percentage", float64(completed)/float64(total)*100),
		zap.String("extra", extra),
	)

	// 终端显示（限制刷新频率避免闪烁）
	// Terminal display (limit refresh rate to avoid flickering)
	if pm.isTerminal && time.Since(pm.lastRender) > 100*time.Millisecond {
		pm.renderTerminal()
		pm.lastRender = time.Now()
	}
}

// CompleteTask 标记任务完成
// CompleteTask marks a task as completed
func (pm *ProgressManager) CompleteTask(taskID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if task, exists := pm.tasks[taskID]; exists {
		duration := time.Since(task.StartTime)
		pm.logger.Info("task_completed",
			zap.String("task_id", taskID),
			zap.String("label", task.Label),
			zap.Duration("duration", duration),
			zap.Int("total_items", task.Total),
		)
		delete(pm.tasks, taskID)
	}

	if pm.isTerminal {
		if len(pm.tasks) == 0 {
			fmt.Println("\033[32m✓ 所有任务完成\033[0m")
		} else {
			pm.renderTerminal()
		}
	}
}

// renderTerminal 渲染终端进度显示
// renderTerminal renders progress display in terminal
func (pm *ProgressManager) renderTerminal() {
	var lines []string

	for _, task := range pm.tasks {
		percent := 0.0
		if task.Total > 0 {
			percent = float64(task.Completed) / float64(task.Total) * 100
		}

		// 进度条可视化
		// Visual progress bar
		barWidth := 20
		filled := int(percent / 100 * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		// 计算运行时间
		// Calculate running time
		duration := time.Since(task.StartTime).Round(time.Second)

		// 构建进度行
		// Build progress line
		line := fmt.Sprintf("\033[36m[%s]\033[0m %s \033[32m%d/%d\033[0m (\033[33m%.1f%%\033[0m) \033[90m[%s]\033[0m",
			task.Label, bar, task.Completed, task.Total, percent, duration)

		// 添加额外信息
		// Add extra information
		if task.Extra != "" {
			line += " \033[35m" + task.Extra + "\033[0m"
		}

		lines = append(lines, line)
	}

	// 清屏并重新输出（使用 \033[2J 清屏，\033[H 移动光标到左上角）
	// Clear screen and re-render (use \033[2J to clear, \033[H to move cursor to top-left)
	fmt.Print("\033[2J\033[H")
	for _, line := range lines {
		fmt.Println(line)
	}
}

// GetTaskProgress 获取任务进度信息
// GetTaskProgress gets task progress information
func (pm *ProgressManager) GetTaskProgress(taskID string) (*ProgressTask, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	task, exists := pm.tasks[taskID]
	if !exists {
		return nil, false
	}

	// 返回副本避免并发问题
	// Return copy to avoid concurrency issues
	return &ProgressTask{
		Label:     task.Label,
		Completed: task.Completed,
		Total:     task.Total,
		Extra:     task.Extra,
		StartTime: task.StartTime,
		TaskID:    task.TaskID,
	}, true
}

// GetAllTasks 获取所有任务信息
// GetAllTasks gets all task information
func (pm *ProgressManager) GetAllTasks() map[string]*ProgressTask {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 返回副本避免并发问题
	// Return copy to avoid concurrency issues
	result := make(map[string]*ProgressTask)
	for id, task := range pm.tasks {
		result[id] = &ProgressTask{
			Label:     task.Label,
			Completed: task.Completed,
			Total:     task.Total,
			Extra:     task.Extra,
			StartTime: task.StartTime,
			TaskID:    task.TaskID,
		}
	}
	return result
}

// ClearAllTasks 清除所有任务
// ClearAllTasks clears all tasks
func (pm *ProgressManager) ClearAllTasks() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.tasks = make(map[string]*ProgressTask)
	if pm.isTerminal {
		fmt.Print("\033[2J\033[H") // 清屏 / Clear screen
	}
}

// isTerminal 检查是否在终端环境中运行
// isTerminal checks if running in terminal environment
func isTerminal() bool {
	// 检查是否在 CI 环境或强制禁用颜色
	// Check if in CI environment or colors are forced disabled
	if os.Getenv("TERM") == "dumb" || os.Getenv("NO_COLOR") != "" {
		return false
	}

	// 检查输出是否是终端
	// Check if output is a terminal
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// 检查是否是字符设备（终端）
	// Check if it's a character device (terminal)
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// FormatBytes 格式化字节数为易读格式
// FormatBytes formats bytes to human readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration 格式化时间间隔为易读格式
// FormatDuration formats duration to human readable format
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}