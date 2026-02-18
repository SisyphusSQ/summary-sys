package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/SisyphusSQ/summary-sys/internal/collector"
	"github.com/SisyphusSQ/summary-sys/internal/formatter"
	"github.com/SisyphusSQ/summary-sys/internal/ssh"
	"github.com/SisyphusSQ/summary-sys/utils/format"
)

type SystemCollectorService struct {
	localCollector *collector.LocalCollector
}

type RemoteCollectorService struct {
	timeout  time.Duration
	parallel int
}

func NewSystemCollectorService() *SystemCollectorService {
	return &SystemCollectorService{
		localCollector: collector.NewLocalCollector(),
	}
}

func NewRemoteCollectorService() *RemoteCollectorService {
	return &RemoteCollectorService{
		timeout:  30 * time.Second,
		parallel: 5,
	}
}

// collectRemote collects system info from a single remote host via SSH
func (s *RemoteCollectorService) collectRemote(ctx context.Context, host, user, password, keyPath string, port int) (*collector.SystemInfo, error) {
	var authMethod ssh.AuthMethod

	if keyPath != "" {
		authMethod = ssh.KeyAuth{KeyPath: keyPath}
	} else if password != "" {
		authMethod = ssh.PasswordAuth{Password: password}
	} else {
		// Try default SSH keys
		authMethod = ssh.DefaultKeyAuth{}
	}

	sshClient, err := ssh.NewClient(&ssh.Config{
		Host:       host,
		Port:       port,
		User:       user,
		AuthMethod: authMethod,
		Timeout:    s.timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("SSH connection failed: %w", err)
	}
	defer sshClient.Close()

	remoteCollector, err := collector.NewRemoteCollector(sshClient)
	if err != nil {
		return nil, fmt.Errorf("create remote collector: %w", err)
	}

	return remoteCollector.Collect(ctx)
}

// collectRemoteBatch collects system info from multiple remote hosts in parallel
func (s *RemoteCollectorService) collectRemoteBatch(ctx context.Context, hosts []string, user, password, keyPath string, port int) ([]RemoteHostResult, error) {
	var wg sync.WaitGroup
	results := make([]RemoteHostResult, len(hosts))
	sem := make(chan struct{}, s.parallel)

	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			info, err := s.collectRemote(ctx, h, user, password, keyPath, port)
			results[idx] = RemoteHostResult{
				Host: h,
				Info: info,
				Err:  err,
			}
		}(i, host)
	}

	wg.Wait()
	return results, nil
}

// RemoteHostResult holds the result of a remote collection attempt
type RemoteHostResult struct {
	Host string
	Info *collector.SystemInfo
	Err  error
}

func InitRemoteTools(s *server.MCPServer, remoteSvc *RemoteCollectorService) {
	remoteSummaryTool := mcp.NewTool(
		"system_summary_remote",
		mcp.WithDescription("Get system summary from a remote host via SSH"),
		mcp.WithString("host",
			mcp.Required(),
			mcp.Description("Remote SSH host IP or hostname"),
		),
		mcp.WithNumber("port",
			mcp.DefaultNumber(22),
			mcp.Description("SSH port"),
		),
		mcp.WithString("user",
			mcp.Description("SSH username (default: root)"),
		),
		mcp.WithString("password",
			mcp.Description("SSH password (optional if using key)"),
		),
		mcp.WithString("key_path",
			mcp.Description("SSH private key path (optional)"),
		),
	)

	s.AddTool(remoteSummaryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		host, err := request.RequireString("host")
		if err != nil {
			return mcp.NewToolResultError("host parameter is required"), nil
		}

		port := 22
		if p, ok := args["port"]; ok {
			if pf, ok := p.(float64); ok {
				port = int(pf)
			}
		}

		user := "root"
		if u, ok := args["user"].(string); ok {
			user = u
		}

		password, _ := args["password"].(string)
		keyPath, _ := args["key_path"].(string)

		info, err := remoteSvc.collectRemote(ctx, host, user, password, keyPath, port)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to collect from %s: %v", host, err)), nil
		}

		fmter, err := formatter.NewFormatter("json")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		output, err := fmter.Format(info)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(output), nil
	})

	remoteBatchTool := mcp.NewTool(
		"system_summary_remote_batch",
		mcp.WithDescription("Get system summary from multiple remote hosts via SSH in parallel"),
		mcp.WithArray("hosts",
			mcp.Required(),
			mcp.Description("List of remote SSH hosts"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithNumber("port",
			mcp.DefaultNumber(22),
			mcp.Description("SSH port"),
		),
		mcp.WithString("user",
			mcp.Description("SSH username (default: root)"),
		),
		mcp.WithString("password",
			mcp.Description("SSH password (optional if using key)"),
		),
		mcp.WithString("key_path",
			mcp.Description("SSH private key path (optional)"),
		),
		mcp.WithNumber("parallel",
			mcp.DefaultNumber(5),
			mcp.Description("Number of parallel connections"),
		),
	)

	s.AddTool(remoteBatchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		var hosts []string
		if hostsArg, ok := args["hosts"]; ok {
			switch v := hostsArg.(type) {
			case []any:
				for _, h := range v {
					if hStr, ok := h.(string); ok {
						hosts = append(hosts, hStr)
					}
				}
			case string:
				hosts = strings.Split(v, ",")
			}
		}

		if len(hosts) == 0 {
			return mcp.NewToolResultError("hosts parameter is required"), nil
		}

		port := 22
		if p, ok := args["port"]; ok {
			if pf, ok := p.(float64); ok {
				port = int(pf)
			}
		}

		user := "root"
		if u, ok := args["user"].(string); ok {
			user = u
		}

		password, _ := args["password"].(string)
		keyPath, _ := args["key_path"].(string)

		parallel := 5
		if par, ok := args["parallel"]; ok {
			if pf, ok := par.(float64); ok {
				parallel = int(pf)
			}
		}

		remoteSvc.parallel = parallel
		results, err := remoteSvc.collectRemoteBatch(ctx, hosts, user, password, keyPath, port)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Batch collection failed: %v", err)), nil
		}

		output, err := formatRemoteResults(results)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(output), nil
	})
}

