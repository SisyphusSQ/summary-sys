package collector

import (
	"context"
	"fmt"
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
DF=$(df -B1 -k --output=source,fstype,size,used,avail,pcent 2>/dev/null | tail -n +2)
LOAD=$(cat /proc/loadavg 2>/dev/null)
NETSTAT=$(ss -tan 2>/dev/null | wc -l)

echo "HOSTNAME:$HOSTNAME"
echo "UNAME:$UNAME"
echo "UPTIME:$UPTIME"
echo "NPROC:$NPROC"
echo "MEM:$MEM"
echo "DF:$DF"
echo "LOAD:$LOAD"
echo "NETSTAT:$NETSTAT"
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
