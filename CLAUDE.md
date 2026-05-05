# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CFX is a high-performance Cloudflare IP optimization tool written in Go, migrated from Python to provide better performance and concurrency. It automatically discovers, tests, and selects the best Cloudflare IP addresses based on TCP connectivity, availability, and bandwidth speed.

## Key Commands

### Build and Run
```bash
# Build the project
go build -o cfx .

# Run the application
./cfx

# Run with custom config file
./cfx -config /path/to/config.yaml

# Build and run in one command
go run .
```

### Development Commands
```bash
# Install dependencies
go mod tidy

# Run tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestName ./path/to/package

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o cfx-linux .
GOOS=windows GOARCH=amd64 go build -o cfx.exe .

# Check for race conditions
go test -race ./...

# Profile performance
go test -bench=. ./...
```

### Docker Commands
```bash
# Build Docker image
docker build -t cfx .

# Run with Docker Compose
docker-compose up

# Run with custom config
docker run -v $(pwd)/config.yaml:/app/config.yaml cfx
```

## Architecture Overview

The application follows a modular pipeline architecture with distinct phases:

1. **Configuration Layer** (`internal/config/`)
   - `config.go`: Configuration loading and validation
   - `zap_logger.go`: Structured logging setup

2. **Data Models** (`internal/model/`)
   - `config.go`: All configuration structs and types

3. **Core Pipeline** (`internal/app.go`)
   - Orchestrates the 9-phase workflow
   - Handles node fetching, filtering, testing, and selection

4. **Network Operations** (`internal/network/`)
   - `probe_tcp.go`: High-concurrency TCP connectivity testing
   - `availability.go`: Secondary availability verification via API
   - `bandwidth.go`: Real download speed testing

5. **Utilities** (`internal/utils/`)
   - `parser.go`: Adaptive parsing for multiple data source formats
   - `filter.go`: Port and country-based filtering
   - `http.go`: Concurrent data source fetching
   - `str_tools.go`: String manipulation utilities

6. **DNS Integration** (`internal/dns/`)
   - `cloudflare.go`: Batch Cloudflare DNS record updates

7. **HTTP Server** (`internal/http/`)
   - `server.go`: Optional HTTP API for monitoring/control

## Key Design Patterns

- **Pipeline Architecture**: Sequential processing phases with clear data flow
- **Concurrent Processing**: Goroutines with semaphore-based concurrency control
- **Retry Mechanisms**: Automatic retries for network operations
- **Graceful Degradation**: Fallback strategies when optimal paths fail
- **Structured Logging**: Zap logger with consistent log levels

## Configuration Management

The application uses Viper for configuration with support for:
- YAML/JSON config files
- Environment variables
- Command-line flags

Key configuration sections:
- `global`: Main optimization settings
- `tcp`: TCP testing parameters
- `bandwidth`: Speed testing configuration
- `availability`: Secondary verification settings
- `filter`: Pre-filtering options
- `cloudflare`: DNS update configuration
- `schedule`: Cron-based scheduling

## Testing Strategy

- Unit tests in `*_test.go` files (note: TCP test file is `tcp.go`, not `tcp_test.go`)
- Integration tests for network operations
- Mock-free design relying on real network calls with timeouts
- Test data in `data/` directory

## Common Development Tasks

### Adding a New Filter
1. Add filter logic to `internal/utils/filter.go`
2. Update config struct in `internal/model/config.go`
3. Integrate into pipeline in `internal/app.go`
4. Update configuration documentation

### Adding a New Data Source Format
1. Extend parser logic in `internal/utils/parser.go`
2. Add format detection and parsing functions
3. Test with sample data

### Modifying Test Parameters
1. Update configuration structs in `internal/model/config.go`
2. Modify test logic in `internal/network/` files
3. Update default values in config loading

### Recent Architecture Improvements

#### 日志系统重构
- 实现增强的日志管理器 (`internal/config/enhanced_logger.go`)
- 支持日志轮转和自动压缩 (lumberjack)
- 多级别分文件存储 (INFO/ERROR 分离)
- 结构化 JSON 日志格式
- 控制台彩色输出

#### 进度管理优化
- 创建通用进度管理器 (`internal/utils/progress.go`)
- 支持多任务并行进度显示
- 终端单行更新显示
- 进度条可视化效果
- 进度信息同时记录到日志

#### 配置结构优化
- 抽象通用网络测试配置 (`NetworkTestConfig`)
- 消除重复的配置字段 (TCP、Availability、Bandwidth 共用配置)
- 使用内嵌结构减少冗余

#### 并发框架抽象
- 创建通用并发执行器 (`internal/utils/concurrent.go`)
- 统一可用性检测和带宽测试的并发逻辑
- 支持重试机制和错误处理
- 内置 panic 恢复机制

#### 测试完善
- 为进度管理器添加单元测试
- 为并发执行器添加单元测试
- 为日志管理器添加单元测试
- 测试覆盖率显著提高

## Performance Considerations

- Use buffered channels for goroutine coordination
- Implement proper timeout handling for all network operations
- Limit concurrent connections to avoid resource exhaustion
- Use connection pooling where appropriate
- Implement circuit breakers for external API calls

## Error Handling Patterns

- Wrap errors with context using `fmt.Errorf` and `%w`
- Use structured logging for debugging
- Implement graceful degradation for non-critical failures
- Provide meaningful error messages for configuration issues

## Code Style Guidelines

- Follow standard Go conventions (gofmt, golint)
- Use camelCase for exported identifiers
- Document all exported functions and types
- Use Chinese comments for clarity (as per project convention)
- Implement proper error handling at all levels

## Deployment Considerations

- Single binary deployment (no external dependencies)
- Configuration via environment variables for containerization
- Log rotation and management for long-running instances
- Health checks for scheduled operations
- Proper signal handling for graceful shutdown
