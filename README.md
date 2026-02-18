# summary-sys

English | [简体中文](./README_CN.md)

A Go CLI tool for collecting system summary information, inspired by Percona Toolkit's `pt-summary`. Supports local and remote (SSH) collection, with MCP Server integration.

## Features

- **Local System Collection**: Gather CPU, memory, disk, network, processes, and load average
- **Remote SSH Collection**: Collect system info from remote servers via SSH
- **Multiple Output Formats**: Text and JSON output support
- **Parallel Execution**: Efficient parallel collection for multiple SSH hosts
- **MCP Server Integration**: Expose system info as MCP tools for AI assistants

## Quick Start

```bash
# Build
go build -o bin/summary-sys ./main.go

# Local collection (default)
./bin/summary-sys summary

# JSON output
./bin/summary-sys summary -f json

# SSH remote collection
./bin/summary-sys summary --ssh --hosts=192.168.1.10 --ssh-user=root --ssh-key=~/.ssh/id_rsa

# Multiple hosts with parallel execution
./bin/summary-sys summary --ssh --hosts=host1,host2,host3 --parallel 5
```

## Commands

### `summary` - System Summary Collection

Collect system information from local or remote hosts.

```bash
./bin/summary-sys summary [flags]
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | `-f` | `text` | Output format: text, json |
| `--output` | `-o` | stdout | Output file path |
| `--ssh` | - | false | Enable SSH remote collection |
| `--hosts` | - | [] | SSH host list (comma-separated) |
| `--ssh-user` | - | `root` | SSH username |
| `--ssh-port` | - | `22` | SSH port |
| `--ssh-key` | - | - | SSH private key path |
| `--ssh-password` | - | - | SSH password |
| `--parallel` | - | `5` | Number of parallel SSH connections |
| `--timeout` | - | `30` | Collection timeout in seconds |

**Examples:**

```bash
# Local collection with text output
./bin/summary-sys summary

# Local collection with JSON output
./bin/summary-sys summary -f json

# Single remote host
./bin/summary-sys summary --ssh --hosts=192.168.1.10 --ssh-user=root --ssh-key=~/.ssh/id_rsa

# Multiple remote hosts in parallel
./bin/summary-sys summary --ssh --hosts=host1,host2,host3 --ssh-user=admin --ssh-key=~/.ssh/id_rsa --parallel 10

# Save to file
./bin/summary-sys summary -f json -o output.json
```

### `mcp` - MCP Server

Start MCP stdio server with system info tools.

```bash
./bin/summary-sys mcp
```

**MCP Transport Options:**

| Transport | Description | Use Case |
|----------|-------------|----------|
| `stdio` | Standard input/output | Claude Desktop, Cursor, VS Code |
| `sse` | Server-Sent Events over HTTP | Web apps, REST integrations |
| `http` | Streamable HTTP | Modern HTTP clients |
| `all` | All transports | Development/testing |

**MCP Command Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--transport` | `stdio` | Transport: stdio, sse, http, all |
| `--host` | `0.0.0.0` | Host for SSE/HTTP |
| `--port` | `8080` | Port for SSE/HTTP |
| `--config` | - | Path to JSON config file |

**MCP Examples:**

```bash
# Start with stdio (default, for Claude/Cursor)
./bin/summary-sys mcp

# Start with SSE
./bin/summary-sys mcp --transport sse --host localhost --port 8080

# Start with HTTP
./bin/summary-sys mcp --transport http --host 0.0.0.0 --port 8080

# Start all transports
./bin/summary-sys mcp --transport all

# Start with config file
./bin/summary-sys mcp --config mcp.json
```

**MCP JSON Configuration:**

Create a config file (e.g., `mcp.json`):

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
# Use config file
./bin/summary-sys mcp --config mcp.json
```

**Use with Claude Desktop:**

Add to `claude_desktop_config.json`:

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

**Use with SSE (for web apps):**

```json
{
  "mcpServers": {
    "summary-sys": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

Then start the server:
```bash
./bin/summary-sys mcp --transport sse --port 8080
```

**Available MCP Tools:**

| Tool | Description |
|------|-------------|
| `system_summary_local` | Get complete system summary (JSON format) |
| `system_cpu` | CPU information (cores, model, usage) |
| `system_memory` | Memory information (total, used, available) |
| `system_disk` | Disk usage by partition |
| `system_network` | Network interfaces and connections |
| `system_processes` | Top CPU/memory consuming processes |
| `system_loadavg` | System load average |
| `system_info` | Basic system info (hostname, OS, kernel, uptime) |

### Other Commands

- `version`: Print build and version info

## Project Structure

```
summary-sys/
├── cmd/
│   ├── summary.go       # summary command
│   ├── mcp.go           # mcp server command
│   ├── root.go          # root command
│   └── version.go       # version command
├── internal/
│   ├── collector/
│   │   ├── collector.go # Collector interface
│   │   ├── types.go     # Data structures
│   │   ├── local.go     # Local collection (gopsutil)
│   │   └── remote.go   # SSH remote collection
│   ├── ssh/
│   │   ├── client.go    # SSH client
│   │   └── auth.go      # SSH authentication
│   ├── formatter/
│   │   ├── formatter.go
│   │   ├── text.go      # Text formatter
│   │   └── json.go      # JSON formatter
│   └── mcp/
│       ├── server.go
│       └── handler/
│           └── system.go  # System collection tools
├── pkg/log              # Logger
├── utils/              # Utilities
├── main.go             # Entry point
└── go.mod
```

## Development

```bash
# Install dependencies
go mod tidy

# Build
go build -o bin/summary-sys ./main.go

# Run
go run main.go <command>

# Format code
go fmt ./...

# Lint (if configured)
golangci-lint run
```

## Architecture

### Collector Interface

```go
type Collector interface {
    Collect(ctx context.Context) (*SystemInfo, error)
    Name() string
}
```

Implementations:
- `LocalCollector`: Uses gopsutil to collect local system info
- `RemoteCollector`: Executes commands via SSH on remote hosts

### Formatter Interface

```go
type Formatter interface {
    Format(info *collector.SystemInfo) (string, error)
    Name() string
    ContentType() string
}
```

Implementations:
- `TextFormatter`: Human-readable text output
- `JSONFormatter`: JSON output for programmatic use

## Performance

- **Local Collection**: ~3 seconds (includes 1s CPU usage sampling)
- **Remote Collection**: Single SSH session for all commands (batch execution)
- **Parallel SSH**: Configurable concurrency (default: 5 parallel connections)

## License

MIT. See `LICENSE`.
