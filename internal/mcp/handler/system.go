package handler

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/SisyphusSQ/summary-sys/internal/collector"
	"github.com/SisyphusSQ/summary-sys/internal/formatter"
)

type SystemCollectorService struct {
	localCollector *collector.LocalCollector
}

func NewSystemCollectorService() *SystemCollectorService {
	return &SystemCollectorService{
		localCollector: collector.NewLocalCollector(),
	}
}

func InitSystemTools(s *server.MCPServer, svc *SystemCollectorService) {
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
	result += fmt.Sprintf("Total: %s\n", formatBytes(mem.Total))
	result += fmt.Sprintf("Used: %s (%.1f%%)\n", formatBytes(mem.Used), mem.UsedPercent)
	result += fmt.Sprintf("Available: %s\n", formatBytes(mem.Available))
	if mem.SwapTotal > 0 {
		result += fmt.Sprintf("Swap: %s / %s (%.1f%%)\n", formatBytes(mem.SwapUsed), formatBytes(mem.SwapTotal), mem.SwapPercent)
	}
	return result
}

func formatDiskInfo(disk *collector.DiskInfo) string {
	result := "Disk Information:\n"
	for _, d := range *disk {
		result += fmt.Sprintf("%s: %s used / %s (%.1f%%)\n", d.MountPoint, formatBytes(d.Used), formatBytes(d.Total), d.UsedPercent)
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

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(b)/float64(div), "KMGTPE"[exp])
}
