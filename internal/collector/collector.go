package collector

import (
	"context"
	"time"
)

type Collector interface {
	Collect(ctx context.Context) (*SystemInfo, error)
	Name() string
}

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
