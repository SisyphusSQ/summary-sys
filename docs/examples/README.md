# MCP Configuration Examples

This directory contains example configurations for integrating `summary-sys` MCP server with various AI assistants and applications.

## Files Overview

| File | Description |
|------|-------------|
| `mcp.json` | Basic MCP server configuration file |
| `claude_desktop_config.json` | Claude Desktop integration |
| `cursor_mcp.json` | Cursor IDE integration |
| `webapp_mcp_config.json` | Web application SSE/HTTP integration |
| `docker-compose.yml` | Docker deployment example |

## Quick Start

### 1. Claude Desktop

```bash
# 1. Build the binary
go build -o bin/summary-sys ./main.go

# 2. Copy the binary to a permanent location
cp bin/summary-sys /usr/local/bin/summary-sys

# 3. Add config to Claude Desktop
# macOS: ~/Library/Application Support/Claude/claude_desktop_config.json
# Linux: ~/.config/Claude/claude_desktop_config.json

# 4. Restart Claude Desktop
```

Edit `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "summary-sys": {
      "command": "/usr/local/bin/summary-sys",
      "args": ["mcp"]
    }
  }
}
```

### 2. Cursor IDE

Edit `cursor_mcp.json` and add to Cursor settings:

```json
{
  "mcpServers": {
    "summary-sys": {
      "command": "/usr/local/bin/summary-sys",
      "args": ["mcp"]
    }
  }
}
```

### 3. Web Application (SSE)

```bash
# Start MCP server with SSE transport
./bin/summary-sys mcp --transport sse --port 8080
```

Then configure your web app:

```json
{
  "mcpServers": {
    "summary-sys": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

## Available MCP Tools

Once connected, you can use these tools:

| Tool | Description |
|------|-------------|
| `system_summary_local` | Get complete system summary (JSON) |
| `system_cpu` | CPU information |
| `system_memory` | Memory information |
| `system_disk` | Disk usage by partition |
| `system_network` | Network interfaces and connections |
| `system_processes` | Top CPU/memory processes |
| `system_loadavg` | System load average |
| `system_info` | Basic system info |

## Environment Variables

You can also configure via environment variables:

```bash
export SUMMARY_SYS_TRANSPORT=stdio
export SUMMARY_SYS_PORT=8080
export SUMMARY_SYS_LOG_LEVEL=info

./bin/summary-sys mcp
```

## Debug with Inspector

Use `@modelcontextprotocol/inspector` to test and debug your MCP server.

### Installation

```bash
# Install globally
npm install -g @modelcontextprotocol/inspector

# Or run directly with npx
npx @modelcontextprotocol/inspector
```

### Usage with stdio Transport

```bash
# Build the binary first
go build -o bin/summary-sys ./main.go

# Run inspector with stdio
npx @modelcontextprotocol/inspector ./bin/summary-sys mcp
```

This will open a web UI at `http://localhost:5173` where you can:
- View available tools
- Test tool calls with custom parameters
- See request/response logs
- Debug connection issues

### Usage with SSE/HTTP Transport

```bash
# Start MCP server in one terminal
./bin/summary-sys mcp --transport sse --port 8080

# Run inspector with HTTP URL
npx @modelcontextprotocol/inspector http://localhost:8080/mcp
```

### Inspector Options

```bash
# Specify port (default: 5173)
npx @modelcontextprotocol/inspector ./bin/summary-sys mcp --port 3000

# Specify host (default: localhost)
npx @modelcontextprotocol/inspector ./bin/summary-sys mcp --port 3000 --host 0.0.0.0

# With custom transport args
npx @modelcontextprotocol/inspector ./bin/summary-sys mcp -- --transport sse --port 8080
```

### Troubleshooting

| Issue | Solution |
|-------|----------|
| Connection refused | Ensure MCP server is running before starting inspector |
| Tools not showing | Check if transport matches (stdio vs http) |
| Timeout errors | Increase timeout with `--timeout` flag |
