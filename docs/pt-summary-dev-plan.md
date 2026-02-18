# pt-summary Go 重构项目开发计划

## 项目概述

将 Percona Toolkit 中的 `pt-summary` 系统摘要工具用 Go 重构，支持：
- 本地系统信息采集（CPU、内存、磁盘、网络、进程等）
- SSH 远程采集
- 格式化报告输出
- 可选 MCP Server 封装

## 技术选型

| 组件 | 技术 | 版本 |
|------|------|------|
| 本地采集 | github.com/shirou/gopsutil/v3 | latest |
| SSH | golang.org/x/crypto/ssh | latest |
| CLI | github.com/spf13/cobra | 1.10.2 (已有) |
| 日志 | go.uber.org/zap | 1.27.1 (已有) |
| MCP | github.com/mark3labs/mcp-go | 0.44.0 (已有) |

---

## Phase 1: 项目初始化

### 1.1 添加依赖

**文件**: `go.mod`

```go
require (
    github.com/shirou/gopsutil/v3 v3.25.1
    golang.org/x/crypto v0.35.0
)
```

**命令**:
```bash
go get github.com/shirou/gopsutil/v3
go get golang.org/x/crypto
go mod tidy
```

### 1.2 创建目录结构

```
cmd/pt-summary/
├── main.go              # CLI 入口
internal/
├── collector/
│   ├── collector.go    # 接口定义
│   ├── local.go        # 本地采集
│   ├── remote.go       # SSH 远程采集
│   └── types.go        # 数据结构
├── ssh/
│   ├── client.go       # SSH 连接与执行
│   └── auth.go         # 认证方式
├── formatter/
│   ├── formatter.go    # 接口定义
│   ├── text.go        # 文本格式
│   ├── json.go        # JSON 格式
│   └── html.go        # HTML 格式 (可选)
└── report/
    ├── report.go       # 报告生成
    └── sections/       # 各系统模块
        ├── cpu.go
        ├── memory.go
        ├── disk.go
        ├── network.go
        ├── process.go
        └── system.go
```

---

## Phase 2: Core 数据结构定义

### 2.1 采集结果数据结构

**文件**: `internal/collector/types.go`

```go
package collector

import "time"

// SystemInfo 采集的系统信息汇总
type SystemInfo struct {
    Hostname     string         `json:"hostname"`
    OS           string         `json:"os"`
    Platform     string         `json:"platform"`
    Kernel       string         `json:"kernel"`
    Architecture string         `json:"architecture"`
    Uptime       time.Duration `json:"uptime"`
    Timestamp    time.Time      `json:"timestamp"`
    
    CPU      *CPUInfo      `json:"cpu,omitempty"`
    Memory   *MemoryInfo   `json:"memory,omitempty"`
    Disk     *DiskInfo     `json:"disk,omitempty"`
    Network  *NetworkInfo  `json:"network,omitempty"`
    Process  *ProcessInfo  `json:"process,omitempty"`
    LoadAvg  *LoadAvgInfo  `json:"load_avg,omitempty"`
    Who      []*WhoInfo    `json:"who,omitempty"`
}

// CPUInfo CPU 信息
type CPUInfo struct {
    PhysicalCores int     `json:"physical_cores"`
    LogicalCores  int     `json:"logical_cores"`
    Models        []string `json:"models"`
    UsagePercent float64 `json:"usage_percent"`
    CStates       []CPUState `json:"c_states,omitempty"`
}

// CPUState C 状态
type CPUState struct {
    Name  string  `json:"name"`
    Usage float64 `json:"usage"`
}

// MemoryInfo 内存信息
type MemoryInfo struct {
    Total       uint64  `json:"total"`
    Available   uint64  `json:"available"`
    Used        uint64  `json:"used"`
    UsedPercent float64 `json:"used_percent"`
    SwapTotal   uint64  `json:"swap_total"`
    SwapUsed    uint64  `json:"swap_used"`
    SwapPercent float64 `json:"swap_percent"`
}

// DiskInfo 磁盘信息
type DiskInfo []DiskPartition

type DiskPartition struct {
    Device     string  `json:"device"`
    MountPoint string  `json:"mount_point"`
    FSType     string  `json:"fs_type"`
    Total      uint64  `json:"total"`
    Used       uint64  `json:"used"`
    Available  uint64  `json:"available"`
    UsedPercent float64 `json:"used_percent"`
}

// NetworkInfo 网络信息
type NetworkInfo struct {
    Interfaces  []NetInterface `json:"interfaces"`
    NetstatTCP  int            `json:"netstat_tcp"`
    NetstatUDP  int            `json:"netstat_udp"`
    ConnStates  map[string]int `json:"conn_states"`
}

// NetInterface 网卡信息
type NetInterface struct {
    Name       string   `json:"name"`
    MTU        int      `json:"mtu"`
    Addrs      []string `json:"addrs"`
    Statistics *NetStats `json:"statistics,omitempty"`
}

// NetStats 网卡流量
type NetStats struct {
    BytesSent   uint64 `json:"bytes_sent"`
    BytesRecv   uint64 `json:"bytes_recv"`
    PacketsSent uint64 `json:"packets_sent"`
    PacketsRecv uint64 `json:"packets_recv"`
    ErrIn       uint64 `json:"err_in"`
    ErrOut      uint64 `json:"err_out"`
    DropIn      uint64 `json:"drop_in"`
    DropOut     uint64 `json:"drop_out"`
}

// ProcessInfo 进程信息
type ProcessInfo struct {
    Count       int           `json:"count"`
    TopCPU      []*ProcessStat `json:"top_cpu,omitempty"`
    TopMemory   []*ProcessStat `json:"top_memory,omitempty"`
}

// ProcessStat 进程统计
type ProcessStat struct {
    PID         int32  `json:"pid"`
    Name        string `json:"name"`
    User        string `json:"user"`
    CPUPercent  float64 `json:"cpu_percent"`
    MemPercent  float64 `json:"mem_percent"`
    Status      string `json:"status"`
    Command     string `json:"command"`
}

// LoadAvgInfo 负载信息
type LoadAvgInfo struct {
    Load1  float64 `json:"load_1"`
    Load5  float64 `json:"load_5"`
    Load15 float64 `json:"load_15"`
    Runable int    `json:"runable"`
    Total   int    `json:"total"`
}

// WhoInfo 登录用户信息
type WhoInfo []WhoEntry

type WhoEntry struct {
    Terminal string `json:"terminal"`
    User     string `json:"user"`
    Host     string `json:"host"`
    LoginTime time.Time `json:"login_time"`
}
```

