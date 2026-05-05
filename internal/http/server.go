package http

import (
	"cfx/internal/config"
	"cfx/internal/model"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// StartServer 启动 HTTP 服务，返回错误通道用于监听启动错误
func StartServer(cfg *model.Config) error {
	logger := config.GetLogger()
	addr := fmt.Sprintf(":%d", cfg.Http.Port)
	path := cfg.Http.Path
	if path == "" {
		path = "/ips"
	}

	// 注册路由
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		handleIPRequest(w, r, cfg)
	})

	// 可选：添加健康检查端点
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	logger.Sugar().Infof("HTTP 服务启动，监听 %s，IP 列表路径: %s", addr, path)
	return http.ListenAndServe(addr, nil)
}

// handleIPRequest 处理 /ips 请求，返回 output_file 的内容
func handleIPRequest(w http.ResponseWriter, r *http.Request, cfg *model.Config) {
	logger := config.GetLogger()
	outputFile := cfg.Global.OutputFile

	// 处理相对路径
	if !filepath.IsAbs(outputFile) {
		absPath, err := filepath.Abs(outputFile)
		if err == nil {
			outputFile = absPath
		}
	}

	// 读取文件内容
	data, err := os.ReadFile(outputFile)
	if err != nil {
		logger.Sugar().Warnf("读取输出文件失败: %v", err)
		http.Error(w, "文件不存在或无法读取", http.StatusNotFound)
		data = []byte("")
	}

	// 设置响应头并返回内容
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)

	logger.Sugar().Debugf("成功返回 IP 列表，长度: %d 字节", len(data))
}
