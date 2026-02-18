# summary-sys

[English](./README.md) | 简体中文

Go CLI 工具，用于收集系统摘要信息，灵感来自 Percona Toolkit 的 `pt-summary`。支持本地和远程（SSH）采集，并提供 MCP Server 集成。

## 功能特性

- **本地系统采集**: 收集 CPU、内存、磁盘、网络、进程和负载信息
- **SSH 远程采集**: 通过 SSH 从远程服务器收集系统信息
- **多种输出格式**: 支持 Text 和 JSON 格式输出
- **并行执行**: 支持多 SSH 主机并行采集
- **MCP Server 集成**: 将系统信息暴露为 MCP 工具，供 AI 助手调用

## 快速开始

```bash
# 构建
go build -o bin/summary-sys ./main.go

# 本地采集（默认）
./bin/summary-sys summary

# JSON 输出
./bin/summary-sys summary -f json

# SSH 远程采集
./bin/summary-sys summary --ssh --hosts=192.168.1.10 --ssh-user=root --ssh-key=~/.ssh/id_rsa

# 多主机并行采集
./bin/summary-sys summary --ssh --hosts=host1,host2,host3 --parallel 5
```

## 命令说明

### `summary` - 系统摘要采集

从本地或远程主机收集系统信息。

```bash
./bin/summary-sys summary [flags]
```

**参数说明:**

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--format` | `-f` | `text` | 输出格式: text, json |
| `--output` | `-o` | stdout | 输出文件路径 |
| `--ssh` | - | false | 启用 SSH 远程采集 |
| `--hosts` | - | [] | SSH 主机列表（逗号分隔） |
| `--ssh-user` | - | `root` | SSH 用户名 |
| `--ssh-port` | - | `22` | SSH 端口 |
| `--ssh-key` | - | - | SSH 私钥路径 |
| `--ssh-password` | - | - | SSH 密码 |
| `--parallel` | - | `5` | 并行 SSH 连接数 |
| `--timeout` | - | `30` | 采集超时时间（秒） |

**使用示例:**

```bash
# 本地采集（文本输出）
./bin/summary-sys summary

# 本地采集（JSON 输出）
./bin/summary-sys summary -f json

# 单个远程主机
./bin/summary-sys summary --ssh --hosts=192.168.1.10 --ssh-user=root --ssh-key=~/.ssh/id_rsa

# 多个远程主机并行采集
./bin/summary-sys summary --ssh --hosts=host1,host2,host3 --ssh-user=admin --ssh-key=~/.ssh/id_rsa --parallel 10

# 保存到文件
./bin/summary-sys summary -f json -o output.json
```

### `mcp` - MCP Server

启动 MCP stdio server，提供系统信息工具。

```bash
./bin/summary-sys mcp
```

**MCP 传输协议选项:**

| 传输协议 | 说明 | 使用场景 |
|----------|------|----------|
| `stdio` | 标准输入/输出 | Claude Desktop, Cursor, VS Code |
| `sse` | HTTP Server-Sent Events | Web 应用, REST 集成 |
| `http` | Streamable HTTP | 现代 HTTP 客户端 |
| `all` | 所有协议 | 开发/测试 |

**MCP 命令参数:**

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--transport` | `stdio` | 传输协议: stdio, sse, http, all |
| `--host` | `0.0.0.0` | SSE/HTTP 绑定地址 |
| `--port` | `8080` | SSE/HTTP 端口 |
| `--config` | - | JSON 配置文件路径 |

**MCP 使用示例:**

```bash
# STDIO 模式（默认，用于 Claude/Cursor）
./bin/summary-sys mcp

# SSE 模式
./bin/summary-sys mcp --transport sse --host localhost --port 8080

# HTTP 模式
./bin/summary-sys mcp --transport http --host 0.0.0.0 --port 8080

# 启动所有协议
./bin/summary-sys mcp --transport all

# 使用配置文件
./bin/summary-sys mcp --config mcp.json
```

**MCP JSON 配置文件:**

创建配置文件 (`mcp.json`):

```json
{
  "name": "summary-sys",
  "version": "0.1.0",
  "transport": "stdio",
  "host": "0.0.0.0",
  "port": 8080,
  "enable_stdio": true,
  "enable_sse": true,
  "enable_http": true
}
```

```bash
# 使用配置文件
./bin/summary-sys mcp --config mcp.json
```

**在 Claude Desktop 中使用:**

添加到 `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "summary-sys": {
      "command": "/path/to/bin/summary-sys",
      "args": ["mcp"]
    }
  }
}
```

**SSE 模式（用于 Web 应用）:**

```json
{
  "mcpServers": {
    "summary-sys": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

启动服务器:
```bash
./bin/summary-sys mcp --transport sse --port 8080
```

**可用的 MCP 工具:**

| 工具 | 说明 |
|------|------|
| `system_summary_local` | 获取完整系统摘要（JSON 格式） |
| `system_cpu` | CPU 信息（核心数、型号、使用率） |
| `system_memory` | 内存信息（总量、已用、可用） |
| `system_disk` | 各分区磁盘使用情况 |
| `system_network` | 网络接口和连接数 |
| `system_processes` | Top CPU/内存占用进程 |
| `system_loadavg` | 系统负载平均值 |
| `system_info` | 基础系统信息（主机名、OS、内核、运行时间） |

### 其他命令

- `version`: 输出构建和版本信息

## 项目结构

```
summary-sys/
├── cmd/
│   ├── summary.go       # summary 命令
│   ├── mcp.go            # mcp server 命令
│   ├── root.go           # 根命令
│   └── version.go        # version 命令
├── internal/
│   ├── collector/
│   │   ├── collector.go  # Collector 接口定义
│   │   ├── types.go      # 数据结构
│   │   ├── local.go      # 本地采集（gopsutil）
│   │   └── remote.go     # SSH 远程采集
│   ├── ssh/
│   │   ├── client.go    # SSH 客户端
│   │   └── auth.go      # SSH 认证
│   ├── formatter/
│   │   ├── formatter.go
│   │   ├── text.go      # 文本格式化
│   │   └── json.go      # JSON 格式化
│   └── mcp/
│       ├── server.go
│       └── handler/
│           ├── hello.go   # 示例工具
│           └── system.go  # 系统采集工具
├── pkg/log                # 日志模块
├── utils/                # 工具函数
├── main.go               # 入口文件
└── go.mod
```

## 开发指南

```bash
# 安装依赖
go mod tidy

# 构建
go build -o bin/summary-sys ./main.go

# 运行
go run main.go <command>

# 代码格式化
go fmt ./...

# 代码检查（如已配置）
golangci-lint run
```

## 架构设计

### Collector 接口

```go
type Collector interface {
    Collect(ctx context.Context) (*SystemInfo, error)
    Name() string
}
```

实现:
- `LocalCollector`: 使用 gopsutil 采集本地系统信息
- `RemoteCollector`: 通过 SSH 在远程主机执行命令采集

### Formatter 接口

```go
type Formatter interface {
    Format(info *collector.SystemInfo) (string, error)
    Name() string
    ContentType() string
}
```

实现:
- `TextFormatter`: 人类可读的文本输出
- `JSONFormatter`: JSON 格式输出，便于程序处理

## 性能

- **本地采集**: 约 3 秒（含 1 秒 CPU 使用率采样）
- **远程采集**: 单次 SSH 会话执行所有命令（批量执行）
- **并行 SSH**: 可配置并发数（默认: 5 个并行连接）

## License

MIT，详见 `LICENSE`。