### 2.2 配置结构

**文件**: `internal/config/config.go` (新建)

```go
package config

type Config struct {
    // 通用选项
    OutputFormat string `mapstructure:"output-format"` // text, json, html
    OutputFile   string `mapstructure:"output-file"`
    Timeout      int    `mapstructure:"timeout"` // 秒
    
    // SSH 选项
    SSH *SSHConfig `mapstructure:"ssh"`
    
    // 采集选项
    Collect *CollectConfig `mapstructure:"collect"`
}

type SSHConfig struct {
    Enabled    bool     `mapstructure:"enabled"`
    Hosts      []string `mapstructure:"hosts"`
    User       string   `mapstructure:"user"`
    Port       int      `mapstructure:"port"`
    AuthMethod string   `mapstructure:"auth-method"` // password, key, agent
    Password   string   `mapstructure:"password"`
    KeyPath    string   `mapstructure:"key-path"`
    Passphrase string   `mapstructure:"passphrase"`
}

type CollectConfig struct {
    CPU      bool `mapstructure:"cpu"`
    Memory   bool `mapstructure:"memory"`
    Disk     bool `mapstructure:"disk"`
    Network  bool `mapstructure:"network"`
    Process  bool `mapstructure:"process"`
    LoadAvg  bool `mapstructure:"load-avg"`
    Who      bool `mapstructure:"who"`
}
```

---

## Phase 3: 本地采集器实现

### 3.1 采集器接口

**文件**: `internal/collector/collector.go`

```go
package collector

// Collector 采集器接口
type Collector interface {
    // Collect 采集系统信息
    Collect(ctx context.Context) (*SystemInfo, error)
    // Name 返回采集器名称
    Name() string
}

// Option 采集器选项
type Option func(*options)

type options struct {
    timeout time.Duration
    collect *CollectConfig
}

func WithTimeout(timeout time.Duration) Option {
    return func(o *options) { o.timeout = timeout }
}

func WithCollectConfig(cfg *CollectConfig) Option {
    return func(o *options) { o.collect = cfg }
}
```

### 3.2 本地采集器实现

**文件**: `internal/collector/local.go`

```go
package collector

import (
    "context"
    "runtime"
    "time"

    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/disk"
    "github.com/shirou/gopsutil/v3/host"
    "github.com/shirou/gopsutil/v3/load"
    "github.com/shirou/gopsutil/v3/mem"
    "github.com/shirou/gopsutil/v3/net"
    "github.com/shirou/gopsutil/v3/process"
    "github.com/shirou/gopsutil/v3/user"
    
    l "github.com/SisyphusSQ/summary-sys/pkg/log"
)

// LocalCollector 本地系统信息采集器
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
    l.Logger.Info("start collecting local system info")
    start := time.Now()
    defer func() {
        l.Logger.Infof("local collection completed in %v", time.Since(start))
    }()

    info := &SystemInfo{
        Timestamp: time.Now(),
    }

    var errs []error

    // 基础系统信息
    if err := c.collectSystemInfo(ctx, info); err != nil {
        errs = append(errs, err)
    }

    // CPU
    if c.opts.collect.CPU {
        if err := c.collectCPU(ctx, info); err != nil {
            errs = append(errs, err)
        }
    }

    // Memory
    if c.opts.collect.Memory {
        if err := c.collectMemory(ctx, info); err != nil {
            errs = append(errs, err)
        }
    }

    // Disk
    if c.opts.collect.Disk {
        if err := c.collectDisk(ctx, info); err != nil {
            errs = append(errs, err)
        }
    }

    // Network
    if c.opts.collect.Network {
        if err := c.collectNetwork(ctx, info); err != nil {
            errs = append(errs, err)
        }
    }

    // Process
    if c.opts.collect.Process {
        if err := c.collectProcess(ctx, info); err != nil {
            errs = append(errs, err)
        }
    }

    // Load Average
    if c.opts.collect.LoadAvg {
        if err := c.collectLoadAvg(ctx, info); err != nil {
            errs = append(errs, err)
        }
    }

    // Who
    if c.opts.collect.Who {
        if err := c.collectWho(ctx, info); err != nil {
            errs = append(errs, err)
        }
    }

    if len(errs) > 0 {
        return info, nil // 返回部分数据
    }
    return info, nil
}

// 采集基础系统信息
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

// 采集 CPU 信息
func (c *LocalCollector) collectCPU(ctx context.Context, info *SystemInfo) error {
    // 物理核心数和逻辑核心数
    cpuCount, err := cpu.CountsWithContext(ctx, false)
    if err != nil {
        return err
    }
    logicalCount, err := cpu.CountsWithContext(ctx, true)
    if err != nil {
        return err
    }

    // CPU 型号
    cpuInfo, err := cpu.InfoWithContext(ctx)
    if err != nil {
        return err
    }
    models := make([]string, 0, len(cpuInfo))
    for _, ci := range cpuInfo {
        models = append(models, ci.ModelName)
    }

    // CPU 使用率
    percent, err := cpu.PercentWithContext(ctx, time.Second)
    if err != nil {
        return err
    }

    info.CPU = &CPUInfo{
        PhysicalCores: cpuCount,
        LogicalCores:  logicalCount,
        Models:        models,
        UsagePercent:  percent[0],
    }
    return nil
}

// 采集内存信息
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

// 采集磁盘信息
func (c *LocalCollector) collectDisk(ctx context.Context, info *SystemInfo) error {
    partitions, err := disk.PartitionsWithContext(ctx, false)
    if err != nil {
        return err
    }

    info.Disk = make(DiskInfo, 0, len(partitions))
    for _, p := range partitions {
        usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
        if err != nil {
            continue
        }
        info.Disk = append(info.Disk, DiskPartition{
            Device:     p.Device,
            MountPoint: p.Mountpoint,
            FSType:     p.Fstype,
            Total:      usage.Total,
            Used:       usage.Used,
            Available:  usage.Available,
            UsedPercent: usage.UsedPercent,
        })
    }
    return nil
}

// 采集网络信息
func (c *LocalCollector) collectNetwork(ctx context.Context, info *SystemInfo) error {
    interfaces, err := net.InterfacesWithContext(ctx)
    if err != nil {
        return err
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

        // 流量统计
        stats, err := net.IOCountersWithContext(ctx, iface.Name)
        if err == nil && len(stats) > 0 {
            ni.Statistics = &NetStats{
                BytesSent:   stats[0].BytesSent,
                BytesRecv:   stats[0].BytesRecv,
                PacketsSent: stats[0].PacketsSent,
                PacketsRecv: stats[0].PacketsRecv,
                ErrIn:       stats[0].Errin,
                ErrOut:      stats[0].Errout,
                DropIn:      stats[0].Dropin,
                DropOut:     stats[0].Dropout,
            }
        }
        netInfo.Interfaces = append(netInfo.Interfaces, ni)
    }

    // TCP/UDP 连接统计
    netstat, _ := net.ConnstatsWithContext(ctx, true)
    for _, s := range netstat {
        netInfo.ConnStates[s.Type.String()] = s.Stats["established"] + s.Stats["listen"]
        if s.Type.String() == "tcp" {
            netInfo.NetstatTCP = s.Stats["established"] + s.Stats["listen"]
        } else if s.Type.String() == "udp" {
            netInfo.NetstatUDP = s.Stats["established"] + s.Stats["listen"]
        }
    }

    info.Network = netInfo
    return nil
}

// 采集进程信息
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
            MemPercent: mem,
            Status:     status[0],
            Command:    cmd,
        }
        allProcs = append(allProcs, ps)
    }

    // 按 CPU 排序
    sort.Slice(allProcs, func(i, j int) bool {
        return allProcs[i].CPUPercent > allProcs[j].CPUPercent
    })
    if len(allProcs) > 10 {
        procInfo.TopCPU = allProcs[:10]
    } else {
        procInfo.TopCPU = allProcs
    }

    // 按内存排序
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

// 采集负载信息
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
        Load1:  avg.Load1,
        Load5:  avg.Load5,
        Load15: avg.Load15,
        Runable: misc.ProcsRunning,
        Total:   misc.ProcsTotal,
    }
    return nil
}

// 采集登录用户
func (c *LocalCollector) collectWho(ctx context.Context, info *SystemInfo) error {
    users, err := user.UsersWithContext(ctx)
    if err != nil {
        return err
    }

    info.Who = make([]*WhoInfo, 0, len(users))
    for _, u := range users {
        loginTime, _ := time.Parse("2006-01-02 15:04:05", u.Started)
        entry := WhoEntry{
            Terminal: u.Terminal,
            User:     u.User,
            Host:     u.Host,
            LoginTime: loginTime,
        }
        info.Who = append(info.Who, &entry)
    }
    return nil
}
```

