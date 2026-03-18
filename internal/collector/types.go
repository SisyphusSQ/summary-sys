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

	// 扩展字段 (可选，按需采集)
	SystemExt  *SystemExtInfo  `json:"system_ext,omitempty"`
	CPUExt     *CPUExtInfo    `json:"cpu_ext,omitempty"`
	MemoryExt  *MemoryExtInfo `json:"memory_ext,omitempty"`
	DiskExt    *DiskExtInfo   `json:"disk_ext,omitempty"`
	KernelExt  *KernelExtInfo `json:"kernel_ext,omitempty"`
	NetworkExt *NetworkExtInfo `json:"network_ext,omitempty"`
	ProcessExt *ProcessExtInfo `json:"process_ext,omitempty"`
}

// SystemExtInfo 系统扩展信息
type SystemExtInfo struct {
	Vendor      string `json:"vendor,omitempty"`
	Model       string `json:"model,omitempty"`
	Version     string `json:"version,omitempty"`
	ServiceTag  string `json:"service_tag,omitempty"`
	Threading   string `json:"threading,omitempty"`
	SELinux     string `json:"selinux,omitempty"`
	Virtualized string `json:"virtualized,omitempty"`
}

// CPUExtInfo CPU 扩展信息
type CPUExtInfo struct {
	Speeds         []float64   `json:"speeds,omitempty"`
	Caches         []CacheInfo `json:"caches,omitempty"`
	Hyperthreading bool        `json:"hyperthreading"`
}

// CacheInfo CPU 缓存信息
type CacheInfo struct {
	Level          string `json:"level"`
	Size           uint64 `json:"size"`
	Associativity  string `json:"associativity"`
}

// MemoryExtInfo 内存扩展信息
type MemoryExtInfo struct {
	Free        uint64     `json:"free"`
	Shared      uint64     `json:"shared"`
	Buffers     uint64     `json:"buffers"`
	Caches      uint64     `json:"caches"`
	Dirty       uint64     `json:"dirty"`
	UsedRSS     uint64     `json:"used_rss"`
	Swappiness  int        `json:"swappiness"`
	DirtyPolicy string     `json:"dirty_policy,omitempty"`
	DirtyStatus string     `json:"dirty_status,omitempty"`
	NumaNodes   []NumaNode `json:"numa_nodes,omitempty"`
	Dimms       []DimmInfo `json:"dimms,omitempty"`
}

// NumaNode NUMA 节点信息
type NumaNode struct {
	ID   int    `json:"id"`
	Size uint64 `json:"size"`
	Free uint64 `json:"free"`
	CPUs []int  `json:"cpus"`
}

// DimmInfo DIMM 插槽信息
type DimmInfo struct {
	Locator    string `json:"locator,omitempty"`
	Size       uint64 `json:"size"`
	Speed      string `json:"speed,omitempty"`
	FormFactor string `json:"form_factor,omitempty"`
	Type       string `json:"type,omitempty"`
	TypeDetail string `json:"type_detail,omitempty"`
}

// DiskExtInfo 磁盘扩展信息
type DiskExtInfo struct {
	Schedulers map[string]string    `json:"schedulers,omitempty"`
	Partitions []PartitionInfo     `json:"partitions,omitempty"`
}

// PartitionInfo 分区信息
type PartitionInfo struct {
	Device string `json:"device"`
	Type   string `json:"type"`
	Start  uint64 `json:"start"`
	End    uint64 `json:"end"`
	Size   uint64 `json:"size"`
}

// KernelExtInfo 内核扩展信息
type KernelExtInfo struct {
	DentryState string `json:"dentry_state,omitempty"`
	FileNr     string `json:"file_nr,omitempty"`
	InodeNr    string `json:"inode_nr,omitempty"`
	THPEnabled bool   `json:"thp_enabled"`
}

// NetworkExtInfo 网络扩展信息
type NetworkExtInfo struct {
	Controllers    []string            `json:"controllers,omitempty"`
	FinTimeout     int                 `json:"fin_timeout"`
	PortRange      int                 `json:"port_range"`
	ConnFromRemote map[string]int      `json:"conn_from_remote,omitempty"`
	ConnToLocal    map[string]int      `json:"conn_to_local,omitempty"`
	ConnToPorts    []PortStat          `json:"conn_to_ports,omitempty"`
	ConnStates     map[string]int      `json:"conn_states,omitempty"`
	NetDevices     []NetDevice         `json:"net_devices,omitempty"`
}

// NetDevice 网卡设备信息
type NetDevice struct {
	Name     string `json:"name"`
	Speed    string `json:"speed,omitempty"`
	Duplex   string `json:"duplex,omitempty"`
}

// PortStat 端口连接统计
type PortStat struct {
	Port  int `json:"port"`
	Count int `json:"count"`
}

// ProcessExtInfo 进程扩展信息
type ProcessExtInfo struct {
	NotableProcesses []NotableProc `json:"notable_processes,omitempty"`
}

// NotableProc 特殊进程信息
type NotableProc struct {
	PID    int32 `json:"pid"`
	OOMAdj int   `json:"oom_adj"`
	Name   string `json:"name"`
}

// CPUInfo CPU 信息
type CPUInfo struct {
	PhysicalCores int        `json:"physical_cores"`
	LogicalCores  int        `json:"logical_cores"`
	Sockets       int        `json:"sockets,omitempty"`
	CoresPerSocket int      `json:"cores_per_socket,omitempty"`
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
	Virt       uint64  `json:"virt,omitempty"`
	Res        uint64  `json:"res,omitempty"`
	Shr        uint64  `json:"shr,omitempty"`
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

	// 扩展配置
	SystemExt   bool `mapstructure:"system-ext"`
	CPUDetail   bool `mapstructure:"cpu-detail"`
	MemoryDetail bool `mapstructure:"memory-detail"`
	DiskExt     bool `mapstructure:"disk-ext"`
	NetworkExt  bool `mapstructure:"network-ext"`
	KernelExt   bool `mapstructure:"kernel-ext"`
	ProcessExt  bool `mapstructure:"process-ext"`
	Vmstat      bool `mapstructure:"vmstat"`
}
