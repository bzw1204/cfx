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
	Timeout        int     `json:"timeout" yaml:"timeout"`                   // 超时时间
	MinSuccessRate float64 `json:"min_success_rate" yaml:"min_success_rate"` // 最小成功率
	SocketTimeout  int     `json:"socket_timeout" yaml:"socket_timeout"`     // 套接字超时
	MaxWorkers     int     `json:"max_workers" yaml:"max_workers"`           // 最大线程数
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
type Node struct {
	Timeout        int `json:"timeout" yaml:"timeout"`                 // 节点超时时间（秒）
	Retry          int `json:"retry" yaml:"retry"`                     // 节点重试次数
	Retries        int `json:"retries" yaml:"retries"`                 // 节点重试次数
	ConnectTimeout int `json:"connect_timeout" yaml:"connect_timeout"` // 节点连接超时时间（秒）
}

// 可用性检查配置
type Availability struct {
	Enabled          bool   `json:"enabled" yaml:"enabled"`                     // 是否启用可用性检查
	CheckApi         string `json:"check_api" yaml:"check_api"`                 // 可用性检查 API
	Timeout          int    `json:"timeout" yaml:"timeout"`                     // 可用性检查超时时间（秒）
	ConnectTimeout   int    `json:"connect_timeout" yaml:"connect_timeout"`     // 可用性检查连接超时时间（秒）
	Retry            int    `json:"retry" yaml:"retry"`                         // 可用性检测整体失败（通过率为0）时的最大重试轮数
	RetryDelay       int    `json:"retry_delay" yaml:"retry_delay"`             // 可用性检查重试延迟（秒）
	MaxWorkers       int    `json:"max_workers" yaml:"max_workers"`             // 最大线程数
	IPV6Availability bool   `json:"ipv6_availability" yaml:"ipv6_availability"` // 是否启用 IPv6 可用性检查
}

// 带宽配置
type Bandwidth struct {
	Enabled        bool    `json:"enabled" yaml:"enabled"`                 // 是否启用带宽控制
	SizeMB         float64 `json:"size_mb" yaml:"size_mb"`                 // 带宽大小（MB）
	Timeout        int     `json:"timeout" yaml:"timeout"`                 // 带宽超时时间（秒）
	Retry          int     `json:"retry" yaml:"retry"`                     // 带宽重试次数
	RetryDelay     int     `json:"retry_delay" yaml:"retry_delay"`         // 带宽重试延迟（秒）
	UrlTemplate    string  `json:"url_template" yaml:"url_template"`       // 带宽 URL 模板
	ProcessBuffer  int     `json:"process_buffer" yaml:"process_buffer"`   // 带宽处理缓冲区
	ConnectTimeout int     `json:"connect_timeout" yaml:"connect_timeout"` // 带宽连接超时时间（秒）
	MaxWorkers     int     `json:"max_workers" yaml:"max_workers"`         // 最大线程数
}

// 日志配置
type Logger struct {
	Enabled bool   `json:"enabled" yaml:"enabled"` // 是否启用日志
	Level   string `json:"level" yaml:"level"`     // 日志级别
	Format  string `json:"format" yaml:"format"`   // 日志格式
	Output  string `json:"output" yaml:"output"`   // 日志输出目标
}
type Config struct {
	Global            Global              `json:"global" yaml:"global"`                         // 全局配置
	Tcp               TCP                 `json:"tcp" yaml:"tcp"`                               // TCP 配置
	Filter            Filter              `json:"filter" yaml:"filter"`                         // 过滤器配置
	Cloudflare        Cloudflare          `json:"cloudflare" yaml:"cloudflare"`                 // Cloudflare 配置
	AdditionalSources []AdditionalSources `json:"additional_sources" yaml:"additional_sources"` // 附加源配置
	Dns               DNS                 `json:"dns" yaml:"dns"`                               // DNS 配置
	Node              Node                `json:"node" yaml:"node"`                             // 节点配置
	Availability      Availability        `json:"availability" yaml:"availability"`             // 可用性检查配置
	Bandwidth         Bandwidth           `json:"bandwidth" yaml:"bandwidth"`                   // 带宽配置
	Logger            Logger              `json:"logger" yaml:"logger"`                         // 日志配置
}