> **注意**: local.go 需要添加 `sort` 包的 import。

---

## Phase 4: SSH 远程采集器实现

### 4.1 SSH 客户端

**文件**: `internal/ssh/client.go`

```go
package ssh

import (
    "bytes"
    "context"
    "fmt"
    "time"

    "golang.org/x/crypto/ssh"
    
    l "github.com/SisyphusSQ/summary-sys/pkg/log"
)

// Client SSH 客户端
type Client struct {
    config *Config
    conn   *ssh.Client
}

// Config SSH 配置
type Config struct {
    Host       string
    Port       int
    User       string
    AuthMethod AuthMethod
    Timeout    time.Duration
}

// NewClient 创建 SSH 客户端
func NewClient(cfg *Config) (*Client, error) {
    auth, err := cfg.AuthMethod.build()
    if err != nil {
        return nil, err
    }

    clientConfig := &ssh.ClientConfig{
        User:            cfg.User,
        Auth:            []ssh.AuthMethod{auth},
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
        Timeout:         cfg.Timeout,
    }

    addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
    conn, err := ssh.Dial("tcp", addr, clientConfig)
    if err != nil {
        return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
    }

    l.Logger.Debugf("ssh connected to %s", addr)
    return &Client{config: cfg, conn: conn}, nil
}

// Run 执行远程命令
func (c *Client) Run(ctx context.Context, cmd string) (string, string, error) {
    session, err := c.conn.NewSession()
    if err != nil {
        return "", "", fmt.Errorf("new session: %w", err)
    }
    defer session.Close()

    stdoutBuf := new(bytes.Buffer)
    stderrBuf := new(bytes.Buffer)
    session.Stdout = stdoutBuf
    session.Stderr = stderrBuf

    if err := session.Run(cmd); err != nil {
        return stdoutBuf.String(), stderrBuf.String(), err
    }

    return stdoutBuf.String(), stderrBuf.String(), nil
}

// Close 关闭连接
func (c *Client) Close() error {
    if c.conn != nil {
        return c.conn.Close()
    }
    return nil
}
```

### 4.2 SSH 认证方式

**文件**: `internal/ssh/auth.go`

