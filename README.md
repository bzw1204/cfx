# Cloudflare IP 优选工具 (Go 版本)

这是一个高性能的 Cloudflare IP 优选工具，已从 Python 迁移到 Go，提供更好的性能和并发能力。

## 功能特性

- ✅ **多数据源支持** - 自适应解析 JSON、纯文本等多种格式
- ✅ **智能过滤** - 端口过滤、国家白名单/黑名单过滤
- ✅ **TCP 连通性测试** - 高并发 TCP 延迟测试
- ✅ **可用性检测** - API 二次筛选确保节点可用
- ✅ **带宽测速** - 真实下载速度测试
- ✅ **Cloudflare DNS 批量更新** - 自动更新 DNS 记录实现负载均衡
- ✅ **GitHub 自动同步** - 推送结果到 GitHub
- ✅ **微信通知** - WxPusher 推送运行状态
- ✅ **广告植入** - 支持头部、尾部、行尾广告

## 项目结构

```
cfx/
├── main.go                          # 主程序入口
├── go.mod                           # Go 模块依赖
├── config.yaml                      # 配置文件（YAML/JSON均可）
├── Dockerfile                       # Docker 构建文件
├── docker-compose.yml               # Docker Compose 编排
├── internal/
│   ├── app.go                       # 应用层：管道编排
│   ├── model/
│   │   └── config.go                # 配置结构体定义
│   ├── config/
│   │   ├── config.go                # 配置加载 + 验证
│   │   └── zap_logger.go            # 日志初始化
│   ├── constants/
│   │   └── constants.go             # 常量（国家映射）
│   ├── utils/
│   │   ├── parser.go                # 自适应节点解析引擎
│   │   ├── filter.go                # 端口/国家过滤
│   │   ├── http.go                  # 数据源并发拉取
│   │   └── str_tools.go             # 工具函数
│   ├── network/
│   │   ├── probe_tcp.go             # TCP 连通性探测（高并发）
│   │   ├── availability.go          # 可用性检测
│   │   └── bandwidth.go             # 带宽测速
│   └── dns/
│       └── cloudflare.go            # Cloudflare DNS 批量更新
└── cmd/                             # （可选）子命令入口
```

## 快速开始

### 1. 安装 Go

确保已安装 Go 1.26.1 或更高版本：

```bash
go version
```

### 2. 配置

编辑 `config.yaml` 文件（也支持 `config.json`，键名需与 Go 结构体字段匹配），根据需要修改参数。关键配置项：

```yaml
global:
  enable: true              # 全局模式
  top_n: 10                 # 全局保留节点数
additional_sources:
  - url: "https://example.com/nodes.txt"
    enabled: true
cloudflare:
  enabled: false            # Cloudflare DNS 更新开关
```

### 3. 编译

```bash
go build -o cfx .
```

### 4. 运行

```bash
./cfx
```

程序会自动：
1. 从配置的数据源获取节点
2. 进行前置过滤（端口、国家）
3. TCP 连通性测试
4. 可用性二次筛选
5. 带宽测速
6. 生成 `ip.txt` 文件
7. 更新 Cloudflare DNS（如果启用）
8. 同步到 GitHub（如果配置）

## 配置说明

### 核心参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `USE_GLOBAL_MODE` | 全局最优模式 | true |
| `GLOBAL_TOP_N` | 全局模式节点数 | 15 |
| `PER_COUNTRY_TOP_N` | 每国家节点数 | 1 |
| `BANDWIDTH_CANDIDATES` | 带宽测速候选数 | 90 |
| `TCP_PROBES` | TCP 探测次数 | 3 |
| `MIN_SUCCESS_RATE` | 最低成功率 | 1.0 |
| `MAX_WORKERS` | TCP 并发数 | 200 |

### 过滤配置

- **端口过滤**: `PRE_FILTER_PORT_ENABLED`, `PRE_FILTER_PORTS`
- **国家白名单**: `FILTER_COUNTRIES_ENABLED`, `ALLOWED_COUNTRIES`
- **国家黑名单**: `PRE_FILTER_BLOCKED_ENABLED`, `PRE_FILTER_BLOCKED_COUNTRIES`

### Cloudflare DNS

需要配置：
- `CF_API_TOKEN` - Cloudflare API Token
- `CF_ZONE_ID` - Zone ID
- `CF_DNS_RECORD_NAME` - DNS 记录名称

### 通知配置

WxPusher 配置：
- `WXPUSHER_APP_TOKEN` - 应用 Token
- `WXPUSHER_UIDS` - 用户 UID 列表

## 性能优势

相比 Python 版本，Go 版本具有以下优势：

1. **更高的并发性能** - Goroutine 比 Python 线程更轻量
2. **更低的内存占用** - 编译型语言，无解释器开销
3. **更快的启动速度** - 无需加载 Python 解释器和依赖
4. **单一二进制文件** - 无需安装依赖，部署简单
5. **类型安全** - 编译时检查，减少运行时错误

## 注意事项

1. **文件名问题**: Go 中 `_test.go` 后缀的文件仅用于测试，不会被普通编译包含。本项目中的 TCP 测试文件命名为 `tcp.go` 而非 `tcp_test.go`。

2. **并发控制**: 使用信号量（channel）控制并发数，避免资源耗尽。

3. **超时设置**: 所有网络操作都设置了连接超时和读取超时，防止永久阻塞。

4. **重试机制**: 关键操作（数据源获取、可用性检测、带宽测速、DNS 更新）都支持重试。

## 开发

### 添加新功能

1. 在 `internal/` 下创建新包
2. 导出公开的函数和类型（首字母大写）
3. 在 `internal/app.go` 或 `main.go` 中导入并使用

### 代码风格

- 使用驼峰命名法（CamelCase）
- 公开标识符首字母大写
- 注释使用中文，清晰描述功能
- 错误处理使用 `fmt.Errorf` 和 `%w` 包装

## 许可证

MIT License

## 致谢

感谢 Python 版本的原作者提供了优秀的算法和架构设计。
