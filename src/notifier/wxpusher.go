package notifier

import (
	"bytes"
	"cfx/src/config"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// WxPusherMessage WxPusher 消息结构
type WxPusherMessage struct {
	AppToken string   `json:"appToken"`
	Content  string   `json:"content"`
	Summary  string   `json:"summary"`
	UIDs     []string `json:"uids"`
}

// SendWxPusherNotification 发送 WxPusher 通知
func SendWxPusherNotification(cfg *config.Config, content, summary string) error {
	if !cfg.EnableWxpusher {
		return nil
	}

	message := WxPusherMessage{
		AppToken: cfg.WxpusherAppToken,
		Content:  content,
		Summary:  summary,
		UIDs:     cfg.WxpusherUids,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	client := &http.Client{
		Timeout: time.Duration(cfg.NotifyTimeout) * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Duration(cfg.NotifyConnectTimeout) * time.Second,
			}).DialContext,
		},
	}

	resp, err := client.Post(cfg.WxpusherApiUrl, "application/json; charset=utf-8", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API 返回状态码: %d", resp.StatusCode)
	}

	zap.L().Info("微信通知已发送")
	return nil
}
