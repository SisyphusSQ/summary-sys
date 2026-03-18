package collector

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	l "github.com/SisyphusSQ/summary-sys/pkg/log"
)

type LocalCollector struct {
	opts *options
}

func NewLocalCollector(opts ...Option) *LocalCollector {
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
			SystemExt:    true,
			CPUDetail:    true,
			MemoryDetail: true,
			DiskExt:      true,
			NetworkExt:   true,
			KernelExt:    true,
			ProcessExt:   true,
		},
	}
	for _, opt := range opts {
		opt(o)
	}
	return &LocalCollector{opts: o}
}

func (c *LocalCollector) Name() string {
	return "local"
}

func (c *LocalCollector) Collect(ctx context.Context) (*SystemInfo, error) {
	l.Logger.Infof("start collecting local system info")
	start := time.Now()
	defer func() {
		l.Logger.Infof("local collection completed in %v", time.Since(start))
	}()

	info := &SystemInfo{
		Timestamp: time.Now(),
	}

	var errs []error

	if err := c.collectSystemInfo(ctx, info); err != nil {
		errs = append(errs, err)
	}

	if c.opts.collect.CPU {
		if err := c.collectCPU(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if c.opts.collect.Memory {
		if err := c.collectMemory(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if c.opts.collect.Disk {
		if err := c.collectDisk(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if c.opts.collect.Network {
		if err := c.collectNetwork(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if c.opts.collect.Process {
		if err := c.collectProcess(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if c.opts.collect.LoadAvg {
		if err := c.collectLoadAvg(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if c.opts.collect.Who {
		if err := c.collectWho(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	// 扩展字段采集
	if c.opts.collect.SystemExt {
		if err := c.collectSystemExt(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if c.opts.collect.CPUDetail {
		if err := c.collectCPUDetail(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if c.opts.collect.MemoryDetail {
		if err := c.collectMemoryDetail(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if c.opts.collect.DiskExt {
		if err := c.collectDiskExt(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if c.opts.collect.KernelExt {
		if err := c.collectKernelExt(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if c.opts.collect.NetworkExt {
		if err := c.collectNetworkExt(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if c.opts.collect.ProcessExt {
		if err := c.collectProcessExt(ctx, info); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		l.Logger.Warnf("collection completed with %d errors", len(errs))
	}
	return info, nil
}

func (c *LocalCollector) collectSystemInfo(ctx context.Context, info *SystemInfo) error {
	h, err := host.InfoWithContext(ctx)
	if err != nil {
		return err
	}
	info.Hostname = h.Hostname
	info.Platform = h.Platform // e.g., "Linux"
	info.Kernel = h.KernelVersion

	// Get OS/Release properly
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		content := string(data)
		if idx := strings.Index(content, "PRETTY_NAME="); idx >= 0 {
			line := content[idx:]
			if end := strings.Index(line, "\n"); end >= 0 {
				info.OS = strings.TrimSpace(strings.TrimPrefix(line[:end], "PRETTY_NAME="))
				info.OS = strings.Trim(info.OS, `"`)
			}
		}
	}
	if info.OS == "" {
		info.OS = h.OS
	}

	// Architecture - use 64-bit format
	info.Architecture = "64-bit"
	if runtime.GOARCH == "386" {
		info.Architecture = "32-bit"
	}

	info.Uptime = time.Duration(h.Uptime) * time.Second
	return nil
}

func (c *LocalCollector) collectCPU(ctx context.Context, info *SystemInfo) error {
	cpuInfo, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return err
	}

	physical, _ := cpu.CountsWithContext(ctx, false)
	logical, _ := cpu.CountsWithContext(ctx, true)

	// Get actual physical sockets from lscpu or cpuinfo
	sockets := physical
	coresPerSocket := physical
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			// Count unique physical id to get socket count
			physicalIDs := make(map[string]bool)
			currentPhysicalID := ""
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "physical id") {
					parts := strings.Fields(line)
					if len(parts) >= 4 {
						currentPhysicalID = parts[3]
						physicalIDs[currentPhysicalID] = true
					}
				}
			}
			if len(physicalIDs) > 0 {
				sockets = len(physicalIDs)
				if sockets > 0 {
					coresPerSocket = logical / sockets
				}
			}
		}
	}

	percent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return err
	}

	models := make([]string, 0, len(cpuInfo))
	for _, ci := range cpuInfo {
		models = append(models, ci.ModelName)
	}

	info.CPU = &CPUInfo{
		PhysicalCores:  physical,
		LogicalCores:   logical,
		Models:         models,
		UsagePercent:   percent[0],
		Sockets:        sockets,
		CoresPerSocket: coresPerSocket,
	}
	return nil
}

func (c *LocalCollector) collectMemory(ctx context.Context, info *SystemInfo) error {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return err
	}
	s, _ := mem.SwapMemoryWithContext(ctx)

	info.Memory = &MemoryInfo{
		Total:       v.Total,
		Available:   v.Available,
		Used:        v.Used,
		UsedPercent: v.UsedPercent,
		SwapTotal:   s.Total,
		SwapUsed:    s.Used,
		SwapPercent: s.UsedPercent,
	}
	return nil
}

func (c *LocalCollector) collectDisk(ctx context.Context, info *SystemInfo) error {
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return err
	}

	disks := make(DiskInfo, 0, len(partitions))
	for _, p := range partitions {
		usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil {
			continue
		}
		disks = append(disks, DiskPartition{
			Device:      p.Device,
			MountPoint:  p.Mountpoint,
			FSType:      p.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Available:   usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}
	info.Disk = &disks
	return nil
}

func (c *LocalCollector) collectNetwork(ctx context.Context, info *SystemInfo) error {
	interfaces, err := net.InterfacesWithContext(ctx)
	if err != nil {
		return err
	}

	stats, err := net.IOCounters(false)
	if err != nil {
		return err
	}

	statMap := make(map[string]net.IOCountersStat)
	for _, s := range stats {
		statMap[s.Name] = s
	}

	netInfo := &NetworkInfo{
		Interfaces: make([]NetInterface, 0, len(interfaces)),
		ConnStates: make(map[string]int),
	}

	for _, iface := range interfaces {
		ni := NetInterface{
			Name:  iface.Name,
			MTU:   iface.MTU,
			Addrs: make([]string, 0, len(iface.Addrs)),
		}
		for _, addr := range iface.Addrs {
			ni.Addrs = append(ni.Addrs, addr.Addr)
		}

		if s, ok := statMap[iface.Name]; ok {
			ni.Statistics = &NetStats{
				BytesSent:   s.BytesSent,
				BytesRecv:   s.BytesRecv,
				PacketsSent: s.PacketsSent,
				PacketsRecv: s.PacketsRecv,
				ErrIn:       s.Errin,
				ErrOut:      s.Errout,
				DropIn:      s.Dropin,
				DropOut:     s.Dropout,
			}
		}
		netInfo.Interfaces = append(netInfo.Interfaces, ni)
	}

	conns, err := net.ConnectionsWithContext(ctx, "tcp")
	if err == nil {
		netInfo.NetstatTCP = len(conns)
	}
	connsUDP, err := net.ConnectionsWithContext(ctx, "udp")
	if err == nil {
		netInfo.NetstatUDP = len(connsUDP)
	}

	info.Network = netInfo
	return nil
}

func (c *LocalCollector) collectProcess(ctx context.Context, info *SystemInfo) error {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return err
	}

	procInfo := &ProcessInfo{
		Count:     len(procs),
		TopCPU:    make([]*ProcessStat, 0, 10),
		TopMemory: make([]*ProcessStat, 0, 10),
	}

	allProcs := make([]*ProcessStat, 0, len(procs))
	for _, p := range procs {
		name, _ := p.Name()
		username, _ := p.Username()
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		status, _ := p.Status()
		cmd, _ := p.Cmdline()

		// Get memory info
		memInfo, _ := p.MemoryInfo()
		virt := memInfo.VMS
		res := memInfo.RSS

		ps := &ProcessStat{
			PID:        p.Pid,
			Name:       name,
			User:       username,
			CPUPercent: cpu,
			MemPercent: float64(mem),
			Status:     status[0],
			Command:    cmd,
			Virt:       virt,
			Res:        res,
			Shr:        0, // Shared memory not available in gopsutil
		}
		allProcs = append(allProcs, ps)
	}

	sort.Slice(allProcs, func(i, j int) bool {
		return allProcs[i].CPUPercent > allProcs[j].CPUPercent
	})
	if len(allProcs) > 10 {
		procInfo.TopCPU = allProcs[:10]
	} else {
		procInfo.TopCPU = allProcs
	}

	sort.Slice(allProcs, func(i, j int) bool {
		return allProcs[i].MemPercent > allProcs[j].MemPercent
	})
	if len(allProcs) > 10 {
		procInfo.TopMemory = allProcs[:10]
	} else {
		procInfo.TopMemory = allProcs
	}

	info.Process = procInfo
	return nil
}

func (c *LocalCollector) collectLoadAvg(ctx context.Context, info *SystemInfo) error {
	avg, err := load.AvgWithContext(ctx)
	if err != nil {
		return err
	}
	misc, err := load.MiscWithContext(ctx)
	if err != nil {
		return err
	}

	info.LoadAvg = &LoadAvgInfo{
		Load1:   avg.Load1,
		Load5:   avg.Load5,
		Load15:  avg.Load15,
		Runable: misc.ProcsRunning,
		Total:   misc.ProcsTotal,
	}
	return nil
}

func (c *LocalCollector) collectWho(ctx context.Context, info *SystemInfo) error {
	// Use 'who' command to get logged in users
	cmd := exec.Command("who")
	output, err := cmd.Output()
	if err != nil {
		l.Logger.Debugf("failed to run who command: %v", err)
		return nil
	}

	lines := strings.Split(string(output), "\n")
	users := make([]*WhoEntry, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			entry := &WhoEntry{
				Terminal: parts[0],
				User:     parts[1],
				Host:     parts[2],
			}
			// Try to parse time if available
			if len(parts) >= 4 {
				timeStr := strings.Join(parts[3:], " ")
				if t, err := time.Parse("2006-01-02 15:04", timeStr); err == nil {
					entry.LoginTime = t
				}
			}
			users = append(users, entry)
		}
	}

	info.Who = users
	return nil
}

func (c *LocalCollector) collectSystemExt(ctx context.Context, info *SystemInfo) error {
	h, err := host.InfoWithContext(ctx)
	if err != nil {
		l.Logger.Debugf("failed to get host info: %v", err)
		return nil
	}

	ext := &SystemExtInfo{}

	// Get threading (NPTL version) from runtime
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/version"); err == nil {
			if strings.Contains(string(data), "glibc") {
				re := regexp.MustCompile(`glibc.*(\d+\.\d+)`)
				matches := re.FindStringSubmatch(string(data))
				if len(matches) > 1 {
					ext.Threading = "NPTL " + matches[1]
				}
			}
		}
		if ext.Threading == "" {
			ext.Threading = "NPTL"
		}
	} else {
		ext.Threading = runtime.Version()
	}

	if h.VirtualizationSystem != "" {
		ext.Virtualized = h.VirtualizationSystem
	} else {
		ext.Virtualized = "No virtualization detected"
	}

	selinuxStatus, err := c.getSELinuxStatus(ctx)
	if err == nil {
		ext.SELinux = selinuxStatus
	}

	// Get hardware info from dmidecode (Linux only)
	if runtime.GOOS == "linux" {
		c.collectDMIInfo(ext)
	}

	// Platform family as model
	if h.PlatformFamily != "" {
		ext.Model = h.PlatformFamily
	}

	if h.PlatformVersion != "" {
		ext.Version = h.PlatformVersion
	}

	info.SystemExt = ext
	return nil
}

// collectDMIInfo collects hardware information from dmidecode
func (c *LocalCollector) collectDMIInfo(ext *SystemExtInfo) {
	// Use /sys/class/dmi/id/ directly (no command needed)
	if data, err := os.ReadFile("/sys/class/dmi/id/sys_vendor"); err == nil {
		ext.Vendor = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
		ext.Model = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile("/sys/class/dmi/id/product_serial"); err == nil {
		ext.ServiceTag = strings.TrimSpace(string(data))
	}
}

func (c *LocalCollector) getSELinuxStatus(ctx context.Context) (string, error) {
	selinuxPath := "/sys/fs/selinux/enforce"
	data, err := os.ReadFile(selinuxPath)
	if err != nil {
		selinuxPath = "/etc/selinux/config"
		data, err = os.ReadFile(selinuxPath)
		if err != nil {
			return "Disabled", nil
		}
		content := string(data)
		if strings.Contains(content, "SELINUX=disabled") {
			return "Disabled", nil
		}
		if strings.Contains(content, "SELINUX=enforcing") {
			return "Enforcing", nil
		}
		if strings.Contains(content, "SELINUX=permissive") {
			return "Permissive", nil
		}
		return "Unknown", nil
	}
	if strings.TrimSpace(string(data)) == "1" {
		return "Enforcing", nil
	}
	return "Permissive", nil
}

func (c *LocalCollector) collectCPUDetail(ctx context.Context, info *SystemInfo) error {
	cpuInfo, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return err
	}

	ext := &CPUExtInfo{
		Speeds:         make([]float64, 0),
		Caches:         make([]CacheInfo, 0),
		Hyperthreading: false,
	}

	seenSpeeds := make(map[float64]bool)
	seenCacheLevels := make(map[string]bool)

	for _, ci := range cpuInfo {
		if ci.Mhz > 0 && !seenSpeeds[ci.Mhz] {
			ext.Speeds = append(ext.Speeds, ci.Mhz)
			seenSpeeds[ci.Mhz] = true
		}

		// Only add unique cache configurations (L1, L2, L3 once per socket)
		if ci.CacheSize > 0 && !seenCacheLevels["L3"] {
			ext.Caches = append(ext.Caches, CacheInfo{
				Level:         "L3",
				Size:          uint64(ci.CacheSize) * 1024,
				Associativity: "Fully Associative",
			})
			seenCacheLevels["L3"] = true
		}
		// Try to get L1 and L2 from /sys/devices/system/cpu/cpu0/cache if available
	}

	// Try to get more detailed cache info from sysfs
	if runtime.GOOS == "linux" {
		c.collectCacheFromSysfs(ext)
	}

	logical, _ := cpu.CountsWithContext(ctx, true)
	physical, _ := cpu.CountsWithContext(ctx, false)
	if logical > physical && physical > 0 {
		ext.Hyperthreading = true
	}

	info.CPUExt = ext
	return nil
}

// collectCacheFromSysfs collects L1/L2/L3 cache info from sysfs
func (c *LocalCollector) collectCacheFromSysfs(ext *CPUExtInfo) {
	seenLevels := make(map[string]bool)
	cachePath := "/sys/devices/system/cpu/cpu0/cache"

	for level := 1; level <= 3; level++ {
		indexPath := cachePath + "/index" + strconv.Itoa(level)
		if _, err := os.Stat(indexPath); err != nil {
			continue
		}

		levelStr := ""
		switch level {
		case 1:
			levelStr = "L1"
		case 2:
			levelStr = "L2"
		case 3:
			levelStr = "L3"
		}

		if levelStr != "" && !seenLevels[levelStr] {
			sizePath := indexPath + "/size"
			if data, err := os.ReadFile(sizePath); err == nil {
				sizeStr := strings.TrimSpace(string(data))
				// Parse size like "640K"
				var size uint64
				if strings.HasSuffix(sizeStr, "K") {
					if v, err := strconv.ParseUint(strings.TrimSuffix(sizeStr, "K"), 10, 64); err == nil {
						size = v * 1024
					}
				} else if strings.HasSuffix(sizeStr, "M") {
					if v, err := strconv.ParseUint(strings.TrimSuffix(sizeStr, "M"), 10, 64); err == nil {
						size = v * 1024 * 1024
					}
				}

				// Get associativity
				assocPath := indexPath + "/associativity"
				assoc := "Unknown"
				if data, err := os.ReadFile(assocPath); err == nil {
					assocStr := strings.TrimSpace(string(data))
					switch assocStr {
					case "8":
						assoc = "8-way Set-associative"
					case "16":
						assoc = "16-way Set-associative"
					case "fully":
						assoc = "Fully Associative"
					default:
						assoc = assocStr + "-way Set-associative"
					}
				}

				ext.Caches = append(ext.Caches, CacheInfo{
					Level:         levelStr,
					Size:          size,
					Associativity: assoc,
				})
				seenLevels[levelStr] = true
			}
		}
	}
}

func (c *LocalCollector) collectMemoryDetail(ctx context.Context, info *SystemInfo) error {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return err
	}

	ext := &MemoryExtInfo{
		Free:      v.Free,
		Shared:    v.Shared,
		Buffers:   v.Buffers,
		Caches:    v.Cached,
		Dirty:     v.Dirty,
		UsedRSS:   v.Used,
		NumaNodes: make([]NumaNode, 0),
		Dimms:     make([]DimmInfo, 0),
	}

	// Only collect Linux-specific memory info on Linux
	if runtime.GOOS == "linux" {
		swappinessPath := "/proc/sys/vm/swappiness"
		data, err := os.ReadFile(swappinessPath)
		if err == nil {
			if val, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				ext.Swappiness = val
			}
		}

		// Collect NUMA info
		c.collectNumaInfo(ext)

		// Collect DIMM info
		c.collectDimmInfo(ext)
	}

	info.MemoryExt = ext
	return nil
}

// collectNumaInfo collects NUMA node information from sysfs
func (c *LocalCollector) collectNumaInfo(ext *MemoryExtInfo) {
	sysNodesPath := "/sys/devices/system/node"
	if entries, err := os.ReadDir(sysNodesPath); err == nil {
		for _, entry := range entries {
			if !strings.HasPrefix(entry.Name(), "node") {
				continue
			}
			nodeID := strings.TrimPrefix(entry.Name(), "node")
			nodeIDInt, err := strconv.Atoi(nodeID)
			if err != nil {
				continue
			}

			node := NumaNode{
				ID:   nodeIDInt,
				CPUs: make([]int, 0),
			}

			// Get memory info
			meminfoPath := sysNodesPath + "/" + entry.Name() + "/meminfo"
			if data, err := os.ReadFile(meminfoPath); err == nil {
				content := string(data)
				// Parse Node 0, MemTotal: XXX kB
				for _, line := range strings.Split(content, "\n") {
					if strings.Contains(line, "MemTotal:") {
						parts := strings.Fields(line)
						if len(parts) >= 4 {
							if v, err := strconv.ParseUint(parts[3], 10, 64); err == nil {
								node.Size = v * 1024
							}
						}
					} else if strings.Contains(line, "MemFree:") || strings.Contains(line, "MemAvailable:") {
						parts := strings.Fields(line)
						if len(parts) >= 4 {
							if v, err := strconv.ParseUint(parts[3], 10, 64); err == nil {
								node.Free = v * 1024
							}
						}
					}
				}
			}

			// Get CPUs for this node
			cpulistPath := sysNodesPath + "/" + entry.Name() + "/cpulist"
			if data, err := os.ReadFile(cpulistPath); err == nil {
				// Parse cpulist like "0-9,20-29"
				cpus := parseCPUList(string(data))
				node.CPUs = cpus
			}

			ext.NumaNodes = append(ext.NumaNodes, node)
		}
	}
}

// parseCPUList parses CPU list like "0-9,20-29" into []int
func parseCPUList(list string) []int {
	var cpus []int
	list = strings.TrimSpace(list)
	for _, part := range strings.Split(list, ",") {
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				start, _ := strconv.Atoi(rangeParts[0])
				end, _ := strconv.Atoi(rangeParts[1])
				for i := start; i <= end; i++ {
					cpus = append(cpus, i)
				}
			}
		} else {
			if v, err := strconv.Atoi(part); err == nil {
				cpus = append(cpus, v)
			}
		}
	}
	return cpus
}

// collectDimmInfo collects memory DIMM information
func (c *LocalCollector) collectDimmInfo(ext *MemoryExtInfo) {
	// Try to read from /sys/class/dmi-id/ first
	dimms := make([]DimmInfo, 0)

	// Try to get DIMM info from dmidecode (needs root privileges)
	// This is a placeholder - in production would use exec.Command
	// For now, leave empty and let pt-summary handle it via remote

	ext.Dimms = dimms
}

func (c *LocalCollector) collectDiskExt(ctx context.Context, info *SystemInfo) error {
	ext := &DiskExtInfo{
		Schedulers: make(map[string]string),
		Partitions: make([]PartitionInfo, 0),
	}

	// Only collect on Linux
	if runtime.GOOS != "linux" {
		info.DiskExt = ext
		return nil
	}

	sysBlockPath := "/sys/block"
	entries, err := os.ReadDir(sysBlockPath)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "sd") || strings.HasPrefix(name, "nvme") || strings.HasPrefix(name, "vd") {
				schedulerPath := sysBlockPath + "/" + name + "/scheduler"
				scheduler := ""
				data, err := os.ReadFile(schedulerPath)
				if err == nil {
					content := string(data)
					if strings.Contains(content, "[mq-deadline]") {
						scheduler = "[mq-deadline]"
					} else if strings.Contains(content, "[none]") {
						scheduler = "[none]"
					} else if strings.Contains(content, "[bfq]") {
						scheduler = "[bfq]"
					} else if strings.Contains(content, "[kyber]") {
						scheduler = "[kyber]"
					} else {
						parts := strings.Fields(content)
						for _, p := range parts {
							if strings.HasPrefix(p, "[") && strings.HasSuffix(p, "]") {
								scheduler = p
								break
							}
						}
					}
				}

				queuePath := sysBlockPath + "/" + name + "/queue/nr_requests"
				data, err = os.ReadFile(queuePath)
				queueSize := ""
				if err == nil {
					if val, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
						queueSize = strconv.Itoa(val)
					}
				}

				// Format: [scheduler] queue_size
				if scheduler != "" && queueSize != "" {
					ext.Schedulers[name] = scheduler + " " + queueSize
				} else if scheduler != "" {
					ext.Schedulers[name] = scheduler
				} else if queueSize != "" {
					ext.Schedulers[name] = queueSize
				}
			}
		}
	}

	partitions, err := disk.PartitionsWithContext(context.Background(), false)
	if err == nil {
		for _, p := range partitions {
			ext.Partitions = append(ext.Partitions, PartitionInfo{
				Device: p.Device,
				Type:   "Part",
				Size:   0,
			})
		}
	}

	info.DiskExt = ext
	return nil
}

func (c *LocalCollector) collectKernelExt(ctx context.Context, info *SystemInfo) error {
	ext := &KernelExtInfo{}

	// Only collect on Linux
	if runtime.GOOS == "linux" {
		dentryPath := "/proc/sys/fs/dentry-state"
		data, err := os.ReadFile(dentryPath)
		if err == nil {
			ext.DentryState = strings.TrimSpace(string(data))
		}

		fileNrPath := "/proc/sys/fs/file-nr"
		data, err = os.ReadFile(fileNrPath)
		if err == nil {
			ext.FileNr = strings.TrimSpace(string(data))
		}

		inodeNrPath := "/proc/sys/fs/inode-nr"
		data, err = os.ReadFile(inodeNrPath)
		if err == nil {
			ext.InodeNr = strings.TrimSpace(string(data))
		}

		thpPath := "/sys/kernel/mm/transparent_hugepage/enabled"
		data, err = os.ReadFile(thpPath)
		if err == nil {
			content := string(data)
			if strings.Contains(content, "[always]") {
				ext.THPEnabled = true
			} else if strings.Contains(content, "[madvise]") {
				ext.THPEnabled = false
			} else {
				ext.THPEnabled = false
			}
		}
	}

	info.KernelExt = ext
	return nil
}

func (c *LocalCollector) collectNetworkExt(ctx context.Context, info *SystemInfo) error {
	conns, err := net.ConnectionsWithContext(ctx, "tcp")
	if err != nil {
		return err
	}

	ext := &NetworkExtInfo{
		ConnFromRemote: make(map[string]int),
		ConnToLocal:    make(map[string]int),
		ConnToPorts:    make([]PortStat, 0),
		ConnStates:     make(map[string]int),
		NetDevices:     make([]NetDevice, 0),
	}

	// Collect network device speed/duplex info (Linux only)
	// TODO: implement collectNetDeviceInfo
	// if runtime.GOOS == "linux" {
	// 	c.collectNetDeviceInfo(ext)
	// }

	portCounts := make(map[int]int)
	for _, conn := range conns {
		if conn.Status != "" {
			ext.ConnStates[conn.Status]++
		}

		if conn.Laddr.IP != "127.0.0.1" && conn.Laddr.IP != "::1" && conn.Laddr.IP != "0.0.0.0" {
			ext.ConnToLocal[conn.Laddr.IP]++
		}

		if conn.Raddr.IP != "" && conn.Raddr.IP != "127.0.0.1" && conn.Raddr.IP != "::1" {
			ext.ConnFromRemote[conn.Raddr.IP]++
		}

		portCounts[int(conn.Laddr.Port)]++
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
	return nil
}

func (c *LocalCollector) collectProcessExt(ctx context.Context, info *SystemInfo) error {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return err
	}

	ext := &ProcessExtInfo{
		NotableProcesses: make([]NotableProc, 0),
	}

	// Only collect on Linux, and limit to first 50 processes for performance
	if runtime.GOOS == "linux" {
		maxProcs := 50
		if len(procs) < maxProcs {
			maxProcs = len(procs)
		}
		for i := 0; i < maxProcs; i++ {
			p := procs[i]
			oomScorePath := "/proc/" + strconv.Itoa(int(p.Pid)) + "/oom_score"
			data, err := os.ReadFile(oomScorePath)
			if err == nil {
				if val, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && val != 0 {
					name, _ := p.Name()
					ext.NotableProcesses = append(ext.NotableProcesses, NotableProc{
						PID:    p.Pid,
						OOMAdj: val,
						Name:   name,
					})
				}
			}
		}
	}

	info.ProcessExt = ext
	return nil
}
