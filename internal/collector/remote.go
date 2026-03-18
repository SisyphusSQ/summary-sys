package collector

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/SisyphusSQ/summary-sys/internal/ssh"

	l "github.com/SisyphusSQ/summary-sys/pkg/log"
)

type RemoteCollector struct {
	client *ssh.Client
	opts   *options
}

func NewRemoteCollector(sshClient *ssh.Client, opts ...Option) (*RemoteCollector, error) {
	o := &options{
		timeout: 30 * time.Second,
		collect: &CollectConfig{
			CPU:     true,
			Memory:  true,
			Disk:    true,
			Network: true,
			Process: true,
			LoadAvg: true,
			Who:     true,

			// 扩展字段默认启用
			SystemExt:   true,
			CPUDetail:   true,
			MemoryDetail: true,
			DiskExt:     true,
			NetworkExt:  true,
			KernelExt:   true,
			ProcessExt:  true,
		},
	}
	for _, opt := range opts {
		opt(o)
	}

	return &RemoteCollector{
		client: sshClient,
		opts:   o,
	}, nil
}

func (c *RemoteCollector) Name() string {
	return "remote:" + c.client.Host()
}

func (c *RemoteCollector) Collect(ctx context.Context) (*SystemInfo, error) {
	l.Logger.Infof("start collecting remote system info from %s", c.client.Host())
	start := time.Now()
	defer func() {
		l.Logger.Infof("remote collection completed in %v", time.Since(start))
	}()

	info := &SystemInfo{
		Timestamp: time.Now(),
	}

	if err := c.collectViaCommands(ctx, info); err != nil {
		l.Logger.Warnf("remote collection error: %v", err)
	}

	return info, nil
}

