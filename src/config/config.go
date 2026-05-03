package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// 数据源配置
type AdditionalSource struct {
	Url     string `json:"url"`
	Enabled bool   `json:"enabled"`
}
type Config struct {
	// 全局模式配置
	UseGlobalMode       bool `json:"USE_GLOBAL_MODE"`
	GlobalTopN          int  `json:"GLOBAL_TOP_N"`
	PerCountryTopN      int  `json:"PER_COUNTRY_TOP_N"`
	BandwidthCandidates int  `json:"BANDWIDTH_CANDIDATES"`

	// TCP 测试配置
	TcpProbes            int     `json:"TCP_PROBES"`
	MinSuccessRate       float64 `json:"MIN_SUCCESS_RATE"`
	Timeout              float64 `json:"TIMEOUT"`
	SocketDefaultTimeout int     `json:"SOCKET_DEFAULT_TIMEOUT"`

	// 进度打印配置
	ProgressPrintInterval int `json:"PROGRESS_PRINT_INTERVAL"`

	// 国家过滤配置（白名单）
	FilterCountriesEnabled bool     `json:"FILTER_COUNTRIES_ENABLED"`
	AllowedCountries       []string `json:"ALLOWED_COUNTRIES"`

	// 国家过滤配置（黑名单 - 前置）
	PreFilterBlockedEnabled   bool     `json:"PRE_FILTER_BLOCKED_ENABLED"`
	PreFilterBlockedCountries []string `json:"PRE_FILTER_BLOCKED_COUNTRIES"`

	// 端口过滤配置
	PreFilterPortEnabled bool  `json:"PRE_FILTER_PORT_ENABLED"`
	PreFilterPorts       []int `json:"PRE_FILTER_PORTS"`

	// WxPusher 通知配置
	EnableWxpusher       bool     `json:"ENABLE_WXPUSHER"`
	WxpusherAppToken     string   `json:"WXPUSHER_APP_TOKEN"`
	WxpusherUids         []string `json:"WXPUSHER_UIDS"`
	WxpusherApiUrl       string   `json:"WXPUSHER_API_URL"`
	NotifyTimeout        int      `json:"NOTIFY_TIMEOUT"`
	NotifyConnectTimeout int      `json:"NOTIFY_CONNECT_TIMEOUT"`

	// Cloudflare DNS 更新配置
	CfEnabled           bool   `json:"CF_ENABLED"`
	CfApiToken          string `json:"CF_API_TOKEN"`
	CfZoneId            string `json:"CF_ZONE_ID"`
	CfDnsRecordName     string `json:"CF_DNS_RECORD_NAME"`
	CfTtl               int    `json:"CF_TTL"`
	CfProxied           bool   `json:"CF_PROXIED"`
	CfDnsConnectTimeout int    `json:"CF_DNS_CONNECT_TIMEOUT"`
	CfDnsReadTimeout    int    `json:"CF_DNS_READ_TIMEOUT"`

	AdditionalSources []AdditionalSource `json:"ADDITIONAL_SOURCES"`

	// 获取节点列表重试配置
	FetchMaxRetries     int `json:"FETCH_MAX_RETRIES"`
	FetchRetryDelay     int `json:"FETCH_RETRY_DELAY"`
	FetchTimeout        int `json:"FETCH_TIMEOUT"`
	FetchConnectTimeout int `json:"FETCH_CONNECT_TIMEOUT"`

	// 输出文件配置
	OutputFile string `json:"OUTPUT_FILE"`

	// 日志配置
	EnableLogging bool   `json:"ENABLE_LOGGING"`
	LogFile       string `json:"LOG_FILE"`

	// 可用性检测配置
	TestAvailability           bool   `json:"TEST_AVAILABILITY"`
	AvailabilityCheckApi       string `json:"AVAILABILITY_CHECK_API"`
	AvailabilityTimeout        int    `json:"AVAILABILITY_TIMEOUT"`
	AvailabilityConnectTimeout int    `json:"AVAILABILITY_CONNECT_TIMEOUT"`
	AvailabilityRetryMax       int    `json:"AVAILABILITY_RETRY_MAX"`
	AvailabilityRetryDelay     int    `json:"AVAILABILITY_RETRY_DELAY"`

	// DNS 相关过滤配置
	FilterIpv6Availability        bool     `json:"FILTER_IPV6_AVAILABILITY"`
	FilterBlockedCountriesEnabled bool     `json:"FILTER_BLOCKED_COUNTRIES_ENABLED"`
	BlockedCountries              []string `json:"BLOCKED_COUNTRIES"`
	DnsUpdateTargetCount          int      `json:"DNS_UPDATE_TARGET_COUNT"`

	// 带宽测速配置
	BandwidthSizeMB         float64 `json:"BANDWIDTH_SIZE_MB"`
	BandwidthTimeout        int     `json:"BANDWIDTH_TIMEOUT"`
	BandwidthRetryMax       int     `json:"BANDWIDTH_RETRY_MAX"`
	BandwidthRetryDelay     int     `json:"BANDWIDTH_RETRY_DELAY"`
	BandwidthUrlTemplate    string  `json:"BANDWIDTH_URL_TEMPLATE"`
	BandwidthProcessBuffer  int     `json:"BANDWIDTH_PROCESS_BUFFER"`
	BandwidthConnectTimeout int     `json:"BANDWIDTH_CONNECT_TIMEOUT"`

	// 并发工作线程配置
	MaxWorkers          int `json:"MAX_WORKERS"`
	AvailabilityWorkers int `json:"AVAILABILITY_WORKERS"`
	FallbackWorkers     int `json:"FALLBACK_WORKERS"`
	BandwidthWorkers    int `json:"BANDWIDTH_WORKERS"`

	// 重试策略配置
	DnsUpdateMaxRetries   int `json:"DNS_UPDATE_MAX_RETRIES"`
	DnsUpdateRetryDelay   int `json:"DNS_UPDATE_RETRY_DELAY"`
	GithubSyncMaxRetries  int `json:"GITHUB_SYNC_MAX_RETRIES"`
	GithubSyncRetryDelay  int `json:"GITHUB_SYNC_RETRY_DELAY"`
	GitSyncProcessTimeout int `json:"GIT_SYNC_PROCESS_TIMEOUT"`

	// 广告配置
	AdHeaderEnabled  bool     `json:"AD_HEADER_ENABLED"`
	AdHeaderLines    []string `json:"AD_HEADER_LINES"`
	AdFooterEnabled  bool     `json:"AD_FOOTER_ENABLED"`
	AdFooterLines    []string `json:"AD_FOOTER_LINES"`
	AdPerlineEnabled bool     `json:"AD_PERLINE_ENABLED"`
	AdPerlineText    string   `json:"AD_PERLINE_TEXT"`
}

