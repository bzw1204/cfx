package github

import (
	"cfx/src/config"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// SyncToGitHub 同步到 GitHub
func SyncToGitHub(cfg *config.Config) error {
	scriptDir, err := os.Getwd()
	if err != nil {
		zap.L().Warn("未找到 git_sync.sh，跳过 GitHub 同步")
		return nil
	}

	scriptPath := filepath.Join(scriptDir, "git_sync.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		zap.L().Warn("未找到 git_sync.sh，跳过 GitHub 同步")
		return nil
	}

	// 设置执行权限
	os.Chmod(scriptPath, 0755)

	maxRetries := cfg.GithubSyncMaxRetries
	retryDelay := cfg.GithubSyncRetryDelay
	processTimeout := time.Duration(cfg.GitSyncProcessTimeout) * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		zap.L().Info(fmt.Sprintf("正在同步到 GitHub (尝试 %d/%d)", attempt, maxRetries))

		cmd := exec.Command("bash", scriptPath)

		// 设置超时
		done := make(chan error, 1)
		go func() {
			done <- cmd.Run()
		}()

		select {
		case err := <-done:
			if err != nil {
				zap.L().Error(fmt.Sprintf("推送失败 (退出码 %v)", cmd.ProcessState.ExitCode()))
				if exitErr, ok := err.(*exec.ExitError); ok {
					zap.L().Error(fmt.Sprintf("错误信息: %s", string(exitErr.Stderr)))
				}
			} else {
				zap.L().Info("已自动推送到 GitHub")
				return nil
			}
		case <-time.After(processTimeout):
			cmd.Process.Kill()
			zap.L().Warn(fmt.Sprintf("推送超时（超过 %v 秒）", cfg.GitSyncProcessTimeout))
		}

		if attempt < maxRetries {
			zap.L().Info(fmt.Sprintf("等待 %d 秒后重试", retryDelay))
			time.Sleep(time.Duration(retryDelay) * time.Second)
		}
	}

	return fmt.Errorf("GitHub 同步失败，已重试 %d 次", maxRetries)
}
