package api

import (
	"context"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultRateLimitRetryBudget   = 30 * time.Second
	defaultRateLimitMaxAttempts   = 3
	defaultRateLimitFallbackDelay = 250 * time.Millisecond
	maxRateLimitFallbackDelay     = 2 * time.Second
	networkRetryDelay             = 200 * time.Millisecond
)

type rateLimitRetryPolicy struct {
	budget      time.Duration
	maxAttempts int
	now         func() time.Time
	wait        func(context.Context, time.Duration) error
	jitter      func(time.Duration) time.Duration
}

type cancelOnCloseBody struct {
	io.ReadCloser
	cancel context.CancelFunc
	once   sync.Once
}

func (b *cancelOnCloseBody) Close() error {
	err := b.ReadCloser.Close()
	b.once.Do(b.cancel)
	return err
}

func finishRateLimitedRequest(resp *http.Response, err error, cancel context.CancelFunc) (*http.Response, error) {
	if cancel == nil {
		return resp, err
	}
	if err == nil && resp != nil && resp.Body != nil {
		resp.Body = &cancelOnCloseBody{ReadCloser: resp.Body, cancel: cancel}
		return resp, nil
	}
	cancel()
	return resp, err
}

func defaultRateLimitPolicy() rateLimitRetryPolicy {
	return rateLimitRetryPolicy{
		budget:      defaultRateLimitRetryBudget,
		maxAttempts: defaultRateLimitMaxAttempts,
		now:         time.Now,
		wait:        waitForRetry,
		jitter: func(limit time.Duration) time.Duration {
			if limit <= 0 {
				return 0
			}
			return time.Duration(rand.Int64N(int64(limit) + 1))
		},
	}
}

func (p rateLimitRetryPolicy) withDefaults() rateLimitRetryPolicy {
	defaults := defaultRateLimitPolicy()
	if p.budget <= 0 {
		p.budget = defaults.budget
	}
	if p.maxAttempts <= 0 {
		p.maxAttempts = defaults.maxAttempts
	}
	if p.now == nil {
		p.now = defaults.now
	}
	if p.wait == nil {
		p.wait = defaults.wait
	}
	if p.jitter == nil {
		p.jitter = defaults.jitter
	}
	return p
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseUint(value, 10, 64); err == nil {
		maxSeconds := uint64(math.MaxInt64 / int64(time.Second))
		if seconds > maxSeconds {
			return time.Duration(math.MaxInt64), true
		}
		return time.Duration(seconds) * time.Second, true
	}

	when, err := http.ParseTime(value)
	if err != nil {
		return 0, false
	}
	delay := when.Sub(now)
	if delay <= 0 {
		return 0, false
	}
	return delay, true
}

func (p rateLimitRetryPolicy) fallbackDelay(retry int) time.Duration {
	delay := defaultRateLimitFallbackDelay
	for range retry {
		if delay >= maxRateLimitFallbackDelay/2 {
			delay = maxRateLimitFallbackDelay
			break
		}
		delay *= 2
	}
	jitterLimit := min(delay/2, maxRateLimitFallbackDelay-delay)
	return min(delay+p.jitter(jitterLimit), maxRateLimitFallbackDelay)
}

func closeRetryResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, MaxErrorExcerptBytes))
	_ = resp.Body.Close()
}
