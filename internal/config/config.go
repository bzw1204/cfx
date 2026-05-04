package config

import (
	"cfx/internal/model"
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// LoadConfig 加载配置文件并返回配置结构体
// configPath 配置文件路径
// return 配置文件
func LoadConfig(configPath string) (*model.Config, error) {
	config := &model.Config{}
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	if configPath != "" {
		v.AddConfigPath(configPath)
	}
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalf("配置文件不存在: %s", err)
		}
		return nil, fmt.Errorf("无法读取配置文件: %w", err)
	}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("配置文件序列化失败: %w", err)
	}
	return config, nil
}
