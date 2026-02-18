package collector

import "time"

// SystemInfo 采集的系统信息汇总
type SystemInfo struct {
	Hostname     string        `json:"hostname"`
	OS           string        `json:"os"`
	Platform     string        `json:"platform"`
	Kernel       string        `json:"kernel"`
	Architecture string        `json:"architecture"`
	Uptime       time.Duration `json:"uptime"`
	Timestamp    time.Time     `json:"timestamp"`

	CPU     *CPUInfo     `json:"cpu,omitempty"`
	Memory  *MemoryInfo  `json:"memory,omitempty"`
	Disk    *DiskInfo    `json:"disk,omitempty"`
	Network *NetworkInfo `json:"network,omitempty"`
	Process *ProcessInfo `json:"process,omitempty"`
	LoadAvg *LoadAvgInfo `json:"load_avg,omitempty"`
	Who     []*WhoEntry  `json:"who,omitempty"`
}

// CPUInfo CPU 信息
type CPUInfo struct {
	PhysicalCores int        `json:"physical_cores"`
	LogicalCores  int        `json:"logical_cores"`
	Models        []string   `json:"models"`
	UsagePercent  float64    `json:"usage_percent"`
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
	Device      string  `json:"device"`
	MountPoint  string  `json:"mount_point"`
	FSType      string  `json:"fs_type"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkInfo 网络信息
type NetworkInfo struct {
	Interfaces []NetInterface `json:"interfaces"`
	NetstatTCP int            `json:"netstat_tcp"`
	NetstatUDP int            `json:"netstat_udp"`
	ConnStates map[string]int `json:"conn_states"`
}

// NetInterface 网卡信息
type NetInterface struct {
	Name       string    `json:"name"`
	MTU        int       `json:"mtu"`
	Addrs      []string  `json:"addrs"`
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
	Count     int            `json:"count"`
	TopCPU    []*ProcessStat `json:"top_cpu,omitempty"`
	TopMemory []*ProcessStat `json:"top_memory,omitempty"`
}

// ProcessStat 进程统计
type ProcessStat struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	User       string  `json:"user"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float64 `json:"mem_percent"`
	Status     string  `json:"status"`
	Command    string  `json:"command"`
}

// LoadAvgInfo 负载信息
type LoadAvgInfo struct {
	Load1   float64 `json:"load_1"`
	Load5   float64 `json:"load_5"`
	Load15  float64 `json:"load_15"`
	Runable int     `json:"runable"`
	Total   int     `json:"total"`
}

// WhoEntry 登录用户信息
type WhoEntry struct {
	Terminal  string    `json:"terminal"`
	User      string    `json:"user"`
	Host      string    `json:"host"`
	LoginTime time.Time `json:"login_time"`
}

// CollectConfig 采集配置
type CollectConfig struct {
	CPU     bool `mapstructure:"cpu"`
	Memory  bool `mapstructure:"memory"`
	Disk    bool `mapstructure:"disk"`
	Network bool `mapstructure:"network"`
	Process bool `mapstructure:"process"`
	LoadAvg bool `mapstructure:"load-avg"`
	Who     bool `mapstructure:"who"`
}
