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

	if warnings := validateConfig(config); len(warnings) > 0 {
		for _, w := range warnings {
			log.Printf("⚠️ 配置警告: %s", w)
		}
	}
	return config, nil
}

// defaultConfig 返回默认配置
func defaultConfig() *model.Config {
	return &model.Config{
		Global: model.Global{
			Enable:              true,
			TopN:                50,
			PerCountryTopN:      50,
			BandwidthCandidates: 50,
			FallbackWorkers:     500,
			OutputFile:          "./data/ip.txt",
		},
		Tcp: model.TCP{
			Probes:         2,
			MinSuccessRate: 0.5,
			SocketTimeout:  5,
			NetworkTestConfig: model.NetworkTestConfig{
				Timeout:        5,
				ConnectTimeout: 5,
				Retry:          1,
				RetryDelay:     1,
				MaxWorkers:     300,
			},
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
		Availability: model.Availability{
			Enabled:          true,
			CheckApi:         "https://api.090227.xyz/check",
			IPV6Availability: false,
			NetworkTestConfig: model.NetworkTestConfig{
				Timeout:        3,
				ConnectTimeout: 5,
				Retry:          2,
				RetryDelay:     3,
				MaxWorkers:     10,
			},
		},
		Bandwidth: model.Bandwidth{
			Enabled:       true,
			SizeMB:        50,
			UrlTemplate:   "https://speed.cloudflare.com/__down?bytes={bytes}",
			ProcessBuffer: 2,
			NetworkTestConfig: model.NetworkTestConfig{
				Timeout:        5,
				ConnectTimeout: 3,
				Retry:          2,
				RetryDelay:     3,
				MaxWorkers:     10,
			},
		},
		Logger: model.Logger{
			Enabled:       true,
			Level:         "info",
			Format:        "json",
			Output:        "cfnb.log",
			ConsoleOutput: true,
			FileOutput:    true,
			LogDir:        "./logs",
			MaxFileSize:   100,
			MaxBackups:    7,
			MaxAge:        30,
			Compress:      true,
		},
		Http: model.Http{
			Enabled: true,
			Port:    8080,
			Path:    "/ips",
		},
		Schedule: model.Schedule{
			Enabled:  false,
			Interval: "1h",
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
	v.Set("availability", config.Availability)
	v.Set("bandwidth", config.Bandwidth)
	v.Set("logger", config.Logger)
	v.Set("http", config.Http)
	v.Set("schedule", config.Schedule)

	if err := v.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}

// validateConfig 检查配置中的不合理字段，返回警告列表
func validateConfig(cfg *model.Config) []string {
	var warnings []string

	if cfg.Global.OutputFile == "" {
		warnings = append(warnings, "Global.OutputFile 为空，结果将无法写入文件")
	}
	if cfg.Global.TopN <= 0 {
		warnings = append(warnings, "Global.TopN 必须 > 0")
	}
	if cfg.Global.PerCountryTopN <= 0 {
		warnings = append(warnings, "Global.PerCountryTopN 必须 > 0")
	}
	if cfg.Global.BandwidthCandidates <= 0 {
		warnings = append(warnings, "Global.BandwidthCandidates 必须 > 0")
	}
	if cfg.Bandwidth.SizeMB <= 0 {
		warnings = append(warnings, "Bandwidth.SizeMB 必须 > 0，否则测速 URL 会下载 0 字节")
	}
	if cfg.Tcp.Probes <= 0 {
		warnings = append(warnings, "Tcp.Probes 必须 > 0")
	}
	if cfg.Tcp.Timeout <= 0 {
		warnings = append(warnings, "Tcp.Timeout 必须 > 0")
	}
	if cfg.Tcp.MaxWorkers <= 0 {
		warnings = append(warnings, "Tcp.MaxWorkers 必须 > 0")
	}
	if cfg.Availability.Retry <= 0 {
		warnings = append(warnings, "Availability.Retry 必须 > 0")
	}
	if cfg.Bandwidth.Retry <= 0 {
		warnings = append(warnings, "Bandwidth.Retry 必须 > 0")
	}
	if cfg.Http.Enabled && cfg.Http.Port <= 0 {
		warnings = append(warnings, "Http.Port 必须 > 0")
	}
	if cfg.Schedule.Enabled && cfg.Schedule.Interval == "" {
		warnings = append(warnings, "Schedule.Interval 不能为空")
	}
	return warnings
}
