# Changelog

All notable changes to this project will be documented in this file.

### v0.1.1(20260218)
#### feature:
1. MCP新增远程SSH收集工具 `system_summary_remote`，支持通过SSH获取远程主机系统摘要信息
2. MCP新增批量远程收集工具 `system_summary_remote_batch`，支持并行从多个SSH主机收集系统信息

#### optimization:
1. 抽取 `FormatBytes` 和 `FormatUptime` 函数到 `utils/format/format.go`，供多处复用
2. `internal/formatter/text.go` 使用抽取的公共格式化函数
3. `internal/mcp/handler/system.go` 使用抽取的公共格式化函数，并集成远程SSH收集能力

## v0.1.0 (2026-02-18)

### Features
1. **Local System Collection**: Implemented `summary` command to collect CPU, memory, disk, network, processes, and load average using gopsutil
2. **SSH Remote Collection**: Support remote system info collection via SSH with key or password authentication
3. **Multiple Output Formats**: Text and JSON output support via formatter interface
4. **Parallel SSH Execution**: Added `--parallel` flag for concurrent collection from multiple hosts
5. **MCP Server Integration**: Exposed system info tools via MCP protocol (stdio mode)

### Documentation
- Added `docs/examples/` directory with MCP configuration examples:
  - `mcp.json` - Basic MCP server configuration
  - `claude_desktop_config.json` - Claude Desktop integration
  - `cursor_mcp.json` - Cursor IDE integration
  - `webapp_mcp_config.json` - Web application SSE/HTTP integration
  - `docker-compose.yml` - Docker deployment example
  - `README.md` - Configuration documentation with Inspector usage guide
- Added MCP Inspector usage documentation covering:
  - Installation (npm/npx)
  - stdio transport testing
  - SSE/HTTP transport testing
  - Command-line options
  - Troubleshooting guide

### Architecture
- `LocalCollector`: Uses gopsutil to collect local system info
- `RemoteCollector`: Executes commands via SSH on remote hosts (batch execution)
- `TextFormatter`: Human-readable text output
- `JSONFormatter`: JSON output for programmatic use

### Commands
- `summary`: Collect system summary (local or remote via SSH)
- `mcp`: Start MCP stdio server with system info tools
- `version`: Print build and version info

### MCP Tools
- `system_summary_local`: Get complete system summary (JSON)
- `system_cpu`: CPU information
- `system_memory`: Memory information
- `system_disk`: Disk usage by partition
- `system_network`: Network interfaces and connections
- `system_processes`: Top CPU/memory processes
- `system_loadavg`: System load average
- `system_info`: Basic system info

### Performance
- Local collection: ~3 seconds (includes 1s CPU usage sampling)
- Remote collection: Single SSH session for all commands (batch execution)
- Parallel SSH: Configurable concurrency (default: 5)