```go
package ssh

import (
    "fmt"
    "os"
    "os/exec"

    "golang.org/x/crypto/ssh"
)

// AuthMethod 认证方式接口
type AuthMethod interface {
    build() (ssh.AuthMethod, error)
}

// PasswordAuth 密码认证
type PasswordAuth struct {
    Password string
}

func (p PasswordAuth) build() (ssh.AuthMethod, error) {
    return ssh.Password(p.Password), nil
}

// KeyAuth 密钥认证
type KeyAuth struct {
    KeyPath     string
    Passphrase string
}

func (k KeyAuth) build() (ssh.AuthMethod, error) {
    key, err := os.ReadFile(k.KeyPath)
    if err != nil {
        return nil, fmt.Errorf("read key file: %w", err)
    }

    var signer ssh.Signer
    if k.Passphrase != "" {
        signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(k.Passphrase))
        if err != nil {
            return nil, fmt.Errorf("parse key with passphrase: %w", err)
        }
    } else {
        signer, err = ssh.ParsePrivateKey(key)
        if err != nil {
            return nil, fmt.Errorf("parse key: %w", err)
        }
    }

    return ssh.PublicKeys(signer), nil
}

// AgentAuth SSH Agent 认证
type AgentAuth struct{}

func (a AgentAuth) build() (ssh.AuthMethod, error) {
    cmd := exec.Command("ssh-agent")
    out, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("ssh-agent not available: %w", err)
    }
    // 使用系统 SSH Agent
    return ssh.Agent(), nil
}
```

### 4.3 远程采集器

**文件**: `internal/collector/remote.go`

```go
package collector

import (
    "context"
    "fmt"
    "time"

    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/disk"
    "github.com/shirou/gopsutil/v3/host"
    "github.com/shirou/gopsutil/v3/load"
    "github.com/shirou/gopsutil/v3/mem"
    "github.com/shirou/gopsutil/v3/net"
    "github.com/shirou/gopsutil/v3/process"
    "github.com/shirou/gopsutil/v3/user"

    "github.com/SisyphusSQ/summary-sys/internal/ssh"
    l "github.com/SisyphusSQ/summary-sys/pkg/log"
)

// RemoteCollector SSH 远程采集器
type RemoteCollector struct {
    client *ssh.Client
    opts   *options
}

// NewRemoteCollector 创建远程采集器
func NewRemoteCollector(sshCfg *ssh.Config, opts ...Option) (*RemoteCollector, error) {
    client, err := ssh.NewClient(sshCfg)
    if err != nil {
        return nil, err
    }

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

    return &RemoteCollector{
        client: client,
        opts:   o,
    }, nil
}

func (c *RemoteCollector) Name() string {
    return "remote:" + c.client.config.Host
}

func (c *RemoteCollector) Collect(ctx context.Context) (*SystemInfo, error) {
    l.Logger.Infof("start collecting remote system info from %s", c.client.config.Host)
    start := time.Now()
    defer func() {
        l.Logger.Infof("remote collection completed in %v", time.Since(start))
    }()

    info := &SystemInfo{
        Timestamp: time.Now(),
    }

    // 远程采集使用命令执行 + 解析的方式
    // 这里复用 LocalCollector 的逻辑，通过远程执行命令获取数据
    
    // 简化实现：直接通过远程命令收集关键信息
    // 实际生产中可以使用 JSON 输出模式远程执行
    
    if err := c.collectViaCommands(ctx, info); err != nil {
        l.Logger.Warnf("remote collection error: %v", err)
    }

    return info, nil
}

func (c *RemoteCollector) collectViaCommands(ctx context.Context, info *SystemInfo) error {
    // 1. Hostname
    stdout, _, err := c.client.Run(ctx, "hostname")
    if err == nil {
        info.Hostname = stdout[:len(stdout)-1] // 去掉换行
    }

    // 2. OS 信息
    stdout, _, _ = c.client.Run(ctx, "cat /etc/os-release 2>/dev/null || uname -a")
    info.OS = stdout

    // 3. Uptime
    stdout, _, _ = c.client.Run(ctx, "cat /proc/uptime | awk '{print $1}'")
    if uptime, ok := parseFloat(stdout); ok {
        info.Uptime = time.Duration(uptime) * time.Second
    }

    // 4. CPU 核心数
    stdout, _, _ = c.client.Run(ctx, "nproc")
    if n, ok := parseInt(stdout); ok {
        info.CPU = &CPUInfo{
            PhysicalCores: n,
            LogicalCores:  n,
        }
    }

    // 5. 内存
    stdout, _, _ = c.client.Run(ctx, "free -b | awk '/Mem:/ {print $2,$3,$7}'")
    var total, used, available uint64
    fmt.Sscanf(stdout, "%d %d %d", &total, &used, &available)
    if total > 0 {
        info.Memory = &MemoryInfo{
            Total:       total,
            Used:        used,
            Available:   available,
            UsedPercent: float64(used) / float64(total) * 100,
        }
    }

    // 6. 磁盘
    stdout, _, _ = c.client.Run(ctx, "df -B1 -k --output=source,fstype,size,used,avail,pcent 2>/dev/null | tail -n +2")
    info.Disk = parseDiskInfo(stdout)

    // 7. Load Average
    stdout, _, _ = c.client.Run(ctx, "cat /proc/loadavg")
    var l1, l5, l15 float64
    fmt.Sscanf(stdout, "%f %f %f", &l1, &l5, &l15)
    info.LoadAvg = &LoadAvgInfo{
        Load1:  l1,
        Load5:  l5,
        Load15: l15,
    }

    // 8. 网络连接
    stdout, _, _ = c.client.Run(ctx, "ss -tan | wc -l")
    if n, ok := parseInt(stdout); ok {
        info.Network = &NetworkInfo{
            NetstatTCP: n,
        }
    }

    return nil
}

func (c *RemoteCollector) Close() error {
    return c.client.Close()
}

// 辅助函数
func parseFloat(s string) (float64, bool) {
    var f float64
    _, err := fmt.Sscanf(s, "%f", &f)
    return f, err == nil
}

func parseInt(s string) (int, bool) {
    var n int
    _, err := fmt.Sscanf(s, "%d", &n)
    return n, err == nil
}

func parseDiskInfo(output string) DiskInfo {
    var disks DiskInfo
    lines := splitLines(output)
    for _, line := range lines {
        var dev, fstype string
        var total, used, available uint64
        var usedPercent string
        if _, err := fmt.Sscanf(line, "%s %s %d %d %d %s", &dev, &fstype, &total, &used, &available, &usedPercent); err == nil {
            p := strings.TrimSuffix(usedPercent, "%")
            pct, _ := strconv.ParseFloat(p, 64)
            disks = append(disks, DiskPartition{
                Device:      dev,
                FSType:      fstype,
                Total:       total,
                Used:        used,
                Available:   available,
                UsedPercent: pct,
            })
        }
    }
    return disks
}

func splitLines(s string) []string {
    var lines []string
    for _, line := range strings.Split(s, "\n") {
        if strings.TrimSpace(line) != "" {
            lines = append(lines, line)
        }
    }
    return lines
}
```

