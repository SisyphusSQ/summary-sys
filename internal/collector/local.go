package collector

import (
	"context"
	"runtime"
	"sort"
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
	info.OS = h.OS
	info.Platform = h.Platform
	info.Kernel = h.KernelVersion
	info.Architecture = runtime.GOARCH
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

	percent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return err
	}

	models := make([]string, 0, len(cpuInfo))
	for _, ci := range cpuInfo {
		models = append(models, ci.ModelName)
	}

	info.CPU = &CPUInfo{
		PhysicalCores: physical,
		LogicalCores:  logical,
		Models:        models,
		UsagePercent:  percent[0],
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

		ps := &ProcessStat{
			PID:        p.Pid,
			Name:       name,
			User:       username,
			CPUPercent: cpu,
			MemPercent: float64(mem),
			Status:     status[0],
			Command:    cmd,
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
	return nil
}
