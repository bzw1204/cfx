package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

// isTerminal 判断终端是否支持颜色
func isTerminal() bool {
	// 检查是否在 CI 环境或强制禁用颜色
	if os.Getenv("TERM") == "dumb" || os.Getenv("NO_COLOR") != "" {
		return false
	}
	// 检查输出是否是终端
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	// 检查是否是字符设备（终端）
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// getLogLevel 根据环境变量获取日志级别
func getLogLevel() zapcore.Level {
	if os.Getenv("DEBUG") != "" {
		return zapcore.DebugLevel
	}
	return zapcore.InfoLevel
}

// InitLogger 初始化日志系统
// consoleOutput: 是否输出到终端
// logFilePath: 日志文件路径（为空则不写文件）
func InitLogger(consoleOutput bool, logFilePath string) *zap.Logger {
	level := getLogLevel()

	// 终端编码配置
	consoleEncoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
	}

	// 文件编码配置（JSON 格式）
	fileEncoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
	}

	var cores []zapcore.Core

	// 终端输出（人类可读格式）
	if consoleOutput {
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(consoleEncoderConfig),
			zapcore.AddSync(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	// 日志文件输出（JSON 格式）
	if logFilePath != "" {
		// 确保目录存在
		dir := filepath.Dir(logFilePath)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Printf("警告: 无法创建日志目录 %s: %v\n", dir, err)
			}
		}

		// 打开日志文件
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Printf("警告: 无法打开日志文件 %s: %v\n", logFilePath, err)
		} else {
			fileCore := zapcore.NewCore(
				zapcore.NewJSONEncoder(fileEncoderConfig),
				zapcore.AddSync(file),
				level,
			)
			cores = append(cores, fileCore)
		}
	}

	// 如果没有任何输出配置，至少输出到 stdout
	if len(cores) == 0 {
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(consoleEncoderConfig),
			zapcore.AddSync(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	// 组合所有 core
	core := zapcore.NewTee(cores...)

	logger = zap.New(core)
	zap.ReplaceGlobals(logger)
	logger.Info(fmt.Sprintf("日志初始化完成, 当前时间: %s", time.Now().Format("2006-01-02 15:04:05")))
	return logger
}

// GetLogger 获取全局 logger 实例
func GetLogger() *zap.Logger {
	return logger
}
