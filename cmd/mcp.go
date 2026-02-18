package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	mcpserver "github.com/SisyphusSQ/summary-sys/internal/mcp"
	l "github.com/SisyphusSQ/summary-sys/pkg/log"
	"github.com/SisyphusSQ/summary-sys/vars"
)

var (
	mcpTransport string
	mcpHost      string
	mcpPort      int
	mcpConfig    string
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server",
	Long: `Start MCP (Model Context Protocol) server.

Supported transports:
  - stdio: Standard input/output (default, for Claude/Cursor)
  - sse:   Server-Sent Events over HTTP
  - http:  Streamable HTTP

Examples:
  # Start with stdio (default)
  summary-sys mcp

  # Start with SSE
  summary-sys mcp --transport sse --host localhost --port 8080

  # Start with HTTP
  summary-sys mcp --transport http --host 0.0.0.0 --port 8080

  # Start all transports
  summary-sys mcp --transport all

  # Start with config file
  summary-sys mcp --config mcp.json
`,
	Annotations: map[string]string{
		forceStderrAnnotationKey: "true",
	},
	RunE: runMCP,
}

func init() {
	mcpCmd.Flags().StringVar(&mcpTransport, "transport", "stdio",
		"MCP transport: stdio, sse, http, all")
	mcpCmd.Flags().StringVar(&mcpHost, "host", "0.0.0.0",
		"Host for SSE/HTTP transport")
	mcpCmd.Flags().IntVar(&mcpPort, "port", 8080,
		"Port for SSE/HTTP transport")
	mcpCmd.Flags().StringVar(&mcpConfig, "config", "",
		"Path to MCP config JSON file")

	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	if mcpConfig != "" {
		return runMCPWithConfig(mcpConfig)
	}

	transport := mcpserver.Transport(mcpTransport)

	switch transport {
	case mcpserver.TransportStdio:
		l.Logger.Infof("starting mcp stdio server")
		return mcpserver.ServeStdio(vars.AppName, vars.AppVersion)

	case mcpserver.TransportSSE:
		l.Logger.Infof("starting mcp sse server on %s:%d", mcpHost, mcpPort)
		return mcpserver.ServeSSE(vars.AppName, vars.AppVersion, mcpHost, mcpPort)

	case mcpserver.TransportHTTP:
		l.Logger.Infof("starting mcp http server on %s:%d", mcpHost, mcpPort)
		return mcpserver.ServeHTTP(vars.AppName, vars.AppVersion, mcpHost, mcpPort)

	case "all":
		l.Logger.Infof("starting all mcp transports")
		return mcpserver.ServeAll(vars.AppName, vars.AppName, mcpHost, true, true, true)

	default:
		return fmt.Errorf("unknown transport: %s (use: stdio, sse, http, all)", mcpTransport)
	}
}

func runMCPWithConfig(configPath string) error {
	cfg, err := mcpserver.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	l.Logger.Infof("loaded config from %s", configPath)

	if cfg.Transport == "all" {
		return mcpserver.ServeAll(cfg.Name, cfg.Version, cfg.Host, cfg.EnableStdio, cfg.EnableSSE, cfg.EnableHTTP)
	}

	switch mcpserver.Transport(cfg.Transport) {
	case mcpserver.TransportStdio:
		return mcpserver.ServeStdio(cfg.Name, cfg.Version)
	case mcpserver.TransportSSE:
		return mcpserver.ServeSSE(cfg.Name, cfg.Version, cfg.Host, cfg.Port)
	case mcpserver.TransportHTTP:
		return mcpserver.ServeHTTP(cfg.Name, cfg.Version, cfg.Host, cfg.Port)
	default:
		return fmt.Errorf("unknown transport in config: %s", cfg.Transport)
	}
}

func initMCP() {
}
