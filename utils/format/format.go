package format

import (
	"fmt"
	"time"
)

// FormatBytes formats bytes to human-readable string (KB, MB, GB, etc.)
func FormatBytes(b uint64) string {
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

// FormatUptime formats duration to human-readable uptime string
func FormatUptime(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Minute
	m := d / time.Minute
	if h > 24 {
		days := h / 24
		h = h % 24
		return fmt.Sprintf("%dd %dh %dm", days, h, m)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}
