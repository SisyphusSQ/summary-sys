package formatter

import (
	"strings"
	"testing"

	"github.com/SisyphusSQ/summary-sys/internal/collector"
)

func TestFormatNetwork_Alignment(t *testing.T) {
	f := NewTextFormatter()

	net := &collector.NetworkInfo{
		Interfaces: []collector.NetInterface{
			{
				Name:  "lo",
				MTU:   65536,
				Addrs: []string{"127.0.0.1/8"},
			},
			{
				Name:  "eth0",
				MTU:   1500,
				Addrs: []string{"192.168.1.100/24"},
				Statistics: &collector.NetStats{
					BytesRecv: 1000000,
					BytesSent: 2000000,
				},
			},
			{
				Name:  "br-abc123def456",
				MTU:   1500,
				Addrs: []string{"172.17.0.1/16"},
				Statistics: &collector.NetStats{
					BytesRecv: 5000000,
					BytesSent: 3000000,
				},
			},
		},
		NetstatTCP: 100,
		NetstatUDP: 50,
	}

	output := f.formatNetwork(net)
	t.Logf("Network output:\n%s", output)

	lines := strings.Split(output, "\n")
	if len(lines) < 3 {
		t.Fatalf("Expected at least 3 lines, got %d", len(lines))
	}

	maxNameLen := 0
	for _, iface := range net.Interfaces {
		if len(iface.Name) > maxNameLen {
			maxNameLen = len(iface.Name)
		}
	}

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "###") {
			continue
		}
		if strings.Contains(line, "Connections:") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && !strings.Contains(line, "RX:") {
			name := parts[0]
			if len(name) != maxNameLen {
				t.Errorf("Interface name not aligned: expected width=%d, got=%d, name=%q",
					maxNameLen, len(name), name)
			}
		}
	}
}

func TestFormatProcess_Alignment(t *testing.T) {
	f := NewTextFormatter()

	proc := &collector.ProcessInfo{
		Count: 100,
		TopCPU: []*collector.ProcessStat{
			{
				PID:        1234,
				Name:       "short",
				User:       "root",
				CPUPercent: 10.5,
				MemPercent: 5.2,
			},
			{
				PID:        5678,
				Name:       "very-long-process-name-here",
				User:       "admin",
				CPUPercent: 8.3,
				MemPercent: 3.1,
			},
			{
				PID:        9999,
				Name:       "nginx",
				User:       "www-data",
				CPUPercent: 5.0,
				MemPercent: 2.0,
			},
		},
		TopMemory: []*collector.ProcessStat{
			{
				PID:        4321,
				Name:       "chrome",
				User:       "user",
				CPUPercent: 2.0,
				MemPercent: 15.5,
			},
		},
	}

	output := f.formatProcess(proc)
	t.Logf("Process output:\n%s", output)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "###") {
			continue
		}
		if strings.HasPrefix(line, "Top") {
			continue
		}
		if !strings.Contains(line, "root") && !strings.Contains(line, "admin") && !strings.Contains(line, "user") && !strings.Contains(line, "www-data") {
			continue
		}
		t.Logf("Data line: %q (len=%d)", line, len(line))
	}
}

