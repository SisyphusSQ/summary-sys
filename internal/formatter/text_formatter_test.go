package formatter

import (
	"strings"
	"testing"
	"time"

	"github.com/SisyphusSQ/summary-sys/internal/collector"
)

func baseSystemInfo() *collector.SystemInfo {
	return &collector.SystemInfo{
		Hostname:     "test-host",
		OS:           "Ubuntu 24.04.2 LTS",
		Platform:     "linux",
		Kernel:       "6.8.0-test",
		Architecture: "64-bit",
		Uptime:       49*time.Hour + 3*time.Minute,
		Timestamp:    time.Date(2026, 3, 18, 10, 30, 0, 0, time.FixedZone("CST", 8*3600)),
		LoadAvg: &collector.LoadAvgInfo{
			Load1:  0.12,
			Load5:  0.34,
			Load15: 0.56,
		},
		Who: []*collector.WhoEntry{
			{User: "root"},
			{User: "ubuntu"},
		},
	}
}

func TestFormatIncludesMountedFilesystemsSection(t *testing.T) {
	f := NewTextFormatter()
	info := baseSystemInfo()
	info.Disk = &collector.DiskInfo{
		{
			Device:      "/dev/sda1",
			MountPoint:  "/",
			FSType:      "ext4",
			Total:       500000000000,
			UsedPercent: 50,
		},
		{
			Device:      "/dev/nvme0n1p2",
			MountPoint:  "/var/lib/docker/containers/abc123def456",
			FSType:      "xfs",
			Total:       1000000000000,
			UsedPercent: 30,
		},
	}

	output, err := f.Format(info)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	for _, want := range []string{
		"# Mounted Filesystems",
		"/dev/sda1",
		"/dev/nvme0n1p2",
		"/var/lib/docker/containers/abc123def456",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}

func TestFormatIncludesInterfaceStatisticsSection(t *testing.T) {
	f := NewTextFormatter()
	info := baseSystemInfo()
	info.Network = &collector.NetworkInfo{
		Interfaces: []collector.NetInterface{
			{
				Name: "eth0",
				Statistics: &collector.NetStats{
					BytesRecv:   1000000,
					PacketsRecv: 100,
					ErrIn:       1,
					BytesSent:   2000000,
					PacketsSent: 200,
					ErrOut:      2,
				},
			},
			{
				Name: "br-abc123def456",
				Statistics: &collector.NetStats{
					BytesRecv:   5000000,
					PacketsRecv: 300,
					ErrIn:       0,
					BytesSent:   3000000,
					PacketsSent: 400,
					ErrOut:      0,
				},
			},
		},
	}
	info.NetworkExt = &collector.NetworkExtInfo{
		ConnFromRemote: map[string]int{"10.0.0.8": 3},
		ConnToPorts: []collector.PortStat{
			{Port: 22, Count: 3},
		},
		ConnStates: map[string]int{"ESTABLISHED": 3},
	}

	output, err := f.Format(info)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	for _, want := range []string{
		"# Interface Statistics",
		"eth0",
		"br-abc123de",
		"# Network Connections",
		"10.0.0.8",
		"ESTABLISHED",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}

func TestFormatIncludesProcessSections(t *testing.T) {
	f := NewTextFormatter()
	info := baseSystemInfo()
	info.Process = &collector.ProcessInfo{
		Count: 2,
		TopCPU: []*collector.ProcessStat{
			{
				PID:        1234,
				Name:       "mysqld",
				User:       "root",
				CPUPercent: 10.5,
				MemPercent: 5.2,
				Status:     "R",
				Virt:       8 * 1024 * 1024 * 1024,
				Res:        2 * 1024 * 1024 * 1024,
				Shr:        256 * 1024 * 1024,
			},
		},
	}
	info.ProcessExt = &collector.ProcessExtInfo{
		NotableProcesses: []collector.NotableProc{
			{PID: 4321, OOMAdj: 500, Name: "java"},
		},
	}

	output, err := f.Format(info)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	for _, want := range []string{
		"# Top Processes",
		"mysqld",
		"# Notable Processes",
		"java",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}