func (c *RemoteCollector) collectViaCommands(ctx context.Context, info *SystemInfo) error {
	cmd := `
HOSTNAME=$(hostname)
UNAME=$(uname -a)
UPTIME=$(cat /proc/uptime 2>/dev/null | awk '{print $1}')
NPROC=$(nproc 2>/dev/null)
MEM=$(free -b 2>/dev/null | awk '/Mem:/ {print $2,$3,$7}')
MEMINFO=$(cat /proc/meminfo 2>/dev/null)
DF=$(df -B1 -k --output=source,fstype,size,used,avail,pcent 2>/dev/null | tail -n +2)
LOAD=$(cat /proc/loadavg 2>/dev/null)
NETSTAT=$(ss -tan 2>/dev/null | wc -l)
CPUINFO=$(cat /proc/cpuinfo 2>/dev/null | grep -E 'model name|cpu MHz|cache size' | head -20)
LSCPU=$(lscpu 2>/dev/null)
SWAPPINESS=$(cat /proc/sys/vm/swappiness 2>/dev/null)
KERNEL=$(uname -r)
SELINUX=$(getenforce 2>/dev/null || echo "Disabled")
VIRT=$(systemd-detect-virt 2>/dev/null || echo "No virtualization detected")
DENTRY=$(cat /proc/sys/fs/dentry-state 2>/dev/null)
FILENR=$(cat /proc/sys/fs/file-nr 2>/dev/null)
INODENR=$(cat /proc/sys/fs/inode-nr 2>/dev/null)
THP=$(cat /sys/kernel/mm/transparent_hugepage/enabled 2>/dev/null)
NETCONN=$(ss -tan state established 2>/dev/null | awk '{print $4,$5}' | tail -n +2)

# Network interface statistics
NET_IFACE=$(cat /proc/net/dev 2>/dev/null | grep -v "Inter-" | grep -v "face" | awk '{print $1,$2,$3,$4,$10,$11,$12,$13}')

# Network connection states
NET_STATE=$(ss -tan 2>/dev/null | awk '{print $1}' | sort | uniq -c | sort -rn)

echo "HOSTNAME:$HOSTNAME"
echo "UNAME:$UNAME"
echo "UPTIME:$UPTIME"
echo "NPROC:$NPROC"
echo "MEM:$MEM"
echo "MEMINFO:$MEMINFO"
echo "DF:$DF"
echo "LOAD:$LOAD"
echo "NETSTAT:$NETSTAT"
echo "CPUINFO:$CPUINFO"
echo "LSCPU:$LSCPU"
echo "SWAPPINESS:$SWAPPINESS"
echo "KERNEL:$KERNEL"
echo "SELINUX:$SELINUX"
echo "VIRT:$VIRT"
echo "DENTRY:$DENTRY"
echo "FILENR:$FILENR"
echo "INODENR:$INODENR"
echo "THP:$THP"
echo "NETCONN:$NETCONN"
echo "NET_IFACE:$NET_IFACE"
echo "NET_STATE:$NET_STATE"
`
	stdout, _, err := c.client.Run(ctx, cmd)
	if err != nil {
		return fmt.Errorf("remote command failed: %w", err)
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "HOSTNAME:") {
			info.Hostname = strings.TrimPrefix(line, "HOSTNAME:")
		} else if strings.HasPrefix(line, "UNAME:") {
			info.OS = strings.TrimPrefix(line, "UNAME:")
			parts := strings.Fields(strings.TrimPrefix(line, "UNAME:"))
			if len(parts) >= 3 {
				info.Kernel = parts[2]
			}
		} else if strings.HasPrefix(line, "UPTIME:") {
			if uptime, ok := parseFloat(strings.TrimPrefix(line, "UPTIME:")); ok {
				info.Uptime = time.Duration(uptime * float64(time.Second))
			}
		} else if strings.HasPrefix(line, "NPROC:") {
			if n, ok := parseInt(strings.TrimPrefix(line, "NPROC:")); ok {
				info.CPU = &CPUInfo{
					PhysicalCores: n,
					LogicalCores:  n,
				}
			}
		} else if strings.HasPrefix(line, "MEM:") {
			memData := strings.TrimPrefix(line, "MEM:")
			var total, used, available uint64
			fmt.Sscanf(memData, "%d %d %d", &total, &used, &available)
			if total > 0 {
				info.Memory = &MemoryInfo{
					Total:       total,
					Used:        used,
					Available:   available,
					UsedPercent: float64(used) / float64(total) * 100,
				}
			}
		} else if strings.HasPrefix(line, "MEMINFO:") {
			if c.opts.collect.MemoryDetail {
				c.parseMeminfo(strings.TrimPrefix(line, "MEMINFO:"), info)
			}
		} else if strings.HasPrefix(line, "DF:") {
			dfData := strings.TrimPrefix(line, "DF:")
			disks := parseDiskInfo(dfData)
			if len(disks) > 0 {
				info.Disk = &disks
			}
		} else if strings.HasPrefix(line, "LOAD:") {
			loadData := strings.TrimPrefix(line, "LOAD:")
			var l1, l5, l15 float64
			fmt.Sscanf(loadData, "%f %f %f", &l1, &l5, &l15)
			info.LoadAvg = &LoadAvgInfo{
				Load1:  l1,
				Load5:  l5,
				Load15: l15,
			}
		} else if strings.HasPrefix(line, "NETSTAT:") {
			if n, ok := parseInt(strings.TrimPrefix(line, "NETSTAT:")); ok {
				info.Network = &NetworkInfo{
					NetstatTCP: n - 1,
				}
			}
		} else if strings.HasPrefix(line, "CPUINFO:") && c.opts.collect.CPUDetail {
			cpuInfoData := strings.TrimPrefix(line, "CPUINFO:")
			c.parseCPUInfo(cpuInfoData, info)
		} else if strings.HasPrefix(line, "LSCPU:") && c.opts.collect.CPUDetail {
			lscpuData := strings.TrimPrefix(line, "LSCPU:")
			c.parseLSCPU(lscpuData, info)
		} else if strings.HasPrefix(line, "SWAPPINESS:") && c.opts.collect.MemoryDetail {
			swappiness, _ := parseInt(strings.TrimPrefix(line, "SWAPPINESS:"))
			if info.MemoryExt == nil {
				info.MemoryExt = &MemoryExtInfo{}
			}
			info.MemoryExt.Swappiness = swappiness
		} else if strings.HasPrefix(line, "KERNEL:") {
			if info.SystemExt == nil {
				info.SystemExt = &SystemExtInfo{}
			}
			info.SystemExt.Threading = "NPTL"
		} else if strings.HasPrefix(line, "SELINUX:") && c.opts.collect.SystemExt {
			selinux := strings.TrimPrefix(line, "SELINUX:")
			if info.SystemExt == nil {
				info.SystemExt = &SystemExtInfo{}
			}
			info.SystemExt.SELinux = selinux
		} else if strings.HasPrefix(line, "VIRT:") && c.opts.collect.SystemExt {
			virt := strings.TrimPrefix(line, "VIRT:")
			if info.SystemExt == nil {
				info.SystemExt = &SystemExtInfo{}
			}
			info.SystemExt.Virtualized = virt
		} else if strings.HasPrefix(line, "DENTRY:") && c.opts.collect.KernelExt {
			dentry := strings.TrimPrefix(line, "DENTRY:")
			if info.KernelExt == nil {
				info.KernelExt = &KernelExtInfo{}
			}
			info.KernelExt.DentryState = dentry
		} else if strings.HasPrefix(line, "FILENR:") && c.opts.collect.KernelExt {
			filenr := strings.TrimPrefix(line, "FILENR:")
			if info.KernelExt == nil {
				info.KernelExt = &KernelExtInfo{}
			}
			info.KernelExt.FileNr = filenr
		} else if strings.HasPrefix(line, "INODENR:") && c.opts.collect.KernelExt {
			inodenr := strings.TrimPrefix(line, "INODENR:")
			if info.KernelExt == nil {
				info.KernelExt = &KernelExtInfo{}
			}
			info.KernelExt.InodeNr = inodenr
		} else if strings.HasPrefix(line, "THP:") && c.opts.collect.KernelExt {
			thp := strings.TrimPrefix(line, "THP:")
			if info.KernelExt == nil {
				info.KernelExt = &KernelExtInfo{}
			}
			info.KernelExt.THPEnabled = strings.Contains(thp, "[always]")
		} else if strings.HasPrefix(line, "NETCONN:") && c.opts.collect.NetworkExt {
			netconnData := strings.TrimPrefix(line, "NETCONN:")
			c.parseNetConn(netconnData, info)
		} else if strings.HasPrefix(line, "NET_IFACE:") {
			netIfaceData := strings.TrimPrefix(line, "NET_IFACE:")
			c.parseNetIface(netIfaceData, info)
		} else if strings.HasPrefix(line, "NET_STATE:") && c.opts.collect.NetworkExt {
			netStateData := strings.TrimPrefix(line, "NET_STATE:")
			c.parseNetState(netStateData, info)
		}
	}

	return nil
}

func parseFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err == nil
}

func parseInt(s string) (int, bool) {
	s = strings.TrimSpace(s)
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err == nil
}

func parseDiskInfo(output string) DiskInfo {
	var disks DiskInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var dev, fstype string
		var total, used, available uint64
		var usedPercent float64
		parts := strings.Fields(line)
		if len(parts) >= 6 {
			dev = parts[0]
			fstype = parts[1]
			fmt.Sscanf(parts[2], "%d", &total)
			fmt.Sscanf(parts[3], "%d", &used)
			fmt.Sscanf(parts[4], "%d", &available)
			pctStr := strings.TrimSuffix(parts[5], "%")
			pct, _ := strconv.ParseFloat(pctStr, 64)
			usedPercent = pct
			disks = append(disks, DiskPartition{
				Device:      dev,
				FSType:      fstype,
				Total:       total,
				Used:        used,
				Available:   available,
				UsedPercent: usedPercent,
			})
		}
	}
	return disks
}

func (c *RemoteCollector) parseMeminfo(meminfoData string, info *SystemInfo) {
	ext := &MemoryExtInfo{
		NumaNodes: make([]NumaNode, 0),
		Dimms:     make([]DimmInfo, 0),
	}

	lines := strings.Split(meminfoData, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		val := parts[1]

		switch key {
		case "MemTotal":
			if v, err := strconv.ParseUint(val, 10, 64); err == nil {
				if info.Memory == nil {
					info.Memory = &MemoryInfo{}
				}
				info.Memory.Total = v * 1024
			}
		case "MemFree":
			if v, err := strconv.ParseUint(val, 10, 64); err == nil {
				ext.Free = v * 1024
			}
		case "MemAvailable":
			if v, err := strconv.ParseUint(val, 10, 64); err == nil {
				if info.Memory == nil {
					info.Memory = &MemoryInfo{}
				}
				info.Memory.Available = v * 1024
			}
		case "Buffers":
			if v, err := strconv.ParseUint(val, 10, 64); err == nil {
				ext.Buffers = v * 1024
			}
		case "Cached":
			if v, err := strconv.ParseUint(val, 10, 64); err == nil {
				ext.Caches = v * 1024
			}
		case "SwapCached":
			// ignore
		case "Dirty":
			if v, err := strconv.ParseUint(val, 10, 64); err == nil {
				ext.Dirty = v * 1024
			}
		case "Shmem":
			if v, err := strconv.ParseUint(val, 10, 64); err == nil {
				ext.Shared = v * 1024
			}
		}
	}

	info.MemoryExt = ext
}

