package formatter

import (
	"encoding/json"

	"github.com/SisyphusSQ/summary-sys/internal/collector"
)

type JSONFormatter struct {
	Pretty bool
}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{Pretty: true}
}

func (f *JSONFormatter) Name() string        { return "json" }
func (f *JSONFormatter) ContentType() string { return "application/json" }

func (f *JSONFormatter) Format(info *collector.SystemInfo) (string, error) {
	if f.Pretty {
		b, err := json.MarshalIndent(info, "", "  ")
		return string(b), err
	}
	b, err := json.Marshal(info)
	return string(b), err
}
