package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogManager struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
	config LogConfig
}

type LogConfig struct {
	ConsoleOutput bool   `json:"console_output" yaml:"console_output"` // 是否输出到控制台
	FileOutput    bool   `json:"file_output" yaml:"file_output"`       // 是否输出到文件
	LogDir        string `json:"log_dir" yaml:"log_dir"`               // 日志目录
	LogLevel      string `json:"log_level" yaml:"log_level"`           // 日志级别
	MaxFileSize   int    `json:"max_file_size" yaml:"max_file_size"`   // 单个文件最大大小(MB)
	MaxBackups    int    `json:"max_backups" yaml:"max_backups"`       // 最大备份文件数
	MaxAge        int    `json:"max_age" yaml:"max_age"`               // 最大保存天数
	Compress      bool   `json:"compress" yaml:"compress"`             // 是否压缩旧日志
}

// NewLogManager 创建新的日志管理器
func NewLogManager(cfg LogConfig) (*LogManager, error) {
	level := parseLogLevel(cfg.LogLevel)

	var cores []zapcore.Core

	// 控制台输出（人类友好格式）
	if cfg.ConsoleOutput {
		consoleCore := createConsoleCore(level)
		cores = append(cores, consoleCore)
	}

	// 文件输出（结构化 + 轮转）
	if cfg.FileOutput && cfg.LogDir != "" {
		// 确保日志目录存在
		if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
			return nil, fmt.Errorf("创建日志目录失败: %w", err)
		}

		// 不同级别分文件存储
		infoCore := createFileCore(zapcore.InfoLevel, "INFO", cfg)
		errorCore := createFileCore(zapcore.ErrorLevel, "ERROR", cfg)

		cores = append(cores, infoCore, errorCore)
	}

	// 如果没有任何输出配置，至少输出到 stdout
	if len(cores) == 0 {
		consoleCore := createConsoleCore(level)
		cores = append(cores, consoleCore)
	}

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &LogManager{
		logger: logger,
		sugar:  logger.Sugar(),
		config: cfg,
	}, nil
}

// createConsoleCore 创建控制台输出核心
func createConsoleCore(level zapcore.Level) zapcore.Core {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.TimeEncoderOfLayout("15:04:05"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
	}

	return zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		level,
	)
}

// createFileCore 创建文件输出核心
func createFileCore(level zapcore.Level, levelName string, cfg LogConfig) zapcore.Core {
	// 使用 lumberjack 实现日志轮转
	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(cfg.LogDir, fmt.Sprintf("cfx.%s.log", strings.ToLower(levelName))),
		MaxSize:    cfg.MaxFileSize, // MB
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,     // days
		Compress:   cfg.Compress,
		LocalTime:  true,
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
	}

	return zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(logFile),
		level,
	)
}

// parseLogLevel 解析日志级别字符串
func parseLogLevel(levelStr string) zapcore.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// GetLogger 获取标准 logger
func (lm *LogManager) GetLogger() *zap.Logger {
	return lm.logger
}

// GetSugar 获取 sugared logger
func (lm *LogManager) GetSugar() *zap.SugaredLogger {
	return lm.sugar
}

// Sync 同步日志
func (lm *LogManager) Sync() error {
	return lm.logger.Sync()
}

// LogProgress 记录进度信息到日志
func (lm *LogManager) LogProgress(taskID, operation string, completed, total int, extra string) {
	percentage := 0.0
	if total > 0 {
		percentage = float64(completed) / float64(total) * 100
	}

	lm.sugar.Infow("progress",
		"task_id", taskID,
		"operation", operation,
		"completed", completed,
		"total", total,
		"percentage", fmt.Sprintf("%.1f%%", percentage),
		"extra", extra,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// WithFields 创建带有字段的 logger
func (lm *LogManager) WithFields(fields map[string]interface{}) *zap.Logger {
	var f []zap.Field
	for k, v := range fields {
		switch val := v.(type) {
		case string:
			f = append(f, zap.String(k, val))
		case int:
			f = append(f, zap.Int(k, val))
		case bool:
			f = append(f, zap.Bool(k, val))
		case time.Duration:
			f = append(f, zap.Duration(k, val))
		default:
			f = append(f, zap.Any(k, val))
		}
	}
	return lm.logger.With(f...)
}

// SetGlobalLogger 设置全局logger用于向后兼容
func SetGlobalLogger(l *zap.Logger) {
	logger = l
	zap.ReplaceGlobals(l)
}