func (c *RemoteCollector) parseCPUInfo(cpuInfoData string, info *SystemInfo) {
	ext := &CPUExtInfo{
		Speeds:  make([]float64, 0),
		Caches:  make([]CacheInfo, 0),
		Hyperthreading: false,
	}

	seenSpeeds := make(map[float64]bool)
	lines := strings.Split(cpuInfoData, "\n")
	for _, line := range lines {
		if strings.Contains(line, "cpu MHz") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				if freq, err := strconv.ParseFloat(parts[3], 64); err == nil && !seenSpeeds[freq] {
					ext.Speeds = append(ext.Speeds, freq)
					seenSpeeds[freq] = true
				}
			}
		} else if strings.Contains(line, "cache size") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				if cache, err := strconv.ParseUint(parts[3], 10, 64); err == nil {
					ext.Caches = append(ext.Caches, CacheInfo{
						Level:         "L3",
						Size:          cache * 1024,
						Associativity: "Unknown",
					})
				}
			}
		}
	}

	info.CPUExt = ext
}

func (c *RemoteCollector) parseLSCPU(lscpuData string, info *SystemInfo) {
	if info.CPUExt == nil {
		info.CPUExt = &CPUExtInfo{
			Speeds:         make([]float64, 0),
			Caches:         make([]CacheInfo, 0),
			Hyperthreading: false,
		}
	}
	if info.CPU == nil {
		info.CPU = &CPUInfo{
			Models: make([]string, 0),
		}
	}

	lines := strings.Split(lscpuData, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Thread(s) per core:") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				if threads, err := strconv.Atoi(parts[3]); err == nil && threads > 1 {
					info.CPUExt.Hyperthreading = true
				}
			}
		} else if strings.HasPrefix(line, "Model name:") {
			model := strings.TrimPrefix(line, "Model name:")
			model = strings.TrimSpace(model)
			info.CPU.Models = []string{model}
		} else if strings.HasPrefix(line, "CPU(s):") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if logical, err := strconv.Atoi(parts[1]); err == nil {
					info.CPU.LogicalCores = logical
				}
			}
		} else if strings.HasPrefix(line, "Core(s) per socket:") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				if cores, err := strconv.Atoi(parts[3]); err == nil {
					info.CPU.CoresPerSocket = cores
				}
			}
		} else if strings.HasPrefix(line, "Socket(s):") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if sockets, err := strconv.Atoi(parts[1]); err == nil {
					info.CPU.Sockets = sockets
					info.CPU.PhysicalCores = sockets * info.CPU.CoresPerSocket
				}
			}
		} else if strings.HasPrefix(line, "CPU MHz:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if freq, err := strconv.ParseFloat(parts[2], 64); err == nil {
					// Add to speeds if not already present
					seen := false
					for _, s := range info.CPUExt.Speeds {
						if s == freq {
							seen = true
							break
						}
					}
					if !seen {
						info.CPUExt.Speeds = append(info.CPUExt.Speeds, freq)
					}
				}
			}
		}
	}
}

