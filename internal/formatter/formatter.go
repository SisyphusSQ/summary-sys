package formatter

import "github.com/SisyphusSQ/summary-sys/internal/collector"

type Formatter interface {
	Format(info *collector.SystemInfo) (string, error)
	Name() string
	ContentType() string
}

func NewFormatter(format string) (Formatter, error) {
	switch format {
	case "text":
		return NewTextFormatter(), nil
	case "json":
		return NewJSONFormatter(), nil
	default:
		return nil, nil
	}
}
