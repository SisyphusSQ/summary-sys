package formatter

import (
	"fmt"
	"strings"
	"time"

	"github.com/SisyphusSQ/summary-sys/internal/collector"
)

type TextFormatter struct{}

func NewTextFormatter() *TextFormatter {
	return &TextFormatter{}
}

func (f *TextFormatter) Name() string        { return "text" }
func (f *TextFormatter) ContentType() string { return "text/plain" }

func (f *TextFormatter) Format(info *collector.SystemInfo) (string, error) {
	var sb strings.Builder

	sb.WriteString("=" + strings.Repeat("=", 78) + "\n")
	sb.WriteString(fmt.Sprintf("# %s\n", info.Hostname))
	sb.WriteString(fmt.Sprintf("OS: %s %s\n", info.OS, info.Kernel))
	sb.WriteString(fmt.Sprintf("Uptime: %s\n", formatUptime(info.Uptime)))
	sb.WriteString(fmt.Sprintf("Time: %s\n", info.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString("=" + strings.Repeat("=", 78) + "\n\n")

	if info.CPU != nil {
		sb.WriteString(f.formatCPU(info.CPU))
	}

	if info.Memory != nil {
		sb.WriteString(f.formatMemory(info.Memory))
	}

	if info.Disk != nil {
		sb.WriteString(f.formatDisk(*info.Disk))
	}

	if info.Network != nil {
		sb.WriteString(f.formatNetwork(info.Network))
	}

	if info.LoadAvg != nil {
		sb.WriteString(f.formatLoadAvg(info.LoadAvg))
	}

	if info.Process != nil {
		sb.WriteString(f.formatProcess(info.Process))
	}

	return sb.String(), nil
}

func (f *TextFormatter) formatCPU(cpu *collector.CPUInfo) string {
	var sb strings.Builder
	sb.WriteString("\n### CPU ###\n")
	sb.WriteString(fmt.Sprintf("Cores: %d Physical / %d Logical\n", cpu.PhysicalCores, cpu.LogicalCores))
	if len(cpu.Models) > 0 {
		sb.WriteString(fmt.Sprintf("Model: %s\n", cpu.Models[0]))
	}
	sb.WriteString(fmt.Sprintf("Usage: %.1f%%\n", cpu.UsagePercent))
	return sb.String()
}

func (f *TextFormatter) formatMemory(mem *collector.MemoryInfo) string {
	var sb strings.Builder
	sb.WriteString("\n### Memory ###\n")
	sb.WriteString(fmt.Sprintf("Total:     %s\n", formatBytes(mem.Total)))
	sb.WriteString(fmt.Sprintf("Used:      %s (%.1f%%)\n", formatBytes(mem.Used), mem.UsedPercent))
	sb.WriteString(fmt.Sprintf("Available: %s\n", formatBytes(mem.Available)))
	if mem.SwapTotal > 0 {
		sb.WriteString(fmt.Sprintf("Swap:      %s / %s (%.1f%%)\n",
			formatBytes(mem.SwapUsed), formatBytes(mem.SwapTotal), mem.SwapPercent))
	}
	return sb.String()
}

func (f *TextFormatter) formatDisk(disk collector.DiskInfo) string {
	var sb strings.Builder
	sb.WriteString("\n### Disk ###\n")

	if len(disk) == 0 {
		return sb.String()
	}

	minWidth := 20
	maxMountLen := len("Filesystem")
	for _, d := range disk {
		if len(d.MountPoint) > maxMountLen {
			maxMountLen = len(d.MountPoint)
		}
	}
	if maxMountLen < minWidth {
		maxMountLen = minWidth
	}

	fsWidth := maxMountLen
	sb.WriteString(fmt.Sprintf("%-*s %10s %10s %10s %8s\n", fsWidth, "Filesystem", "Size", "Used", "Avail", "Use%"))
	sb.WriteString(strings.Repeat("-", fsWidth+40) + "\n")
	for _, d := range disk {
		sb.WriteString(fmt.Sprintf("%-*s %10s %10s %10s %7.1f%%\n",
			fsWidth, d.MountPoint, formatBytes(d.Total), formatBytes(d.Used),
			formatBytes(d.Available), d.UsedPercent))
	}
	return sb.String()
}

func (f *TextFormatter) formatNetwork(net *collector.NetworkInfo) string {
	var sb strings.Builder
	sb.WriteString("\n### Network ###\n")

	if len(net.Interfaces) == 0 {
		sb.WriteString("No network interfaces\n")
		return sb.String()
	}

	maxNameLen := 10
	for _, iface := range net.Interfaces {
		if len(iface.Name) > maxNameLen {
			maxNameLen = len(iface.Name)
		}
	}

	for _, iface := range net.Interfaces {
		sb.WriteString(fmt.Sprintf("%-*s: %s\n", maxNameLen, iface.Name, strings.Join(iface.Addrs, ", ")))
		if iface.Statistics != nil {
			sb.WriteString(fmt.Sprintf("%-*s RX: %s TX: %s\n", maxNameLen+2, "",
				formatBytes(iface.Statistics.BytesRecv),
				formatBytes(iface.Statistics.BytesSent)))
		}
	}
	sb.WriteString(fmt.Sprintf("TCP Connections: %d\n", net.NetstatTCP))
	sb.WriteString(fmt.Sprintf("UDP Connections: %d\n", net.NetstatUDP))
	return sb.String()
}

func (f *TextFormatter) formatLoadAvg(load *collector.LoadAvgInfo) string {
	var sb strings.Builder
	sb.WriteString("\n### Load Average ###\n")
	sb.WriteString(fmt.Sprintf("1min:  %.2f\n", load.Load1))
	sb.WriteString(fmt.Sprintf("5min:  %.2f\n", load.Load5))
	sb.WriteString(fmt.Sprintf("15min: %.2f\n", load.Load15))
	sb.WriteString(fmt.Sprintf("Processes: %d running / %d total\n", load.Runable, load.Total))
	return sb.String()
}

func (f *TextFormatter) formatProcess(proc *collector.ProcessInfo) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n### Processes (%d total) ###\n", proc.Count))

	if len(proc.TopCPU) == 0 && len(proc.TopMemory) == 0 {
		return sb.String()
	}

	pidWidth := 8
	userWidth := 15
	cmdWidth := 15
	for _, p := range proc.TopCPU {
		if len(p.Name) > cmdWidth {
			cmdWidth = len(p.Name)
		}
	}
	for _, p := range proc.TopMemory {
		if len(p.Name) > cmdWidth {
			cmdWidth = len(p.Name)
		}
	}

	if len(proc.TopCPU) > 0 {
		sb.WriteString("\nTop 10 CPU:\n")
		sb.WriteString(fmt.Sprintf("%-*s %-*s %8s %8s %s\n", pidWidth, "PID", userWidth, "USER", "CPU%", "MEM%", "COMMAND"))
		for _, p := range proc.TopCPU {
			sb.WriteString(fmt.Sprintf("%-*d %-*s %7.1f %7.1f %s\n",
				pidWidth, p.PID, userWidth, p.User, p.CPUPercent, p.MemPercent, p.Name))
		}
	}

	if len(proc.TopMemory) > 0 {
		sb.WriteString("\nTop 10 Memory:\n")
		sb.WriteString(fmt.Sprintf("%-*s %-*s %8s %8s %s\n", pidWidth, "PID", userWidth, "USER", "CPU%", "MEM%", "COMMAND"))
		for _, p := range proc.TopMemory {
			sb.WriteString(fmt.Sprintf("%-*d %-*s %7.1f %7.1f %s\n",
				pidWidth, p.PID, userWidth, p.User, p.CPUPercent, p.MemPercent, p.Name))
		}
	}
	return sb.String()
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

func formatUptime(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Minute
	m := d / time.Minute
	if h > 24 {
		days := h / 24
		h = h % 24
		return fmt.Sprintf("%dd %dh %dm", days, h, m)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}