func formatRemoteResults(results []RemoteHostResult) (string, error) {
	var sb strings.Builder
	sb.WriteString("Remote Collection Results:\n")
	sb.WriteString(strings.Repeat("-", 50))
	sb.WriteString("\n")

	successCount := 0
	failCount := 0

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("\nHost: %s\n", r.Host))
		if r.Err != nil {
			sb.WriteString(fmt.Sprintf("  Status: FAILED\n"))
			sb.WriteString(fmt.Sprintf("  Error: %v\n", r.Err))
			failCount++
		} else if r.Info != nil {
			sb.WriteString(fmt.Sprintf("  Status: SUCCESS\n"))
			sb.WriteString(fmt.Sprintf("  Hostname: %s\n", r.Info.Hostname))
			sb.WriteString(fmt.Sprintf("  OS: %s\n", r.Info.OS))
			if r.Info.CPU != nil {
				sb.WriteString(fmt.Sprintf("  CPU Cores: %d\n", r.Info.CPU.PhysicalCores))
			}
			if r.Info.Memory != nil {
				sb.WriteString(fmt.Sprintf("  Memory: %s / %s (%.1f%%)\n",
					format.FormatBytes(r.Info.Memory.Used),
					format.FormatBytes(r.Info.Memory.Total),
					r.Info.Memory.UsedPercent))
			}
			if r.Info.LoadAvg != nil {
				sb.WriteString(fmt.Sprintf("  Load: %.2f / %.2f / %.2f\n",
					r.Info.LoadAvg.Load1, r.Info.LoadAvg.Load5, r.Info.LoadAvg.Load15))
			}
			successCount++
		}
	}

	sb.WriteString(strings.Repeat("-", 50))
	sb.WriteString(fmt.Sprintf("\nSummary: %d succeeded, %d failed\n", successCount, failCount))

	return sb.String(), nil
}

func InitSystemTools(s *server.MCPServer, svc *SystemCollectorService) {
	InitRemoteTools(s, NewRemoteCollectorService())

	summaryTool := mcp.NewTool(
		"system_summary_local",
		mcp.WithDescription("Get local system summary including CPU, memory, disk, network, processes and load average"),
	)

	s.AddTool(summaryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info, err := svc.localCollector.Collect(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		fmter, err := formatter.NewFormatter("json")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		output, err := fmter.Format(info)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(output), nil
	})

	cpuTool := mcp.NewTool(
		"system_cpu",
		mcp.WithDescription("Get CPU information including cores, model and usage"),
	)

	s.AddTool(cpuTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info, err := svc.localCollector.Collect(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if info.CPU == nil {
			return mcp.NewToolResultText("No CPU info available"), nil
		}
		return mcp.NewToolResultText(formatCPUInfo(info.CPU)), nil
	})

	memoryTool := mcp.NewTool(
		"system_memory",
		mcp.WithDescription("Get memory information including total, used, available and swap"),
	)

	s.AddTool(memoryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info, err := svc.localCollector.Collect(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if info.Memory == nil {
			return mcp.NewToolResultText("No memory info available"), nil
		}
		return mcp.NewToolResultText(formatMemoryInfo(info.Memory)), nil
	})

	diskTool := mcp.NewTool(
		"system_disk",
		mcp.WithDescription("Get disk usage information including partitions and usage percentage"),
	)

	s.AddTool(diskTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info, err := svc.localCollector.Collect(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if info.Disk == nil {
			return mcp.NewToolResultText("No disk info available"), nil
		}
		return mcp.NewToolResultText(formatDiskInfo(info.Disk)), nil
	})

	networkTool := mcp.NewTool(
		"system_network",
		mcp.WithDescription("Get network information including interfaces and connection stats"),
	)

	s.AddTool(networkTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info, err := svc.localCollector.Collect(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if info.Network == nil {
			return mcp.NewToolResultText("No network info available"), nil
		}
		return mcp.NewToolResultText(formatNetworkInfo(info.Network)), nil
	})

	processTool := mcp.NewTool(
		"system_processes",
		mcp.WithDescription("Get process information including top CPU and memory consumers"),
	)

	s.AddTool(processTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info, err := svc.localCollector.Collect(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if info.Process == nil {
			return mcp.NewToolResultText("No process info available"), nil
		}
		return mcp.NewToolResultText(formatProcessInfo(info.Process)), nil
	})

	loadTool := mcp.NewTool(
		"system_loadavg",
		mcp.WithDescription("Get system load average information"),
	)

	s.AddTool(loadTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info, err := svc.localCollector.Collect(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if info.LoadAvg == nil {
			return mcp.NewToolResultText("No load avg info available"), nil
		}
		return mcp.NewToolResultText(formatLoadAvgInfo(info.LoadAvg)), nil
	})

	infoTool := mcp.NewTool(
		"system_info",
		mcp.WithDescription("Get basic system information including hostname, OS, kernel and uptime"),
	)

	s.AddTool(infoTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info, err := svc.localCollector.Collect(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(formatSystemInfo(info)), nil
	})
}