func (c *RemoteCollector) parseNetConn(netconnData string, info *SystemInfo) {
	ext := &NetworkExtInfo{
		ConnFromRemote: make(map[string]int),
		ConnToLocal:   make(map[string]int),
		ConnToPorts:  make([]PortStat, 0),
		ConnStates:    make(map[string]int),
	}

	portCounts := make(map[int]int)
	lines := strings.Split(netconnData, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		localAddr := parts[0]
		remoteAddr := parts[1]

		if !strings.Contains(remoteAddr, ":") {
			continue
		}

		remoteIP := remoteAddr[:strings.LastIndex(remoteAddr, ":")]
		if remoteIP != "127.0.0.1" && remoteIP != "::1" && !strings.HasPrefix(remoteIP, "0.0.0.0") {
			ext.ConnFromRemote[remoteIP]++
		}

		if strings.Contains(localAddr, ":") {
			localPort := localAddr[strings.LastIndex(localAddr, ":")+1:]
			if port, err := strconv.Atoi(localPort); err == nil {
				portCounts[port]++
			}
		}

		ext.ConnStates["ESTABLISHED"]++
	}

	for port, count := range portCounts {
		ext.ConnToPorts = append(ext.ConnToPorts, PortStat{Port: port, Count: count})
	}

	sort.Slice(ext.ConnToPorts, func(i, j int) bool {
		return ext.ConnToPorts[i].Count > ext.ConnToPorts[j].Count
	})
	if len(ext.ConnToPorts) > 10 {
		ext.ConnToPorts = ext.ConnToPorts[:10]
	}

	info.NetworkExt = ext
}

// parseNetIface parses network interface statistics from /proc/net/dev
func (c *RemoteCollector) parseNetIface(netIfaceData string, info *SystemInfo) {
	lines := strings.Split(netIfaceData, "\n")

	if info.Network == nil {
		info.Network = &NetworkInfo{
			Interfaces: make([]NetInterface, 0),
			ConnStates: make(map[string]int),
		}
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 11 {
			continue
		}

		name := strings.TrimSuffix(parts[0], ":")
		// Skip loopback
		if name == "lo" {
			continue
		}

		rxBytes, _ := strconv.ParseUint(parts[1], 10, 64)
		rxPackets, _ := strconv.ParseUint(parts[2], 10, 64)
		rxErrors, _ := strconv.ParseUint(parts[3], 10, 64)
		txBytes, _ := strconv.ParseUint(parts[8], 10, 64)
		txPackets, _ := strconv.ParseUint(parts[9], 10, 64)
		txErrors, _ := strconv.ParseUint(parts[10], 10, 64)

		iface := NetInterface{
			Name: name,
			Statistics: &NetStats{
				BytesRecv:   rxBytes,
				PacketsRecv: rxPackets,
				ErrIn:       rxErrors,
				BytesSent:   txBytes,
				PacketsSent: txPackets,
				ErrOut:      txErrors,
			},
		}
		info.Network.Interfaces = append(info.Network.Interfaces, iface)
	}
}

// parseNetState parses network connection state statistics
func (c *RemoteCollector) parseNetState(netStateData string, info *SystemInfo) {
	if info.NetworkExt == nil {
		info.NetworkExt = &NetworkExtInfo{
			ConnStates: make(map[string]int),
		}
	}

	lines := strings.Split(netStateData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		count, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		state := parts[1]
		info.NetworkExt.ConnStates[state] = count
	}
}