> **注意**: remote.go 需要添加 `strconv`, `strings` 包的 import。

---

## Phase 5: Formatter 报告格式化

### 5.1 Formatter 接口

**文件**: `internal/formatter/formatter.go`

```go
package formatter

import "github.com/SisyphusSQ/summary-sys/internal/collector"

// Formatter 格式化器接口
type Formatter interface {
    // Format 将系统信息格式化为字符串
    Format(info *collector.SystemInfo) (string, error)
    // Name 返回格式化器名称
    Name() string
    // ContentType 返回 Content-Type
    ContentType() string
}

// NewFormatter 创建格式化器
func NewFormatter(format string) (Formatter, error) {
    switch format {
    case "text":
        return NewTextFormatter(), nil
    case "json":
        return NewJSONFormatter(), nil
    case "html":
        return NewHTMLFormatter(), nil
    default:
        return nil, nil
    }
}
```

### 5.2 Text Formatter (默认)

**文件**: `internal/formatter/text.go`

```go
package formatter

import (
    "fmt"
    "strings"
    "time"

    "github.com/SisyphusSQ/summary-sys/internal/collector"
)

// TextFormatter 文本格式化器
type TextFormatter struct{}

func NewTextFormatter() *TextFormatter {
    return &TextFormatter{}
}

func (f *TextFormatter) Name() string     { return "text" }
func (f *TextFormatter) ContentType() string { return "text/plain" }

func (f *TextFormatter) Format(info *collector.SystemInfo) (string, error) {
    var sb strings.Builder

    // 头部信息
    sb.WriteString("=" + strings.Repeat("=", 78) + "\n")
    sb.WriteString(fmt.Sprintf("# %s\n", info.Hostname))
    sb.WriteString(fmt.Sprintf("OS: %s %s\n", info.OS, info.Kernel))
    sb.WriteString(fmt.Sprintf("Uptime: %s\n", formatUptime(info.Uptime)))
    sb.WriteString(fmt.Sprintf("Time: %s\n", info.Timestamp.Format("2006-01-02 15:04:05")))
    sb.WriteString("=" + strings.Repeat("=", 78) + "\n\n")

    // CPU
    if info.CPU != nil {
        sb.WriteString(f.formatCPU(info.CPU))
    }

    // Memory
    if info.Memory != nil {
        sb.WriteString(f.formatMemory(info.Memory))
    }

    // Disk
    if info.Disk != nil {
        sb.WriteString(f.formatDisk(info.Disk))
    }

    // Network
    if info.Network != nil {
        sb.WriteString(f.formatNetwork(info.Network))
    }

    // Load Average
    if info.LoadAvg != nil {
        sb.WriteString(f.formatLoadAvg(info.LoadAvg))
    }

    // Process
    if info.Process != nil {
        sb.WriteString(f.formatProcess(info.Process))
    }

    // Who
    if info.Who != nil {
        sb.WriteString(f.formatWho(info.Who))
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
    sb.WriteString(fmt.Sprintf("%-20s %10s %10s %10s %8s\n", "Filesystem", "Size", "Used", "Avail", "Use%"))
    sb.WriteString(strings.Repeat("-", 70) + "\n")
    for _, d := range disk {
        sb.WriteString(fmt.Sprintf("%-20s %10s %10s %10s %7.1f%%\n",
            d.MountPoint, formatBytes(d.Total), formatBytes(d.Used),
            formatBytes(d.Available), d.UsedPercent))
    }
    return sb.String()
}

func (f *TextFormatter) formatNetwork(net *collector.NetworkInfo) string {
    var sb strings.Builder
    sb.WriteString("\n### Network ###\n")
    for _, iface := range net.Interfaces {
        sb.WriteString(fmt.Sprintf("%s: %s\n", iface.Name, strings.Join(iface.Addrs, ", ")))
        if iface.Statistics != nil {
            sb.WriteString(fmt.Sprintf("  RX: %s TX: %s\n",
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
    sb.WriteString("\nTop 10 CPU:\n")
    sb.WriteString(fmt.Sprintf("%-8s %-15s %8s %8s %s\n", "PID", "USER", "CPU%%", "MEM%%", "COMMAND"))
    for _, p := range proc.TopCPU {
        sb.WriteString(fmt.Sprintf("%-8d %-15s %7.1f %7.1f %s\n",
            p.PID, p.User, p.CPUPercent, p.MemPercent, p.Name))
    }
    sb.WriteString("\nTop 10 Memory:\n")
    sb.WriteString(fmt.Sprintf("%-8s %-15s %8s %8s %s\n", "PID", "USER", "CPU%%", "MEM%%", "COMMAND"))
    for _, p := range proc.TopMemory {
        sb.WriteString(fmt.Sprintf("%-8d %-15s %7.1f %7.1f %s\n",
            p.PID, p.User, p.CPUPercent, p.MemPercent, p.Name))
    }
    return sb.String()
}

func (f *TextFormatter) formatWho(who collector.WhoInfo) string {
    var sb strings.Builder
    sb.WriteString("\n### Logged In Users ###\n")
    if len(who) == 0 {
        sb.WriteString("No users logged in\n")
        return sb.String()
    }
    sb.WriteString(fmt.Sprintf("%-15s %-10s %-20s\n", "USER", "TTY", "TIME"))
    for _, w := range who {
        sb.WriteString(fmt.Sprintf("%-15s %-10s %-20s\n",
            w.User, w.Terminal, w.LoginTime.Format("2006-01-02 15:04")))
    }
    return sb.String()
}

// 辅助函数
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
    d -= h * time.Hour
    m := d / time.Minute
    if h > 24 {
        days := h / 24
        h = h % 24
        return fmt.Sprintf("%dd %dh %dm", days, h, m)
    }
    return fmt.Sprintf("%dh %dm", h, m)
}
```

