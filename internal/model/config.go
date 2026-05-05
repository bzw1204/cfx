package model

type Global struct {
	Enable              bool   `json:"enable" yaml:"enable"`                             // 是否启用全局模式
	TopN                int    `json:"top_n" yaml:"top_n"`                               // 全局Top N
	PerCountryTopN      int    `json:"per_country_top_n" yaml:"per_country_top_n"`       // 每国Top N
	BandwidthCandidates int    `json:"bandwidth_candidates" yaml:"bandwidth_candidates"` // 带宽候选
	FallbackWorkers     int    `json:"fallback_workers" yaml:"fallback_workers"`         // 备用线程数
	OutputFile          string `json:"output_file" yaml:"output_file"`                   // 输出文件路径
}
type TCP struct {
	Probes         int     `json:"probes" yaml:"probes"`                     // 探测次数
	MinSuccessRate float64 `json:"min_success_rate" yaml:"min_success_rate"` // 最小成功率
	SocketTimeout  int     `json:"socket_timeout" yaml:"socket_timeout"`     // 套接字超时
	NetworkTestConfig      `json:",inline" yaml:",inline"`                  // 内嵌网络测试配置
}

type Filter struct {
	CountriesEnabled bool     `json:"countries_enabled" yaml:"countries_enabled"` // 国家过滤配置（白名单）
	AllowedCountries []string `json:"allowed_countries" yaml:"allowed_countries"` // 允许的国家
	BlockedEnabled   bool     `json:"blocked_enabled" yaml:"blocked_enabled"`     // 国家过滤配置（黑名单 - 前置）
	BlockedCountries []string `json:"blocked_countries" yaml:"blocked_countries"` // 被阻国家
	PortEnabled      bool     `json:"port_enabled" yaml:"port_enabled"`           // 端口过滤配置
	Ports            []int    `json:"ports" yaml:"ports"`                         // 端口
}

type Cloudflare struct {
	Enabled           bool   `json:"enabled" yaml:"enabled"`                         // Cloudflare DNS 更新配置
	ApiToken          string `json:"api_token" yaml:"api_token"`                     // Cloudflare API Token
	ZoneId            string `json:"zone_id" yaml:"zone_id"`                         // Cloudflare Zone ID
	DnsRecordName     string `json:"dns_record_name" yaml:"dns_record_name"`         // Cloudflare DNS Record Name
	Ttl               int    `json:"ttl" yaml:"ttl"`                                 // Cloudflare TTL
	Proxied           bool   `json:"proxied" yaml:"proxied"`                         // Cloudflare Proxied
	DnsConnectTimeout int    `json:"dns_connect_timeout" yaml:"dns_connect_timeout"` // Cloudflare DNS Connect Timeout
	DnsReadTimeout    int    `json:"dns_read_timeout" yaml:"dns_read_timeout"`       // Cloudflare DNS Read Timeout
}

type DNS struct {
	Enabled                bool     `json:"enabled" yaml:"enabled"`                                   // 是否启用 DNS
	FilterIpv6Availability bool     `json:"filter_ipv6_availability" yaml:"filter_ipv6_availability"` // 是否过滤 IPv6 可用性
	BlockedCountries       []string `json:"blocked_countries" yaml:"blocked_countries"`               // 被阻国家
	UpdateTargetCount      int      `json:"update_target_count" yaml:"update_target_count"`           // DNS 更新目标数量
	UpdateMaxRetries       int      `json:"update_max_retries" yaml:"update_max_retries"`             // DNS 更新最大重试次数
	UpdateRetryDelay       int      `json:"update_retry_delay" yaml:"update_retry_delay"`             // DNS 更新重试延迟
}
type AdditionalSources struct {
	Enabled bool   `json:"enabled" yaml:"enabled"` // 是否启用附加源
	Url     string `json:"url" yaml:"url"`         // 附加源
}
// NetworkTestConfig 通用网络测试配置
// 用于TCP测试、可用性检测、带宽测速等网络操作
type NetworkTestConfig struct {
	Timeout        int `json:"timeout" yaml:"timeout"`                 // 超时时间（秒）
	ConnectTimeout int `json:"connect_timeout" yaml:"connect_timeout"` // 连接超时时间（秒）
	Retry          int `json:"retry" yaml:"retry"`                     // 重试次数
	RetryDelay     int `json:"retry_delay" yaml:"retry_delay"`         // 重试延迟（秒）
	MaxWorkers     int `json:"max_workers" yaml:"max_workers"`         // 最大并发数
}

