package retry

import (
	"context"
	"math/rand"
	"time"
)

var (
	DefaultSleep         = 1 * time.Second
	ReadableDefaultSleep = "1s"
)

// Func is the function to be executed and eventually retried.
type Func func() error

type Condition bool

const (
	Break    Condition = true
	Continue Condition = false
)

// ConditionFunc returns additional flag determine whether to break retry or not.
type ConditionFunc func() (Condition, error)

const maxBackoff = 30 * time.Second

// Do runs the passed function until the number of retries is reached.
// The sleep value is slightly modified on every retry (exponential backoff)
// to prevent thundering herd effects.
// If sleep is zero, it defaults to 1s.
func Do(fn Func, retries int, sleep time.Duration) error {
	if retries <= 0 {
		retries = 1
	}
	if sleep == 0 {
		sleep = DefaultSleep
	}

	var err error
	for i := 0; i < retries; i++ {
		if err = fn(); err == nil {
			return nil
		}

		// Last attempt: do not sleep.
		if i == retries-1 {
			break
		}

		// Preventing thundering herd problem.
		sleep += time.Duration(rand.Int63n(int64(sleep))) / 2
		if sleep > maxBackoff {
			sleep = maxBackoff
		}
		time.Sleep(sleep)
		sleep *= 2
	}

	return err
}

// DoWithContext runs fn with retries and respects context cancellation.
// It uses the same backoff + jitter strategy as Do.
func DoWithContext(ctx context.Context, fn Func, retries int, sleep time.Duration) error {
	if retries <= 0 {
		retries = 1
	}
	if sleep == 0 {
		sleep = DefaultSleep
	}

	var err error
	for i := 0; i < retries; i++ {
		if err = fn(); err == nil {
			return nil
		}

		if i == retries-1 {
			break
		}

		sleep += time.Duration(rand.Int63n(int64(sleep))) / 2
		if sleep > maxBackoff {
			sleep = maxBackoff
		}

		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		sleep *= 2
	}

	return err
}

// DoCondition runs fn until Break is returned or retries are exhausted.
func DoCondition(fn ConditionFunc, retries int, sleep time.Duration) error {
	if retries <= 0 {
		retries = 1
	}
	if sleep == 0 {
		sleep = DefaultSleep
	}

	var cond Condition
	var err error
	for i := 0; i < retries; i++ {
		cond, err = fn()
		if cond == Break {
			return err
		}

		if i == retries-1 {
			break
		}

		sleep += time.Duration(rand.Int63n(int64(sleep))) / 2
		if sleep > maxBackoff {
			sleep = maxBackoff
		}
		time.Sleep(sleep)
		sleep *= 2
	}

	return err
}