func TestFormatDisk_LongMountPoint(t *testing.T) {
	f := NewTextFormatter()

	// Test case 1: Normal mount points (should work fine)
	normalDisk := collector.DiskInfo{
		{
			Device:      "/dev/sda1",
			MountPoint:  "/",
			FSType:      "ext4",
			Total:       500000000000,
			Used:        250000000000,
			Available:   250000000000,
			UsedPercent: 50.0,
		},
		{
			Device:      "/dev/sda2",
			MountPoint:  "/home",
			FSType:      "ext4",
			Total:       1000000000000,
			Used:        300000000000,
			Available:   700000000000,
			UsedPercent: 30.0,
		},
	}

	normalOutput := f.formatDisk(normalDisk)
	t.Logf("Normal output:\n%s", normalOutput)

	// Verify alignment by checking line lengths
	lines := strings.Split(normalOutput, "\n")
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 lines, got %d", len(lines))
	}

	// Find header line (contains "Filesystem")
	headerLen := 0
	for _, line := range lines {
		if strings.Contains(line, "Filesystem") && strings.Contains(line, "Size") {
			headerLen = len(line)
			break
		}
	}

	if headerLen == 0 {
		t.Fatal("Could not find header line")
	}

	// Check all data lines have consistent length
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		if !strings.HasPrefix(line, "/") {
			continue
		}
		if len(line) != headerLen {
			t.Errorf("Line %d length mismatch: expected=%d, got=%d, line=%q",
				i, headerLen, len(line), line)
		}
	}

	// Test case 2: Long mount points (triggers alignment issue)
	longDisk := collector.DiskInfo{
		{
			Device:      "/dev/sda1",
			MountPoint:  "/",
			FSType:      "ext4",
			Total:       500000000000,
			Used:        250000000000,
			Available:   250000000000,
			UsedPercent: 50.0,
		},
		{
			Device:      "/dev/nvme0n1p2",
			MountPoint:  "/var/lib/docker/containers/abc123def456",
			FSType:      "ext4",
			Total:       1000000000000,
			Used:        300000000000,
			Available:   700000000000,
			UsedPercent: 30.0,
		},
		{
			Device:      "/dev/sdb1",
			MountPoint:  "/mnt/very-long-mount-point-name",
			FSType:      "xfs",
			Total:       2000000000000,
			Used:        1000000000000,
			Available:   1000000000000,
			UsedPercent: 50.0,
		},
	}

	longOutput := f.formatDisk(longDisk)
	t.Logf("Long mount point output:\n%s", longOutput)

	// Verify alignment
	longLines := strings.Split(longOutput, "\n")

	// Find header line
	longHeaderLen := 0
	for _, line := range longLines {
		if strings.Contains(line, "Filesystem") && strings.Contains(line, "Size") {
			longHeaderLen = len(line)
			break
		}
	}

	if longHeaderLen == 0 {
		t.Fatal("Could not find header line")
	}

	// Check all data lines have consistent length
	for i, line := range longLines {
		if len(line) == 0 {
			continue
		}
		if !strings.HasPrefix(line, "/") {
			continue
		}
		if len(line) != longHeaderLen {
			t.Errorf("Line %d length mismatch: expected=%d, got=%d, line=%q",
				i, longHeaderLen, len(line), line)
		}
	}
}

func TestFormatDisk_Alignment(t *testing.T) {
	f := NewTextFormatter()

	tests := []struct {
		name string
		disk collector.DiskInfo
	}{
		{
			name: "short mount points",
			disk: collector.DiskInfo{
				{MountPoint: "/", Total: 100000000000, Used: 50000000000, Available: 50000000000, UsedPercent: 50.0},
			},
		},
		{
			name: "long mount point",
			disk: collector.DiskInfo{
				{MountPoint: "/very/long/mount/point/that/exceeds/20/chars", Total: 100000000000, Used: 50000000000, Available: 50000000000, UsedPercent: 50.0},
			},
		},
		{
			name: "mixed mount points",
			disk: collector.DiskInfo{
				{MountPoint: "/", Total: 100000000000, Used: 50000000000, Available: 50000000000, UsedPercent: 50.0},
				{MountPoint: "/home/user/data", Total: 500000000000, Used: 200000000000, Available: 300000000000, UsedPercent: 40.0},
				{MountPoint: "/var/lib/docker/containers/abc123456789", Total: 1000000000000, Used: 600000000000, Available: 400000000000, UsedPercent: 60.0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.formatDisk(tt.disk)
			lines := strings.Split(output, "\n")

			if len(lines) < 3 {
				t.Fatalf("Expected at least 3 lines, got %d", len(lines))
			}

			// Find header line
			headerLen := 0
			for _, line := range lines {
				if strings.Contains(line, "Filesystem") && strings.Contains(line, "Size") {
					headerLen = len(line)
					break
				}
			}

			if headerLen == 0 {
				t.Fatal("Could not find header line")
			}

			// Check all data lines have consistent length
			for i, line := range lines {
				if len(line) == 0 {
					continue
				}
				if !strings.HasPrefix(line, "/") {
					continue
				}
				if len(line) != headerLen {
					t.Errorf("Data line %d length mismatch: expected=%d, got=%d, line=%q",
						i, headerLen, len(line), line)
				}
			}
		})
	}
}