// 可用性检查配置
type Availability struct {
	Enabled          bool   `json:"enabled" yaml:"enabled"`                     // 是否启用可用性检查
	CheckApi         string `json:"check_api" yaml:"check_api"`                 // 可用性检查 API
	IPV6Availability bool   `json:"ipv6_availability" yaml:"ipv6_availability"` // 是否启用 IPv6 可用性检查
	NetworkTestConfig       `json:",inline" yaml:",inline"`                   // 内嵌网络测试配置
}

// 带宽配置
type Bandwidth struct {
	Enabled       bool    `json:"enabled" yaml:"enabled"`                 // 是否启用带宽控制
	SizeMB        float64 `json:"size_mb" yaml:"size_mb"`                 // 带宽大小（MB）
	UrlTemplate   string  `json:"url_template" yaml:"url_template"`       // 带宽 URL 模板
	ProcessBuffer int     `json:"process_buffer" yaml:"process_buffer"`   // 带宽处理缓冲区
	NetworkTestConfig     `json:",inline" yaml:",inline"`                 // 内嵌网络测试配置
}

// 日志配置
type Logger struct {
	Enabled       bool   `json:"enabled" yaml:"enabled"`                   // 是否启用日志
	Level         string `json:"level" yaml:"level"`                       // 日志级别
	Format        string `json:"format" yaml:"format"`                     // 日志格式
	Output        string `json:"output" yaml:"output"`                     // 日志输出目标
	ConsoleOutput bool   `json:"console_output" yaml:"console_output"`     // 控制台输出
	FileOutput    bool   `json:"file_output" yaml:"file_output"`           // 文件输出
	LogDir        string `json:"log_dir" yaml:"log_dir"`                   // 日志目录
	MaxFileSize   int    `json:"max_file_size" yaml:"max_file_size"`       // 文件大小限制(MB)
	MaxBackups    int    `json:"max_backups" yaml:"max_backups"`           // 备份数量
	MaxAge        int    `json:"max_age" yaml:"max_age"`                   // 保存天数
	Compress      bool   `json:"compress" yaml:"compress"`                 // 是否压缩
}

// HTTP 服务配置
type Http struct {
	Enabled bool   `json:"enabled" yaml:"enabled"` // 是否启用 HTTP 服务
	Port    int    `json:"port" yaml:"port"`       // HTTP 监听端口
	Path    string `json:"path" yaml:"path"`       // 返回文件内容的路径，如 /ips
}

// 定时任务配置
type Schedule struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`     // 是否启用定时任务
	Interval string `json:"interval" yaml:"interval"`   // 间隔，如 "1h", "30m", "cron:0 */2 * * *"
}

type Config struct {
	Global            Global              `json:"global" yaml:"global"`                         // 全局配置
	Tcp               TCP                 `json:"tcp" yaml:"tcp"`                               // TCP 配置
	Filter            Filter              `json:"filter" yaml:"filter"`                         // 过滤器配置
	Cloudflare        Cloudflare          `json:"cloudflare" yaml:"cloudflare"`                 // Cloudflare 配置
	AdditionalSources []AdditionalSources `json:"additional_sources" yaml:"additional_sources"` // 附加源配置
	Dns               DNS                 `json:"dns" yaml:"dns"`                               // DNS 配置
	Availability      Availability        `json:"availability" yaml:"availability"`             // 可用性检查配置
	Bandwidth         Bandwidth           `json:"bandwidth" yaml:"bandwidth"`                   // 带宽配置
	Logger            Logger              `json:"logger" yaml:"logger"`                         // 日志配置
	Http              Http                `json:"http" yaml:"http"`                             // HTTP 服务配置
	Schedule          Schedule            `json:"schedule" yaml:"schedule"`                     // 定时任务配置
}
