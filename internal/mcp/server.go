package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"

	"github.com/SisyphusSQ/summary-sys/internal/mcp/handler"
)

type Transport string

const (
	TransportStdio Transport = "stdio"
	TransportSSE   Transport = "sse"
	TransportHTTP  Transport = "http"
)

type Config struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Transport   Transport `json:"transport"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	EnableStdio bool      `json:"enable_stdio"`
	EnableSSE   bool      `json:"enable_sse"`
	EnableHTTP  bool      `json:"enable_http"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

func NewServer(name, version string) *server.MCPServer {
	s := server.NewMCPServer(
		name,
		version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	systemSrv := handler.NewSystemCollectorService()
	handler.InitSystemTools(s, systemSrv)

	return s
}

func ServeStdio(name, version string) error {
	s := NewServer(name, version)
	if err := server.ServeStdio(s); err != nil {
		return fmt.Errorf("serve mcp stdio: %w", err)
	}
	return nil
}

func ServeSSE(name, version, host string, port int) error {
	s := NewServer(name, version)
	sseServer := server.NewSSEServer(s)

	addr := fmt.Sprintf("%s:%d", host, port)
	if err := sseServer.Start(addr); err != nil {
		return fmt.Errorf("serve mcp sse: %w", err)
	}

	fmt.Printf("MCP SSE server started on http://%s\n", addr)
	waitForSignal()
	return nil
}

func ServeHTTP(name, version, host string, port int) error {
	s := NewServer(name, version)
	httpServer := server.NewStreamableHTTPServer(s)

	addr := fmt.Sprintf("%s:%d", host, port)
	if err := httpServer.Start(addr); err != nil {
		return fmt.Errorf("serve mcp http: %w", err)
	}

	fmt.Printf("MCP HTTP server started on http://%s\n", addr)
	waitForSignal()
	return nil
}

func ServeAll(name, version, host string, stdio, sse, http bool) error {
	s := NewServer(name, version)

	errChan := make(chan error, 3)
	stopChan := make(chan struct{})

	if stdio {
		go func() {
			fmt.Println("MCP STDIO server started")
			if err := server.ServeStdio(s); err != nil {
				errChan <- fmt.Errorf("stdio: %w", err)
			}
		}()
	}

	if sse {
		go func() {
			addr := fmt.Sprintf("%s:%d", host, 8080)
			sseServer := server.NewSSEServer(s)
			fmt.Printf("MCP SSE server started on http://%s/mcp\n", addr)
			if err := sseServer.Start(addr); err != nil {
				errChan <- fmt.Errorf("sse: %w", err)
			}
		}()
	}

	if http {
		go func() {
			addr := fmt.Sprintf("%s:%d", host, 8081)
			httpServer := server.NewStreamableHTTPServer(s)
			fmt.Printf("MCP HTTP server started on http://%s/mcp\n", addr)
			if err := httpServer.Start(addr); err != nil {
				errChan <- fmt.Errorf("http: %w", err)
			}
		}()
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		close(stopChan)
	}()

	select {
	case err := <-errChan:
		return err
	case <-stopChan:
		fmt.Println("Shutting down MCP server...")
		return nil
	}
}

func waitForSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\nShutting down MCP server...")
}