// LoadConfig 加载配置文件，缺失字段使用默认值
func LoadConfig(configPath string) (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析 JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	setDefaults(&config)

	return &config, nil
}

// setDefaults 为配置设置默认值
func setDefaults(cfg *Config) {
	defaults := map[string]any{
		"USE_GLOBAL_MODE":                  true,
		"GLOBAL_TOP_N":                     15,
		"PER_COUNTRY_TOP_N":                1,
		"BANDWIDTH_CANDIDATES":             90,
		"TCP_PROBES":                       3,
		"MIN_SUCCESS_RATE":                 1.0,
		"TIMEOUT":                          2.0,
		"SOCKET_DEFAULT_TIMEOUT":           3,
		"PROGRESS_PRINT_INTERVAL":          1,
		"FILTER_COUNTRIES_ENABLED":         false,
		"ALLOWED_COUNTRIES":                []string{"US"},
		"PRE_FILTER_BLOCKED_ENABLED":       true,
		"PRE_FILTER_BLOCKED_COUNTRIES":     []string{"CN"},
		"PRE_FILTER_PORT_ENABLED":          true,
		"PRE_FILTER_PORTS":                 []int{443},
		"ENABLE_WXPUSHER":                  true,
		"WXPUSHER_APP_TOKEN":               "your_app_token_here",
		"WXPUSHER_UIDS":                    []string{"your_uid_here"},
		"WXPUSHER_API_URL":                 "http://wxpusher.zjiecode.com/api/send/message",
		"NOTIFY_TIMEOUT":                   3,
		"NOTIFY_CONNECT_TIMEOUT":           3,
		"CF_ENABLED":                       true,
		"CF_API_TOKEN":                     "your_CF_API_TOKEN",
		"CF_ZONE_ID":                       "your_CF_ZONE_ID",
		"CF_DNS_RECORD_NAME":               "your_CF_DNS_RECORD_NAME",
		"CF_TTL":                           60,
		"CF_PROXIED":                       false,
		"CF_DNS_CONNECT_TIMEOUT":           3,
		"CF_DNS_READ_TIMEOUT":              3,
		"FETCH_MAX_RETRIES":                3,
		"FETCH_RETRY_DELAY":                3,
		"FETCH_TIMEOUT":                    3,
		"FETCH_CONNECT_TIMEOUT":            3,
		"OUTPUT_FILE":                      "/app/data/ip.txt",
		"ENABLE_LOGGING":                   false,
		"LOG_FILE":                         "/app/data/cfnb.log",
		"TEST_AVAILABILITY":                true,
		"AVAILABILITY_CHECK_API":           "https://api.090227.xyz/check",
		"AVAILABILITY_TIMEOUT":             3,
		"AVAILABILITY_CONNECT_TIMEOUT":     3,
		"AVAILABILITY_RETRY_MAX":           2,
		"AVAILABILITY_RETRY_DELAY":         3,
		"FILTER_IPV6_AVAILABILITY":         true,
		"FILTER_BLOCKED_COUNTRIES_ENABLED": true,
		"BLOCKED_COUNTRIES": []string{
			"BD", "BI", "BY", "CD", "CF", "CN", "CU", "DE", "ET", "HK",
			"IR", "KP", "LY", "MO", "NG", "NL", "PK", "RU", "SD", "SO",
			"SY", "TH", "TW", "UA", "VE", "VN", "YE", "ZW",
		},
		"DNS_UPDATE_TARGET_COUNT":   15,
		"BANDWIDTH_SIZE_MB":         0.5,
		"BANDWIDTH_TIMEOUT":         3,
		"BANDWIDTH_RETRY_MAX":       2,
		"BANDWIDTH_RETRY_DELAY":     3,
		"BANDWIDTH_URL_TEMPLATE":    "https://speed.cloudflare.com/__down?bytes={bytes}",
		"BANDWIDTH_PROCESS_BUFFER":  2,
		"BANDWIDTH_CONNECT_TIMEOUT": 3,
		"MAX_WORKERS":               200,
		"AVAILABILITY_WORKERS":      10,
		"FALLBACK_WORKERS":          10,
		"BANDWIDTH_WORKERS":         10,
		"DNS_UPDATE_MAX_RETRIES":    3,
		"DNS_UPDATE_RETRY_DELAY":    3,
		"GITHUB_SYNC_MAX_RETRIES":   3,
		"GITHUB_SYNC_RETRY_DELAY":   3,
		"GIT_SYNC_PROCESS_TIMEOUT":  180,
		"AD_HEADER_ENABLED":         false,
		"AD_HEADER_LINES":           []string{},
		"AD_FOOTER_ENABLED":         false,
		"AD_FOOTER_LINES":           []string{},
		"AD_PERLINE_ENABLED":        false,
		"AD_PERLINE_TEXT":           "",
	}

	// 注意：Go 中无法像 Python 那样动态检测字段是否存在
	// 这里我们只在字段为零值时设置默认值
	if !cfg.UseGlobalMode && cfg.GlobalTopN == 0 {
		cfg.UseGlobalMode = defaults["USE_GLOBAL_MODE"].(bool)
	}
	if cfg.GlobalTopN == 0 {
		cfg.GlobalTopN = defaults["GLOBAL_TOP_N"].(int)
	}
	if cfg.PerCountryTopN == 0 {
		cfg.PerCountryTopN = defaults["PER_COUNTRY_TOP_N"].(int)
	}
	if cfg.BandwidthCandidates == 0 {
		cfg.BandwidthCandidates = defaults["BANDWIDTH_CANDIDATES"].(int)
	}
	if cfg.TcpProbes == 0 {
		cfg.TcpProbes = defaults["TCP_PROBES"].(int)
	}
	if cfg.MinSuccessRate == 0 {
		cfg.MinSuccessRate = defaults["MIN_SUCCESS_RATE"].(float64)
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaults["TIMEOUT"].(float64)
	}
	if cfg.SocketDefaultTimeout == 0 {
		cfg.SocketDefaultTimeout = defaults["SOCKET_DEFAULT_TIMEOUT"].(int)
	}
	if cfg.ProgressPrintInterval == 0 {
		cfg.ProgressPrintInterval = defaults["PROGRESS_PRINT_INTERVAL"].(int)
	}
	if cfg.PreFilterBlockedCountries == nil {
		cfg.PreFilterBlockedCountries = defaults["PRE_FILTER_BLOCKED_COUNTRIES"].([]string)
	}
	if cfg.PreFilterPorts == nil {
		cfg.PreFilterPorts = []int{443}
	}
	if cfg.WxpusherUids == nil {
		cfg.WxpusherUids = []string{"your_uid_here"}
	}
	if cfg.WxpusherApiUrl == "" {
		cfg.WxpusherApiUrl = defaults["WXPUSHER_API_URL"].(string)
	}
	if cfg.NotifyTimeout == 0 {
		cfg.NotifyTimeout = defaults["NOTIFY_TIMEOUT"].(int)
	}
	if cfg.NotifyConnectTimeout == 0 {
		cfg.NotifyConnectTimeout = defaults["NOTIFY_CONNECT_TIMEOUT"].(int)
	}
	if cfg.CfTtl == 0 {
		cfg.CfTtl = defaults["CF_TTL"].(int)
	}
	if cfg.CfDnsConnectTimeout == 0 {
		cfg.CfDnsConnectTimeout = defaults["CF_DNS_CONNECT_TIMEOUT"].(int)
	}
	if cfg.CfDnsReadTimeout == 0 {
		cfg.CfDnsReadTimeout = defaults["CF_DNS_READ_TIMEOUT"].(int)
	}
	if cfg.FetchMaxRetries == 0 {
		cfg.FetchMaxRetries = defaults["FETCH_MAX_RETRIES"].(int)
	}
	if cfg.FetchRetryDelay == 0 {
		cfg.FetchRetryDelay = defaults["FETCH_RETRY_DELAY"].(int)
	}
	if cfg.FetchTimeout == 0 {
		cfg.FetchTimeout = defaults["FETCH_TIMEOUT"].(int)
	}
	if cfg.FetchConnectTimeout == 0 {
		cfg.FetchConnectTimeout = defaults["FETCH_CONNECT_TIMEOUT"].(int)
	}
	if cfg.OutputFile == "" {
		cfg.OutputFile = defaults["OUTPUT_FILE"].(string)
	}
	if cfg.AvailabilityCheckApi == "" {
		cfg.AvailabilityCheckApi = defaults["AVAILABILITY_CHECK_API"].(string)
	}
	if cfg.AvailabilityTimeout == 0 {
		cfg.AvailabilityTimeout = defaults["AVAILABILITY_TIMEOUT"].(int)
	}
	if cfg.AvailabilityConnectTimeout == 0 {
		cfg.AvailabilityConnectTimeout = defaults["AVAILABILITY_CONNECT_TIMEOUT"].(int)
	}
	if cfg.AvailabilityRetryMax == 0 {
		cfg.AvailabilityRetryMax = defaults["AVAILABILITY_RETRY_MAX"].(int)
	}
	if cfg.AvailabilityRetryDelay == 0 {
		cfg.AvailabilityRetryDelay = defaults["AVAILABILITY_RETRY_DELAY"].(int)
	}
	if cfg.BlockedCountries == nil {
		cfg.BlockedCountries = defaults["BLOCKED_COUNTRIES"].([]string)
	}
	if cfg.DnsUpdateTargetCount == 0 {
		cfg.DnsUpdateTargetCount = defaults["DNS_UPDATE_TARGET_COUNT"].(int)
	}
	if cfg.BandwidthSizeMB == 0 {
		cfg.BandwidthSizeMB = defaults["BANDWIDTH_SIZE_MB"].(float64)
	}
	if cfg.BandwidthTimeout == 0 {
		cfg.BandwidthTimeout = defaults["BANDWIDTH_TIMEOUT"].(int)
	}
	if cfg.BandwidthRetryMax == 0 {
		cfg.BandwidthRetryMax = defaults["BANDWIDTH_RETRY_MAX"].(int)
	}
	if cfg.BandwidthRetryDelay == 0 {
		cfg.BandwidthRetryDelay = defaults["BANDWIDTH_RETRY_DELAY"].(int)
	}
	if cfg.BandwidthUrlTemplate == "" {
		cfg.BandwidthUrlTemplate = defaults["BANDWIDTH_URL_TEMPLATE"].(string)
	}
	if cfg.BandwidthProcessBuffer == 0 {
		cfg.BandwidthProcessBuffer = defaults["BANDWIDTH_PROCESS_BUFFER"].(int)
	}
	if cfg.BandwidthConnectTimeout == 0 {
		cfg.BandwidthConnectTimeout = defaults["BANDWIDTH_CONNECT_TIMEOUT"].(int)
	}
	if cfg.MaxWorkers == 0 {
		cfg.MaxWorkers = defaults["MAX_WORKERS"].(int)
	}
	if cfg.AvailabilityWorkers == 0 {
		cfg.AvailabilityWorkers = defaults["AVAILABILITY_WORKERS"].(int)
	}
	if cfg.FallbackWorkers == 0 {
		cfg.FallbackWorkers = defaults["FALLBACK_WORKERS"].(int)
	}
	if cfg.BandwidthWorkers == 0 {
		cfg.BandwidthWorkers = defaults["BANDWIDTH_WORKERS"].(int)
	}
	if cfg.DnsUpdateMaxRetries == 0 {
		cfg.DnsUpdateMaxRetries = defaults["DNS_UPDATE_MAX_RETRIES"].(int)
	}
	if cfg.DnsUpdateRetryDelay == 0 {
		cfg.DnsUpdateRetryDelay = defaults["DNS_UPDATE_RETRY_DELAY"].(int)
	}
	if cfg.GithubSyncMaxRetries == 0 {
		cfg.GithubSyncMaxRetries = defaults["GITHUB_SYNC_MAX_RETRIES"].(int)
	}
	if cfg.GithubSyncRetryDelay == 0 {
		cfg.GithubSyncRetryDelay = defaults["GITHUB_SYNC_RETRY_DELAY"].(int)
	}
	if cfg.GitSyncProcessTimeout == 0 {
		cfg.GitSyncProcessTimeout = defaults["GIT_SYNC_PROCESS_TIMEOUT"].(int)
	}
	if cfg.AdHeaderLines == nil {
		cfg.AdHeaderLines = []string{}
	}
	if cfg.AdFooterLines == nil {
		cfg.AdFooterLines = []string{}
	}
}

// GetConfigFilePath 获取配置文件路径
func GetConfigFilePath() string {
	if cwd, err := os.Getwd(); err == nil {
		cwdConfig := filepath.Join(cwd, "config.json")
		if _, err := os.Stat(cwdConfig); err == nil {
			return cwdConfig
		}
	}

	exePath, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "config.json")
}
