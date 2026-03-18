package formatter

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/SisyphusSQ/summary-sys/internal/collector"
	"github.com/SisyphusSQ/summary-sys/utils/format"
)

type TextFormatter struct{}

func NewTextFormatter() *TextFormatter {
	return &TextFormatter{}
}

func (f *TextFormatter) Name() string        { return "text" }
func (f *TextFormatter) ContentType() string { return "text/plain" }

func (f *TextFormatter) Format(info *collector.SystemInfo) (string, error) {
	return f.formatPercona(info), nil
}

func (f *TextFormatter) formatPercona(info *collector.SystemInfo) string {
	var sb strings.Builder

	// Get actual timezone - use system timezone not runtime location
	tzName, tzOffset := "CST", "+0800"
	if runtime.GOOS == "linux" {
		// Try to read timezone from /etc/localtime or /etc/timezone
		if tzData, err := os.ReadFile("/etc/timezone"); err == nil {
			tzName = strings.TrimSpace(string(tzData))
		}
		// Get offset
		_, offset := time.Now().Zone()
		tzOffset = fmt.Sprintf("%+d00", offset/3600)
	} else {
		tzName = time.Now().Location().String()
		_, offset := time.Now().Zone()
		tzOffset = fmt.Sprintf("%+d00", offset/3600)
	}

	// Header
	sb.WriteString("# Percona Toolkit System Summary Report " + strings.Repeat("#", 23) + "\n")
	sb.WriteString(fmt.Sprintf("        Date | %s (local TZ: %s %s)\n",
		info.Timestamp.Format("2006-01-02 15:04:05"), tzName, tzOffset))
	sb.WriteString(fmt.Sprintf("    Hostname | %s\n", info.Hostname))

	// Calculate user count from Who info
	userCount := 0
	if info.Who != nil {
		userCount = len(info.Who)
	}

	if info.LoadAvg != nil {
		uptimeDays := int(info.Uptime.Hours() / 24)
		uptimeHours := int(info.Uptime.Hours()) % 24
		uptimeMins := int(info.Uptime.Minutes()) % 60
		sb.WriteString(fmt.Sprintf("      Uptime | %d days, %d:%02d,  %d users,  load average: %.2f, %.2f, %.2f\n",
			uptimeDays, uptimeHours, uptimeMins, userCount, info.LoadAvg.Load1, info.LoadAvg.Load5, info.LoadAvg.Load15))
	} else {
		sb.WriteString(fmt.Sprintf("      Uptime | %s\n", format.FormatUptime(info.Uptime)))
	}

	// System info
	if info.SystemExt != nil {
		sb.WriteString(fmt.Sprintf("      System | %s; %s; %s\n",
			firstNonEmpty(info.SystemExt.Vendor, "Unknown"),
			firstNonEmpty(info.SystemExt.Model, "Unknown"),
			firstNonEmpty(info.SystemExt.Version, "v00001")))
		sb.WriteString(fmt.Sprintf(" Service Tag | %s\n", firstNonEmpty(info.SystemExt.ServiceTag, "Unknown")))
	} else {
		sb.WriteString("      System | Unknown\n")
		sb.WriteString(" Service Tag | Unknown\n")
	}

	sb.WriteString(fmt.Sprintf("    Platform | %s\n", firstNonEmpty(info.Platform, "Linux")))
	if info.OS != "" {
		sb.WriteString(fmt.Sprintf("     Release | %s\n", info.OS))
	} else {
		sb.WriteString("     Release | Unknown\n")
	}
	sb.WriteString(fmt.Sprintf("      Kernel | %s\n", firstNonEmpty(info.Kernel, "Unknown")))
	sb.WriteString(fmt.Sprintf("Architecture | CPU = %s, OS = %s\n",
		firstNonEmpty(info.Architecture, "64-bit"), runtime.GOARCH))

	if info.SystemExt != nil {
		sb.WriteString(fmt.Sprintf("   Threading | %s\n", firstNonEmpty(info.SystemExt.Threading, "NPTL")))
		sb.WriteString(fmt.Sprintf("     SELinux | %s\n", firstNonEmpty(info.SystemExt.SELinux, "Disabled")))
		sb.WriteString(fmt.Sprintf(" Virtualized | %s\n", firstNonEmpty(info.SystemExt.Virtualized, "No virtualization detected")))
	} else {
		sb.WriteString("   Threading | NPTL\n")
		sb.WriteString("     SELinux | Disabled\n")
		sb.WriteString(" Virtualized | No virtualization detected\n")
	}

	// Processor
	sb.WriteString("# Processor " + strings.Repeat("#", 49) + "\n")
	if info.CPU != nil {
		ht := "no"
		if info.CPUExt != nil && info.CPUExt.Hyperthreading {
			ht = "yes"
		}
		// Use sockets and cores per socket if available, otherwise physical=cores
		physical := info.CPU.PhysicalCores
		cores := info.CPU.PhysicalCores
		if info.CPU.Sockets > 0 {
			cores = info.CPU.CoresPerSocket
			physical = info.CPU.Sockets
		}
		sb.WriteString(fmt.Sprintf("  Processors | physical = %d, cores = %d, virtual = %d, hyperthreading = %s\n",
			physical, cores, info.CPU.LogicalCores, ht))
	}

	if info.CPUExt != nil && len(info.CPUExt.Speeds) > 0 {
		sb.WriteString("      Speeds | ")
		speedStrs := make([]string, 0)
		speedCounts := make(map[float64]int)
		for _, s := range info.CPUExt.Speeds {
			speedCounts[s]++
		}
		for speed, count := range speedCounts {
			if count > 1 {
				speedStrs = append(speedStrs, fmt.Sprintf("%dx%.3f", count, speed))
			} else {
				speedStrs = append(speedStrs, fmt.Sprintf("%.3f", speed))
			}
		}
		sb.WriteString(strings.Join(speedStrs, ", "))
		sb.WriteString("\n")
	} else {
		sb.WriteString("      Speeds | N/A\n")
	}

	if info.CPU != nil && len(info.CPU.Models) > 0 {
		modelStr := info.CPU.Models[0]
		count := 1
		if len(info.CPU.Models) > 1 {
			for i := 1; i < len(info.CPU.Models); i++ {
				if info.CPU.Models[i] == modelStr {
					count++
				}
			}
		}
		sb.WriteString(fmt.Sprintf("      Models | %dx%s\n", count, modelStr))
	}

	if info.CPUExt != nil && len(info.CPUExt.Caches) > 0 {
		// Count total caches (each cache level * number of sockets)
		totalCaches := len(info.CPUExt.Caches)
		if info.CPU != nil && info.CPU.Sockets > 0 {
			totalCaches = info.CPU.Sockets * len(info.CPUExt.Caches)
		}
		lastCache := info.CPUExt.Caches[len(info.CPUExt.Caches)-1]
		sb.WriteString(fmt.Sprintf("      Caches | %dx%d KB\n", totalCaches, lastCache.Size/1024))
		sb.WriteString("  Designation               Configuration                  Size     Associativity\n")
		sb.WriteString("  ========================= ============================== ======== ======================\n")
		for _, c := range info.CPUExt.Caches {
			levelNum := strings.TrimPrefix(c.Level, "L")
			sb.WriteString(fmt.Sprintf("  %-24s Enabled, Not Socketed, %-7s %d kB   %s\n",
				c.Level+"-Cache", "Level "+levelNum, c.Size/1024, c.Associativity))
		}
	}

	// Memory
	sb.WriteString("# Memory " + strings.Repeat("#", 49) + "\n")
	if info.Memory != nil {
		sb.WriteString(fmt.Sprintf("         Total | %s\n", format.FormatBytes(info.Memory.Total)))
		if info.MemoryExt != nil {
			sb.WriteString(fmt.Sprintf("          Free | %s\n", format.FormatBytes(info.MemoryExt.Free)))
			usedPhys := info.Memory.Used
			swapAlloc := info.Memory.SwapTotal
			swapUsed := info.Memory.SwapUsed
			sb.WriteString(fmt.Sprintf("          Used | physical = %s, swap allocated = %s, swap used = %s, virtual = %s\n",
				format.FormatBytes(usedPhys), format.FormatBytes(swapAlloc), format.FormatBytes(swapUsed), format.FormatBytes(usedPhys)))
		} else {
			sb.WriteString(fmt.Sprintf("          Used | %s (%.1f%%)\n", format.FormatBytes(info.Memory.Used), info.Memory.UsedPercent))
		}
	}

	if info.MemoryExt != nil {
		sb.WriteString(fmt.Sprintf("        Shared | %s\n", format.FormatBytes(info.MemoryExt.Shared)))
		sb.WriteString(fmt.Sprintf("       Buffers | %s\n", format.FormatBytes(info.MemoryExt.Buffers)))
		sb.WriteString(fmt.Sprintf("        Caches | %s\n", format.FormatBytes(info.MemoryExt.Caches)))
		sb.WriteString(fmt.Sprintf("         Dirty | %s\n", format.FormatBytes(info.MemoryExt.Dirty)))
		sb.WriteString(fmt.Sprintf("       UsedRSS | %s\n", format.FormatBytes(info.MemoryExt.UsedRSS)))
		sb.WriteString(fmt.Sprintf("    Swappiness | %d\n", info.MemoryExt.Swappiness))
		sb.WriteString(fmt.Sprintf("   DirtyPolicy | %s\n", firstNonEmpty(info.MemoryExt.DirtyPolicy, "20, 10")))
		sb.WriteString(fmt.Sprintf("   DirtyStatus | %s\n", firstNonEmpty(info.MemoryExt.DirtyStatus, "0, 0")))

		// NUMA Nodes
		if len(info.MemoryExt.NumaNodes) > 0 {
			sb.WriteString(fmt.Sprintf("    Numa Nodes | %d\n", len(info.MemoryExt.NumaNodes)))
			sb.WriteString("   Numa Policy | default\n")
			sb.WriteString("Preferred Node | current\n")
			sb.WriteString("   Node    Size        Free        CPUs\n")
			sb.WriteString("   ====    ====        ====        ====\n")
			for _, node := range info.MemoryExt.NumaNodes {
				cpus := ""
				if len(node.CPUs) > 0 {
					// Format CPUs as ranges or comma-separated
					if len(node.CPUs) <= 10 {
						for i, c := range node.CPUs {
							if i > 0 {
								cpus += " "
							}
							cpus += fmt.Sprintf("%d", c)
						}
					} else {
						cpus = fmt.Sprintf("%d CPUs", len(node.CPUs))
					}
				}
				sb.WriteString(fmt.Sprintf("   node%d   %d MB    %d MB      %s\n",
					node.ID, node.Size/(1024*1024), node.Free/(1024*1024), cpus))
			}
		}

		// DIMM info
		if len(info.MemoryExt.Dimms) > 0 {
			sb.WriteString("\n  Locator   Size     Speed             Form Factor   Type          Type Detail\n")
			sb.WriteString("  ========= ======== ================= ============= ============= ===========\n")
			for _, dimm := range info.MemoryExt.Dimms {
				sb.WriteString(fmt.Sprintf("  %-8s %7s %-16s %-12s %-11s %s\n",
					truncateStr(dimm.Locator, 8),
					format.FormatBytes(dimm.Size),
					truncateStr(dimm.Speed, 16),
					truncateStr(dimm.FormFactor, 12),
					truncateStr(dimm.Type, 11),
					truncateStr(dimm.TypeDetail, 8)))
			}
		}
	}

	// Mounted Filesystems
	sb.WriteString("# Mounted Filesystems " + strings.Repeat("#", 39) + "\n")
	sb.WriteString("  Filesystem                      Size  Used Type     Opts                                                  Mountpoint\n")
	if info.Disk != nil {
		for _, d := range *info.Disk {
			sb.WriteString(fmt.Sprintf("  %-30s %7s %5s%% %-8s %-55s %s\n",
				truncateStr(d.Device, 30),
				format.FormatBytes(d.Total),
				fmt.Sprintf("%.0f", d.UsedPercent),
				d.FSType,
				"rw,relatime",
				truncateStr(d.MountPoint, 55)))
		}
	}

	// Disk Schedulers - only show on Linux
	if runtime.GOOS == "linux" && info.DiskExt != nil && len(info.DiskExt.Schedulers) > 0 {
		sb.WriteString("# Disk Schedulers And Queue Size " + strings.Repeat("#", 31) + "\n")
		for dev, sched := range info.DiskExt.Schedulers {
			sb.WriteString(fmt.Sprintf("     %s | %s\n", dev, sched))
		}
	}

	// Kernel Inode State - only show on Linux
	if runtime.GOOS == "linux" && info.KernelExt != nil {
		sb.WriteString("# Kernel Inode State " + strings.Repeat("#", 42) + "\n")
		if info.KernelExt.DentryState != "" {
			sb.WriteString(fmt.Sprintf("dentry-state | %s\n", info.KernelExt.DentryState))
		}
		if info.KernelExt.FileNr != "" {
			sb.WriteString(fmt.Sprintf("     file-nr | %s\n", info.KernelExt.FileNr))
		}
		if info.KernelExt.InodeNr != "" {
			sb.WriteString(fmt.Sprintf("    inode-nr | %s\n", info.KernelExt.InodeNr))
		}
	}

	// Network Config
	sb.WriteString("# Network Config " + strings.Repeat("#", 45) + "\n")
	sb.WriteString("  Controller | N/A\n")

	// Interface Statistics
	sb.WriteString("# Interface Statistics " + strings.Repeat("#", 37) + "\n")
	sb.WriteString("  interface          rx_bytes   rx_packets    rx_errors   tx_bytes   tx_packets    tx_errors\n")
	sb.WriteString("  ========== ============ ============ =========== ============ ============ ============\n")
	if info.Network != nil {
		for _, iface := range info.Network.Interfaces {
			rxBytes := uint64(0)
			txBytes := uint64(0)
			rxPackets := uint64(0)
			txPackets := uint64(0)
			rxErrors := uint64(0)
			txErrors := uint64(0)
			if iface.Statistics != nil {
				rxBytes = iface.Statistics.BytesRecv
				txBytes = iface.Statistics.BytesSent
				rxPackets = iface.Statistics.PacketsRecv
				txPackets = iface.Statistics.PacketsSent
				rxErrors = iface.Statistics.ErrIn
				txErrors = iface.Statistics.ErrOut
			}
			sb.WriteString(fmt.Sprintf("  %-11s %12s %12s %12s %12s %12s %12s\n",
				truncateStr(iface.Name, 11),
				format.FormatBytes(rxBytes),
				fmt.Sprintf("%d", rxPackets),
				fmt.Sprintf("%d", rxErrors),
				format.FormatBytes(txBytes),
				fmt.Sprintf("%d", txPackets),
				fmt.Sprintf("%d", txErrors)))
		}
	}

	// Network Connections
	if info.NetworkExt != nil && (len(info.NetworkExt.ConnFromRemote) > 0 || len(info.NetworkExt.ConnToPorts) > 0) {
		sb.WriteString("# Network Connections " + strings.Repeat("#", 40) + "\n")

		if len(info.NetworkExt.ConnFromRemote) > 0 {
			sb.WriteString("  Connections from remote IP addresses\n")
			count := 0
			for ip, cnt := range info.NetworkExt.ConnFromRemote {
				sb.WriteString(fmt.Sprintf("    %-30s %d\n", ip, cnt))
				count++
				if count >= 20 {
					sb.WriteString("    ... (truncated)\n")
					break
				}
			}
		}

		if len(info.NetworkExt.ConnToPorts) > 0 {
			sb.WriteString("  Connections to top 10 local ports\n")
			for _, p := range info.NetworkExt.ConnToPorts {
				sb.WriteString(fmt.Sprintf("    %-8d %d\n", p.Port, p.Count))
			}
		}

		if len(info.NetworkExt.ConnStates) > 0 {
			sb.WriteString("  States of connections\n")
			for state, cnt := range info.NetworkExt.ConnStates {
				sb.WriteString(fmt.Sprintf("    %-15s %d\n", state, cnt))
			}
		}
	}

	// Top Processes
	sb.WriteString("# Top Processes " + strings.Repeat("#", 46) + "\n")
	sb.WriteString("   PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND\n")
	if info.Process != nil {
		for _, p := range info.Process.TopCPU {
			sb.WriteString(fmt.Sprintf(" %6d %-8s %3d %3d %8s %7s %5s %1s  %4.1f %4.1f %10s %-20s\n",
				p.PID,
				truncateStr(p.User, 8),
				20, 0,
				format.FormatBytes(p.Virt),
				format.FormatBytes(p.Res),
				format.FormatBytes(p.Shr),
				truncateStr(p.Status, 1),
				p.CPUPercent,
				p.MemPercent,
				"0:00.00",
				truncateStr(p.Name, 20)))
		}
	}

	// Notable Processes
	if info.ProcessExt != nil && len(info.ProcessExt.NotableProcesses) > 0 {
		sb.WriteString("# Notable Processes " + strings.Repeat("#", 41) + "\n")
		sb.WriteString("  PID    OOM    COMMAND\n")
		for _, p := range info.ProcessExt.NotableProcesses {
			sb.WriteString(fmt.Sprintf("  %d    %d    %s\n", p.PID, p.OOMAdj, p.Name))
		}
	}

	// Memory management - only show on Linux
	if runtime.GOOS == "linux" && info.KernelExt != nil {
		sb.WriteString("# Memory management " + strings.Repeat("#", 41) + "\n")
		if info.KernelExt.THPEnabled {
			sb.WriteString("Transparent huge pages are enabled.\n")
		} else {
			sb.WriteString("Transparent huge pages are disabled.\n")
		}
	}

	// The End
	sb.WriteString("# The End " + strings.Repeat("#", 50) + "\n")

	return sb.String()
}

func firstNonEmpty(s ...string) string {
	for _, str := range s {
		if str != "" {
			return str
		}
	}
	return ""
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