### 5.3 JSON Formatter

**文件**: `internal/formatter/json.go`

```go
package formatter

import (
    "encoding/json"
    "strings"

    "github.com/SisyphusSQ/summary-sys/internal/collector"
)

// JSONFormatter JSON 格式化器
type JSONFormatter struct {
    Pretty bool
}

func NewJSONFormatter() *JSONFormatter {
    return &JSONFormatter{Pretty: true}
}

func (f *JSONFormatter) Name() string     { return "json" }
func (f *JSONFormatter) ContentType() string { return "application/json" }

func (f *JSONFormatter) Format(info *collector.SystemInfo) (string, error) {
    if f.Pretty {
        b, err := json.MarshalIndent(info, "", "  ")
        return string(b), err
    }
    b, err := json.Marshal(info)
    return string(b), err
}
```

### 5.4 HTML Formatter (可选)

**文件**: `internal/formatter/html.go`

```go
package formatter

import (
    "html/template"
    "strings"

    "github.com/SisyphusSQ/summary-sys/internal/collector"
)

// HTMLFormatter HTML 格式化器
type HTMLFormatter struct {
    template *template.Template
}

func NewHTMLFormatter() *HTMLFormatter {
    tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>System Summary - {{.Hostname}}</title>
    <style>
        body { font-family: monospace; margin: 20px; background: #f5f5f5; }
        .header { background: #333; color: #fff; padding: 20px; }
        .section { background: #fff; margin: 10px 0; padding: 15px; border-radius: 5px; }
        h2 { margin-top: 0; color: #333; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f0f0f0; }
        .metric { font-weight: bold; }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.Hostname}}</h1>
        <p>{{.OS}} | Uptime: {{.Uptime}}</p>
    </div>
    {{if .CPU}}
    <div class="section">
        <h2>CPU</h2>
        <p>Cores: {{.CPU.LogicalCores}} | Usage: {{.CPU.UsagePercent}}%</p>
    </div>
    {{end}}
    {{if .Memory}}
    <div class="section">
        <h2>Memory</h2>
        <p>Total: {{.Memory.Total}} | Used: {{.Memory.UsedPercent}}%</p>
    </div>
    {{end}}
</body>
</html>`
    return &HTMLFormatter{
        template: template.Must(template.New("html").Parse(tmpl)),
    }
}

func (f *HTMLFormatter) Name() string     { return "html" }
func (f *HTMLFormatter) ContentType() string { return "text/html" }

func (f *HTMLFormatter) Format(info *collector.SystemInfo) (string, error) {
    var sb strings.Builder
    err := f.template.Execute(&sb, info)
    return sb.String(), err
}
```

---

## Phase 6: CLI 命令封装

### 6.1 pt-summary 子命令

**文件**: `cmd/pt-summary/main.go`

```go
package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"

    l "github.com/SisyphusSQ/summary-sys/pkg/log"
    "github.com/SisyphusSQ/summary-sys/internal/collector"
    "github.com/SisyphusSQ/summary-sys/internal/formatter"
    "github.com/SisyphusSQ/summary-sys/internal/ssh"
    "github.com/SisyphusSQ/summary-sys/vars"
)

var (
    outputFormat string
    outputFile   string
    sshEnabled   bool
    sshHosts     []string
    sshUser      string
    sshPort      int
    sshKeyPath   string
    timeout      int
)

var ptSummaryCmd = &cobra.Command{
    Use:   "pt-summary",
    Short: "Collect and display system summary information",
    Long: `pt-summary collects system information including CPU, memory, disk, 
network, processes, and load average. It supports both local and remote 
(SSH) collection.`,
    RunE: runPtSummary,
}

func init() {
    // 输出选项
    ptSummaryCmd.Flags().StringVarP(&outputFormat, "format", "f", "text",
        "Output format: text, json, html")
    ptSummaryCmd.Flags().StringVarP(&outputFile, "output", "o", "",
        "Output file (default: stdout)")
    
    // SSH 选项
    ptSummaryCmd.Flags().BoolVar(&sshEnabled, "ssh", false,
        "Collect from remote hosts via SSH")
    ptSummaryCmd.Flags().StringSliceVar(&sshHosts, "hosts", []string{},
        "SSH hosts (can be repeated)")
    ptSummaryCmd.Flags().StringVar(&sshUser, "ssh-user", "root",
        "SSH username")
    ptSummaryCmd.Flags().IntVar(&sshPort, "ssh-port", 22,
        "SSH port")
    ptSummaryCmd.Flags().StringVar(&sshKeyPath, "ssh-key", "",
        "SSH private key path")
    
    // 超时
    ptSummaryCmd.Flags().IntVar(&timeout, "timeout", 30,
        "Collection timeout in seconds")
}

func runPtSummary(cmd *cobra.Command, args []string) error {
    // 创建格式化器
    fmter, err := formatter.NewFormatter(outputFormat)
    if err != nil {
        return fmt.Errorf("create formatter: %w", err)
    }

    var results []string

    if sshEnabled && len(sshHosts) > 0 {
        // SSH 远程采集
        for _, host := range sshHosts {
            l.Logger.Info("collecting from remote host", "host", host)
            
            client, err := ssh.NewClient(&ssh.Config{
                Host: host,
                Port: sshPort,
                User: sshUser,
                AuthMethod: ssh.KeyAuth{KeyPath: sshKeyPath},
            })
            if err != nil {
                l.Logger.Error("failed to connect SSH", "host", host, "error", err)
                continue
            }
            defer client.Close()

            remoteCol := collector.NewRemoteCollector(client)
            info, err := remoteCol.Collect(cmd.Context())
            if err != nil {
                l.Logger.Error("failed to collect from remote", "host", host, "error", err)
                continue
            }

            output, err := fmter.Format(info)
            if err != nil {
                l.Logger.Error("failed to format output", "error", err)
                continue
            }
            results = append(results, output)
        }
    } else {
        // 本地采集
        l.Logger.Info("collecting local system info")
        localCol := collector.NewLocalCollector()
        info, err := localCol.Collect(cmd.Context())
        if err != nil {
            return fmt.Errorf("collect local info: %w", err)
        }

        output, err := fmter.Format(info)
        if err != nil {
            return fmt.Errorf("format output: %w", err)
        }
        results = append(results, output)
    }

    // 输出结果
    for _, r := range results {
        if outputFile != "" {
            err := os.WriteFile(outputFile, []byte(r), 0644)
            if err != nil {
                return fmt.Errorf("write output file: %w", err)
            }
            l.Logger.Info("output written to", "file", outputFile)
        } else {
            fmt.Println(r)
        }
    }

    return nil
}

