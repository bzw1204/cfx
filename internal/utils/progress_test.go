package utils

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestProgressManager(t *testing.T) {
	logger := zap.NewNop()
	pm := NewProgressManager(logger)

	taskID := "test_task"
	label := "测试任务"
	total := 100

	// 测试进度更新
	pm.UpdateProgress(taskID, label, 0, total, "")

	// 验证任务存在
	task, exists := pm.GetTaskProgress(taskID)
	if !exists {
		t.Errorf("任务应该存在")
	}

	if task.Label != label {
		t.Errorf("期望标签 %s, 实际 %s", label, task.Label)
	}

	if task.Total != total {
		t.Errorf("期望总数 %d, 实际 %d", total, task.Total)
	}

	// 测试进度更新
	pm.UpdateProgress(taskID, label, 50, total, "进行中")
	task, _ = pm.GetTaskProgress(taskID)
	if task.Completed != 50 {
		t.Errorf("期望完成数 50, 实际 %d", task.Completed)
	}

	// 测试任务完成
	pm.CompleteTask(taskID)
	_, exists = pm.GetTaskProgress(taskID)
	if exists {
		t.Errorf("任务应该已被删除")
	}
}

func TestProgressManager_MultipleTasks(t *testing.T) {
	logger := zap.NewNop()
	pm := NewProgressManager(logger)

	// 添加多个任务
	pm.UpdateProgress("task1", "任务1", 10, 100, "")
	pm.UpdateProgress("task2", "任务2", 20, 200, "")
	pm.UpdateProgress("task3", "任务3", 30, 300, "")

	// 验证所有任务
	allTasks := pm.GetAllTasks()
	if len(allTasks) != 3 {
		t.Errorf("期望 3 个任务, 实际 %d", len(allTasks))
	}

	// 完成一个任务
	pm.CompleteTask("task2")
	allTasks = pm.GetAllTasks()
	if len(allTasks) != 2 {
		t.Errorf("期望 2 个任务, 实际 %d", len(allTasks))
	}

	// 清除所有任务
	pm.ClearAllTasks()
	allTasks = pm.GetAllTasks()
	if len(allTasks) != 0 {
		t.Errorf("期望 0 个任务, 实际 %d", len(allTasks))
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1536, "1.5 KB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %s; 期望 %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{500 * time.Millisecond, "500ms"},
		{5 * time.Second, "5.0s"},
		{90 * time.Second, "1.5m"},
		{2 * time.Minute, "2.0m"},
	}

	for _, tt := range tests {
		result := FormatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("FormatDuration(%v) = %s; 期望 %s", tt.duration, result, tt.expected)
		}
	}
}