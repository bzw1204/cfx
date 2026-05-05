package config

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestNewLogManager(t *testing.T) {
	// 测试控制台输出
	cfg := LogConfig{
		ConsoleOutput: true,
		FileOutput:    false,
		LogLevel:      "info",
	}

	logMgr, err := NewLogManager(cfg)
	if err != nil {
		t.Errorf("创建日志管理器失败: %v", err)
	}

	if logMgr == nil {
		t.Error("日志管理器不应该为nil")
	}

	logger := logMgr.GetLogger()
	if logger == nil {
		t.Error("Logger不应该为nil")
	}

	sugar := logMgr.GetSugar()
	if sugar == nil {
		t.Error("Sugared logger不应该为nil")
	}

	logMgr.Sync()
}

func TestNewLogManager_FileOutput(t *testing.T) {
	t.Cleanup(func() {
		// 清理测试目录
		os.RemoveAll("./test_logs")
	})

	cfg := LogConfig{
		ConsoleOutput: false,
		FileOutput:    true,
		LogDir:        "./test_logs",
		LogLevel:      "info",
		MaxFileSize:   1,
		MaxBackups:    2,
		MaxAge:        1,
		Compress:      false,
	}

	logMgr, err := NewLogManager(cfg)
	if err != nil {
		t.Errorf("创建日志管理器失败: %v", err)
	}

	// 验证目录创建
	if _, err := os.Stat(cfg.LogDir); os.IsNotExist(err) {
		t.Errorf("日志目录应该被创建")
	}

	// 写入一些日志
	logger := logMgr.GetLogger()
	logger.Info("测试日志消息")

	// 检查日志文件
	infoLogPath := filepath.Join(cfg.LogDir, "cfx.info.log")
	if _, err := os.Stat(infoLogPath); os.IsNotExist(err) {
		t.Errorf("info日志文件应该存在")
	}

	logMgr.Sync()
}

func TestNewLogManager_LogLevel(t *testing.T) {
	t.Cleanup(func() {
		os.RemoveAll("./test_logs_debug")
	})

	// 测试不同的日志级别
	levels := []string{"debug", "info", "warn", "error", "invalid"}

	for _, level := range levels {
		cfg := LogConfig{
			ConsoleOutput: true,
			FileOutput:    true,
			LogDir:        "./test_logs_debug",
			LogLevel:      level,
		}

		logMgr, err := NewLogManager(cfg)
		if err != nil {
			t.Errorf("级别 %s 创建日志管理器失败: %v", level, err)
		}

		if logMgr == nil {
			t.Errorf("级别 %s 日志管理器不应该为nil", level)
		}

		// 写入测试日志
		logger := logMgr.GetLogger()
		logger.Debug("debug消息")
		logger.Info("info消息")
		logger.Warn("warn消息")
		logger.Error("error消息")

		logMgr.Sync()
	}
}

func TestLogManager_WithFields(t *testing.T) {
	cfg := LogConfig{
		ConsoleOutput: true,
		LogLevel:      "info",
	}

	logMgr, err := NewLogManager(cfg)
	if err != nil {
		t.Errorf("创建日志管理器失败: %v", err)
	}

	fields := map[string]interface{}{
		"user_id":   123,
		"username":  "testuser",
		"active":    true,
		"duration":  5 * 60 * 1000, // 5分钟
	}

	logger := logMgr.WithFields(fields)
	if logger == nil {
		t.Error("带字段的logger不应该为nil")
	}

	// 写入带字段的日志
	logger.Info("带字段的测试消息")

	logMgr.Sync()
}

func TestLogManager_LogProgress(t *testing.T) {
	cfg := LogConfig{
		ConsoleOutput: true,
		LogLevel:      "info",
	}

	logMgr, err := NewLogManager(cfg)
	if err != nil {
		t.Errorf("创建日志管理器失败: %v", err)
	}

	// 测试进度日志
	logMgr.LogProgress("test_task", "测试任务", 50, 100, "进度良好")

	logMgr.Sync()
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"warning", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"panic", zapcore.PanicLevel},
		{"fatal", zapcore.FatalLevel},
		{"invalid", zapcore.InfoLevel}, // 默认级别
		{"", zapcore.InfoLevel},        // 空字符串默认级别
	}

	for _, tt := range tests {
		result := parseLogLevel(tt.input)
		if result != tt.expected {
			t.Errorf("parseLogLevel(%q) = %v; 期望 %v", tt.input, result, tt.expected)
		}
	}
}