// 注册命令
func init() {
    // 注册到 root cmd
    // 方式 1: 作为子命令
    // rootCmd.AddCommand(ptSummaryCmd)
    
    // 方式 2: 独立命令入口
    vars.AppName = "pt-summary"
    ptSummaryCmd.SetVersionTemplate(fmt.Sprintf("pt-summary %s\n", vars.AppVersion))
}
```

### 6.2 主入口文件

**文件**: `cmd/pt-summary/main.go` (完整版入口)

```go
package main

import (
    "os"

    "github.com/spf13/cobra"

    "github.com/SisyphusSQ/summary-sys/cmd"
    l "github.com/SisyphusSQ/summary-sys/pkg/log"
)

func main() {
    // 初始化命令
    cmd.Execute()
}

// 也可以选择独立入口
/*
func main() {
    // 初始化日志
    _ = l.New(false, l.OutputStdio)
    
    // 直接运行
    rootCmd := &cobra.Command{Use: "pt-summary"}
    rootCmd.AddCommand(ptSummaryCmd)
    
    if err := rootCmd.Execute(); err != nil {
        l.Logger.Error("execution failed", "error", err)
        os.Exit(1)
    }
}
*/
```

---

## Phase 7: MCP Server 封装 (可选)

### 7.1 MCP Handler

**文件**: `internal/mcp/handler/system.go`

```go
package handler

import (
    "context"

    "github.com/mark3labs/mcp-go/server"
    "github.com/mark3labs/mcp-go/tool"

    "github.com/SisyphusSQ/summary-sys/internal/collector"
    "github.com/SisyphusSQ/summary-sys/internal/formatter"
    "github.com/SisyphusSQ/summary-sys/internal/ssh"
)

// SystemCollectorService 系统采集服务
type SystemCollectorService struct {
    localCollector *collector.LocalCollector
}

// NewSystemCollectorService 创建服务
func NewSystemCollectorService() *SystemCollectorService {
    return &SystemCollectorService{
        localCollector: collector.NewLocalCollector(),
    }
}

// InitSystemTools 注册 MCP 工具
func InitSystemTools(s *server.MCPServer, svc *SystemCollectorService) {
    // 1. 本地系统摘要
    s.AddTool(tool.NewTool(
        "system_summary_local",
        "Get local system summary (CPU, memory, disk, network, processes)",
        tool.NewSchema(
            "output_format", tool.Optional[string], "Output format: text, json",
        ),
        func(ctx context.Context, args map[string]any) (any, error) {
            outputFormat := "json"
            if v, ok := args["output_format"].(string); ok {
                outputFormat = v
            }

            info, err := svc.localCollector.Collect(ctx)
            if err != nil {
                return nil, err
            }

            fmter, err := formatter.NewFormatter(outputFormat)
            if err != nil {
                return nil, err
            }

            return fmter.Format(info)
        },
    ))

    // 2. CPU 信息
    s.AddTool(tool.NewTool(
        "system_cpu",
        "Get CPU information",
        tool.NewSchema(),
        func(ctx context.Context, args map[string]any) (any, error) {
            info, err := svc.localCollector.Collect(ctx)
            if err != nil {
                return nil, err
            }
            return info.CPU, nil
        },
    ))

    // 3. 内存信息
    s.AddTool(tool.NewTool(
        "system_memory",
        "Get memory information",
        tool.NewSchema(),
        func(ctx context.Context, args map[string]any) (any, error) {
            info, err := svc.localCollector.Collect(ctx)
            if err != nil {
                return nil, err
            }
            return info.Memory, nil
        },
    ))

    // 4. 磁盘信息
    s.AddTool(tool.NewTool(
        "system_disk",
        "Get disk usage information",
        tool.NewSchema(),
        func(ctx context.Context, args map[string]any) (any, error) {
            info, err := svc.localCollector.Collect(ctx)
            if err != nil {
                return nil, err
            }
            return info.Disk, nil
        },
    ))

    // 5. 网络信息
    s.AddTool(tool.NewTool(
        "system_network",
        "Get network information",
        tool.NewSchema(),
        func(ctx context.Context, args map[string]any) (any, error) {
            info, err := svc.localCollector.Collect(ctx)
            if err != nil {
                return nil, err
            }
            return info.Network, nil
        },
    ))
}

// RemoteCollectorService 远程采集服务
type RemoteCollectorService struct {
    sshConfig *ssh.Config
}

// NewRemoteCollectorService 创建远程采集服务
func NewRemoteCollectorService(sshCfg *ssh.Config) *RemoteCollectorService {
    return &RemoteCollectorService{sshConfig: sshCfg}
}

