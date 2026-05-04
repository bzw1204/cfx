package config

import (
	"cfx/internal/model"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// LoadConfig 加载配置文件并返回配置结构体
// configPath 配置文件路径
// return 配置文件
func LoadConfig(configPath string) (*model.Config, error) {
	config := defaultConfig()
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	if configPath != "" {
		v.AddConfigPath(configPath)
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("配置文件不存在，正在创建默认配置...")
			if writeErr := writeDefaultConfig(v, config, configPath); writeErr != nil {
				return nil, fmt.Errorf("创建默认配置文件失败: %w", writeErr)
			}
			log.Println("默认配置文件已生成，请根据需要修改后重新运行")
			return config, nil
		}
		return nil, fmt.Errorf("无法读取配置文件: %w", err)
	}

	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("配置文件序列化失败: %w", err)
	}
	return config, nil
}

// defaultConfig 返回默认配置
func defaultConfig() *model.Config {
	return &model.Config{
		Global: model.Global{
			Enable:              true,
			TopN:                10,
			PerCountryTopN:      10,
			BandwidthCandidates: 90,
			FallbackWorkers:     100,
			OutputFile:          "./data/ip.txt",
		},
		Tcp: model.TCP{
			Probes:         2,
			Timeout:        5,
			MinSuccessRate: 0.5,
			SocketTimeout:  5,
			MaxWorkers:     300,
		},
		Filter: model.Filter{
			CountriesEnabled: false,
			AllowedCountries: []string{"US"},
			BlockedEnabled:   true,
			BlockedCountries: []string{"CN"},
			PortEnabled:      true,
			Ports:            []int{443},
		},
		Cloudflare: model.Cloudflare{
			Enabled:           false,
			ApiToken:          "your_api_token",
			ZoneId:            "your_zone_id",
			DnsRecordName:     "your_dns_record_name",
			Ttl:               60,
			Proxied:           false,
			DnsConnectTimeout: 3,
			DnsReadTimeout:    3,
		},
		Dns: model.DNS{
			Enabled:                false,
			FilterIpv6Availability: true,
			BlockedCountries:       []string{},
			UpdateTargetCount:      15,
			UpdateMaxRetries:       3,
			UpdateRetryDelay:       3,
		},
		AdditionalSources: []model.AdditionalSources{
			{Enabled: true, Url: "https://zip.cm.edu.kg/all.txt"},
			{Enabled: true, Url: "https://countrymerge.pages.dev/all.txt"},
			{Enabled: true, Url: "https://wtf-359.pages.dev/wtf.txt"},
		},
		Node: model.Node{
			Timeout:        5,
			Retry:          3,
			Retries:        3,
			ConnectTimeout: 5,
		},
		Availability: model.Availability{
			Enabled:          true,
			CheckApi:         "https://api.090227.xyz/check",
			Timeout:          3,
			ConnectTimeout:   5,
			Retry:            2,
			RetryDelay:       3,
			MaxWorkers:       10,
			IPV6Availability: false,
		},
		Bandwidth: model.Bandwidth{
			Enabled:        true,
			SizeMB:         0.5,
			Timeout:        5,
			Retry:          2,
			RetryDelay:     3,
			UrlTemplate:    "https://speed.cloudflare.com/__down?bytes={bytes}",
			ProcessBuffer:  2,
			ConnectTimeout: 3,
			MaxWorkers:     10,
		},
		Logger: model.Logger{
			Enabled: false,
			Level:   "info",
			Format:  "json",
			Output:  "cfnb.log",
		},
	}
}

// writeDefaultConfig 将默认配置写入文件
func writeDefaultConfig(v *viper.Viper, config *model.Config, configPath string) error {
	// 确定 config 文件所在目录
	configDir := "."
	if configPath != "" {
		configDir = configPath
	}
	configFile := filepath.Join(configDir, "config.yaml")

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 将默认配置写入 viper 并持久化
	v.Set("global", config.Global)
	v.Set("tcp", config.Tcp)
	v.Set("filter", config.Filter)
	v.Set("cloudflare", config.Cloudflare)
	v.Set("dns", config.Dns)
	v.Set("additional_sources", config.AdditionalSources)
	v.Set("node", config.Node)
	v.Set("availability", config.Availability)
	v.Set("bandwidth", config.Bandwidth)
	v.Set("logger", config.Logger)

	if err := v.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}