func formatCPUInfo(cpu *collector.CPUInfo) string {
	result := "CPU Information:\n"
	result += fmt.Sprintf("Physical Cores: %d\n", cpu.PhysicalCores)
	result += fmt.Sprintf("Logical Cores: %d\n", cpu.LogicalCores)
	if len(cpu.Models) > 0 {
		result += fmt.Sprintf("Model: %s\n", cpu.Models[0])
	}
	result += fmt.Sprintf("Usage: %.1f%%\n", cpu.UsagePercent)
	return result
}

func formatMemoryInfo(mem *collector.MemoryInfo) string {
	result := "Memory Information:\n"
	result += fmt.Sprintf("Total: %s\n", format.FormatBytes(mem.Total))
	result += fmt.Sprintf("Used: %s (%.1f%%)\n", format.FormatBytes(mem.Used), mem.UsedPercent)
	result += fmt.Sprintf("Available: %s\n", format.FormatBytes(mem.Available))
	if mem.SwapTotal > 0 {
		result += fmt.Sprintf("Swap: %s / %s (%.1f%%)\n", format.FormatBytes(mem.SwapUsed), format.FormatBytes(mem.SwapTotal), mem.SwapPercent)
	}
	return result
}

func formatDiskInfo(disk *collector.DiskInfo) string {
	result := "Disk Information:\n"
	for _, d := range *disk {
		result += fmt.Sprintf("%s: %s used / %s (%.1f%%)\n", d.MountPoint, format.FormatBytes(d.Used), format.FormatBytes(d.Total), d.UsedPercent)
	}
	return result
}

func formatNetworkInfo(net *collector.NetworkInfo) string {
	result := "Network Information:\n"
	for _, iface := range net.Interfaces {
		if len(iface.Addrs) > 0 {
			result += fmt.Sprintf("%s: %s\n", iface.Name, iface.Addrs[0])
		}
	}
	result += fmt.Sprintf("TCP Connections: %d\n", net.NetstatTCP)
	result += fmt.Sprintf("UDP Connections: %d\n", net.NetstatUDP)
	return result
}

func formatProcessInfo(proc *collector.ProcessInfo) string {
	result := fmt.Sprintf("Total Processes: %d\n\n", proc.Count)
	result += "Top 5 CPU:\n"
	for i, p := range proc.TopCPU {
		if i >= 5 {
			break
		}
		result += fmt.Sprintf("  %d: %s (%.1f%%)\n", p.PID, p.Name, p.CPUPercent)
	}
	result += "\nTop 5 Memory:\n"
	for i, p := range proc.TopMemory {
		if i >= 5 {
			break
		}
		result += fmt.Sprintf("  %d: %s (%.1f%%)\n", p.PID, p.Name, p.MemPercent)
	}
	return result
}

func formatLoadAvgInfo(load *collector.LoadAvgInfo) string {
	result := "Load Average:\n"
	result += fmt.Sprintf("1min: %.2f\n", load.Load1)
	result += fmt.Sprintf("5min: %.2f\n", load.Load5)
	result += fmt.Sprintf("15min: %.2f\n", load.Load15)
	result += fmt.Sprintf("Running: %d / %d total\n", load.Runable, load.Total)
	return result
}

func formatSystemInfo(info *collector.SystemInfo) string {
	result := "System Information:\n"
	result += fmt.Sprintf("Hostname: %s\n", info.Hostname)
	result += fmt.Sprintf("OS: %s\n", info.OS)
	result += fmt.Sprintf("Platform: %s\n", info.Platform)
	result += fmt.Sprintf("Kernel: %s\n", info.Kernel)
	result += fmt.Sprintf("Architecture: %s\n", info.Architecture)
	result += fmt.Sprintf("Uptime: %s\n", info.Uptime)
	return result
}