// InitRemoteTools 注册远程采集 MCP 工具
func InitRemoteTools(s *server.MCPServer, svc *RemoteCollectorService) {
    s.AddTool(tool.NewTool(
        "system_summary_remote",
        "Get remote system summary via SSH",
        tool.NewSchema(
            "host", tool.Required[string], "Remote host address",
            "output_format", tool.Optional[string], "Output format: text, json",
        ),
        func(ctx context.Context, args map[string]any) (any, error) {
            host, _ := args["host"].(string)
            outputFormat := "json"
            if v, ok := args["output_format"].(string); ok {
                outputFormat = v
            }

            client, err := ssh.NewClient(&ssh.Config{
                Host: host,
                Port: svc.sshConfig.Port,
                User: svc.sshConfig.User,
                AuthMethod: svc.sshConfig.AuthMethod,
            })
            if err != nil {
                return nil, err
            }
            defer client.Close()

            remoteCol, err := collector.NewRemoteCollector(client)
            if err != nil {
                return nil, err
            }

            info, err := remoteCol.Collect(ctx)
            if err != nil {
                return nil, err
            }

            fmter, err := formatter.NewFormatter(outputFormat)
            if err != nil {
                return nil, err
            }

            return fmter.Format(info)
        },
    ))
}
```

### 7.2 MCP Server 入口

**文件**: `cmd/mcp-system/main.go`

```go
package main

import (
    "fmt"

    "github.com/mark3labs/mcp-go/server"

    "github.com/SisyphusSQ/summary-sys/internal/mcp/handler"
    "github.com/SisyphusSQ/summary-sys/internal/mcp/server"
    "github.com/SisyphusSQ/summary-sys/vars"
)

func main() {
    s := server.NewServer("pt-summary-mcp", vars.AppVersion)

    // 本地采集工具
    localSvc := handler.NewSystemCollectorService()
    handler.InitSystemTools(s, localSvc)

    // 远程采集工具 (可选，需要配置 SSH)
    // remoteSvc := handler.NewRemoteCollectorService(&ssh.Config{...})
    // handler.InitRemoteTools(s, remoteSvc)

    if err := server.ServeStdio(s); err != nil {
        fmt.Fprintln(os.Stderr, "MCP server error:", err)
    }
}
```

---

## Phase 8: 测试计划

### 8.1 单元测试

| 测试文件 | 测试内容 |
|----------|----------|
| `internal/collector/collector_test.go` | 采集器接口实现测试 |
| `internal/collector/local_test.go` | 本地采集器测试 (需要 mock gopsutil) |
| `internal/formatter/text_test.go` | Text 格式化器测试 |
| `internal/formatter/json_test.go` | JSON 格式化器测试 |
| `internal/ssh/client_test.go` | SSH 客户端测试 (需要 mock ssh) |

### 8.2 集成测试

| 测试文件 | 测试内容 |
|----------|----------|
| `cmd/pt-summary/main_test.go` | CLI 端到端测试 |
| `integration/local_test.go` | 本地采集完整流程测试 |
| `integration/ssh_test.go` | SSH 远程采集测试 (需要 SSH server) |

### 8.3 示例测试用例

**文件**: `internal/formatter/text_test.go`

```go
package formatter

import (
    "testing"
    "time"

    "github.com/SisyphusSQ/summary-sys/internal/collector"
)

func TestTextFormatter_Format(t *testing.T) {
    fmter := NewTextFormatter()
    info := &collector.SystemInfo{
        Hostname:   "test-host",
        OS:         "Linux",
        Kernel:     "5.15.0",
        Uptime:     3600 * time.Second,
        Timestamp:  time.Now(),
        CPU: &collector.CPUInfo{
            PhysicalCores: 4,
            LogicalCores:  8,
            UsagePercent:  45.5,
            Models:        []string{"Intel(R) Xeon(R)"},
        },
        Memory: &collector.MemoryInfo{
            Total:       16 * 1024 * 1024 * 1024,
            Used:        8 * 1024 * 1024 * 1024,
            Available:   8 * 1024 * 1024 * 1024,
            UsedPercent: 50.0,
        },
    }

    output, err := fmter.Format(info)
    if err != nil {
        t.Fatalf("format failed: %v", err)
    }

    if !contains(output, "test-host") {
        t.Error("output should contain hostname")
    }
    if !contains(output, "CPU") {
        t.Error("output should contain CPU section")
    }
    if !contains(output, "Memory") {
        t.Error("output should contain Memory section")
    }
}

func contains(s, substr string) bool {
    return len(s) > 0 && len(substr) > 0 && 
           (len(s) >= len(substr) && 
            (s[:len(substr)] == substr || contains(s[1:], substr)))
}
```

---

## 实现顺序建议

```
Phase 1: 项目初始化
  ├── 添加依赖 (go get)
  └── 创建目录结构

Phase 2: Core 数据结构
  ├── internal/collector/types.go
  └── internal/config/config.go

Phase 3: 本地采集器
  ├── internal/collector/collector.go (接口)
  └── internal/collector/local.go (实现)

Phase 4: Formatter
  ├── internal/formatter/formatter.go
  ├── internal/formatter/text.go
  └── internal/formatter/json.go

Phase 5: SSH 远程采集器
  ├── internal/ssh/client.go
  ├── internal/ssh/auth.go
  └── internal/collector/remote.go

Phase 6: CLI 命令
  └── cmd/pt-summary/main.go

Phase 7: MCP Server (可选)
  ├── internal/mcp/handler/system.go
  └── cmd/mcp-system/main.go

Phase 8: 测试
  └── 各模块测试文件
```

---

## 参考命令

```bash
# 开发调试
go run ./cmd/pt-summary/main.go
go run ./cmd/pt-summary/main.go --debug
go run ./cmd/pt-summary/main.go --format=json

# SSH 远程采集
go run ./cmd/pt-summary/main.go --ssh --hosts=192.168.1.10 --ssh-user=root

# MCP Server
go run ./cmd/mcp-system/main.go

# 测试
go test ./internal/collector/...
go test ./internal/formatter/...
go test -race ./...

# 构建
go build -o bin/pt-summary ./cmd/pt-summary
go build -o bin/pt-summary-mcp ./cmd/mcp-system
```

---

## 注意事项

1. **gopsutil 兼容性**: 部分函数需要根据实际运行平台判断是否可用
2. **SSH 认证**: 生产环境建议使用密钥 + passphrase，避免密码明文
3. **超时处理**: 远程采集需要合理的超时设置
4. **错误处理**: 部分采集失败不应影响其他模块
5. **权限**: 某些系统信息可能需要 root 权限

---

*文档版本: v1.0*
*最后更新: 2026-02-18*
