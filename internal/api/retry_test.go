package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

type trackingRetryBody struct {
	remaining int
	read      int
	closed    bool
}

func (b *trackingRetryBody) Read(p []byte) (int, error) {
	if b.remaining == 0 {
		return 0, io.EOF
	}
	n := min(len(p), b.remaining)
	for i := range n {
		p[i] = 'x'
	}
	b.remaining -= n
	b.read += n
	return n, nil
}

func (b *trackingRetryBody) Close() error {
	b.closed = true
	return nil
}

func TestRateLimitRetryAfterVariants(t *testing.T) {
	fixedNow := time.Date(2026, time.September, 27, 1, 2, 3, 0, time.UTC)
	tests := []struct {
		name       string
		retryAfter string
		wantDelay  time.Duration
	}{
		{name: "seconds", retryAfter: "2", wantDelay: 2 * time.Second},
		{name: "HTTP date", retryAfter: fixedNow.Add(5 * time.Second).Format(http.TimeFormat), wantDelay: 5 * time.Second},
		{name: "missing", wantDelay: defaultRateLimitFallbackDelay},
		{name: "invalid", retryAfter: "later", wantDelay: defaultRateLimitFallbackDelay},
		{name: "negative seconds", retryAfter: "-1", wantDelay: defaultRateLimitFallbackDelay},
		{name: "expired date", retryAfter: fixedNow.Add(-time.Second).Format(http.TimeFormat), wantDelay: defaultRateLimitFallbackDelay},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			var body *trackingRetryBody
			client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						body = &trackingRetryBody{remaining: MaxErrorExcerptBytes * 2}
						resp := runRawResponse(req, http.StatusTooManyRequests, "")
						resp.Body = body
						resp.Header.Set("Retry-After", tt.retryAfter)
						return resp, nil
					}
					return runRawResponse(req, http.StatusOK, `{}`), nil
				}),
			})
			var waited []time.Duration
			client.rateLimitPolicy = rateLimitRetryPolicy{
				budget:      10 * time.Second,
				maxAttempts: 3,
				now:         func() time.Time { return fixedNow },
				wait: func(_ context.Context, delay time.Duration) error {
					waited = append(waited, delay)
					return nil
				},
				jitter: func(time.Duration) time.Duration { return 0 },
			}
			var warnings bytes.Buffer
			client = client.WithRetryWriter(&warnings)

			var result map[string]any
			if err := client.Get("/resource", &result); err != nil {
				t.Fatal(err)
			}
			if calls != 2 || len(waited) != 1 || waited[0] != tt.wantDelay {
				t.Fatalf("calls = %d, waits = %v, want one wait of %s", calls, waited, tt.wantDelay)
			}
			if body == nil || !body.closed || body.read > MaxErrorExcerptBytes {
				t.Fatalf("retry body closed = %v, bytes read = %d", body != nil && body.closed, body.read)
			}
			if !strings.Contains(warnings.String(), "Rate limited") || !strings.Contains(warnings.String(), "attempt 2/3") {
				t.Fatalf("warning = %q", warnings.String())
			}
		})
	}
}

func TestRateLimitRetryStopsAtBudgetAndAttemptBounds(t *testing.T) {
	t.Run("parent context deadline limits budget", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		calls := 0
		client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				resp := runRawResponse(req, http.StatusTooManyRequests, `{"message":"slow down"}`)
				resp.Header.Set("Retry-After", "3")
				return resp, nil
			}),
		}).WithContext(ctx)
		client.rateLimitPolicy = rateLimitRetryPolicy{
			budget:      10 * time.Second,
			maxAttempts: 3,
			now:         time.Now,
			wait:        func(context.Context, time.Duration) error { t.Fatal("unexpected wait"); return nil },
			jitter:      func(time.Duration) time.Duration { return 0 },
		}

		err := client.Get("/resource", &map[string]any{})
		var httpErr *HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusTooManyRequests || httpErr.RetryAfter != "3" || calls != 1 {
			t.Fatalf("error = %v, calls = %d, want one terminal 429 preserving Retry-After", err, calls)
		}
	})

	t.Run("consecutive responses exhaust budget", func(t *testing.T) {
		calls := 0
		waits := 0
		now := time.Date(2026, time.September, 27, 1, 2, 3, 0, time.UTC)
		client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				resp := runRawResponse(req, http.StatusTooManyRequests, `{"message":"slow down"}`)
				resp.Header.Set("Retry-After", "1")
				return resp, nil
			}),
		})
		client.rateLimitPolicy = rateLimitRetryPolicy{
			budget:      1500 * time.Millisecond,
			maxAttempts: 3,
			now:         func() time.Time { return now },
			wait: func(_ context.Context, delay time.Duration) error {
				waits++
				now = now.Add(delay)
				return nil
			},
			jitter: func(time.Duration) time.Duration { return 0 },
		}

		err := client.Get("/resource", &map[string]any{})
		if !IsHTTPStatus(err, http.StatusTooManyRequests) || calls != 2 || waits != 1 {
			t.Fatalf("error = %v, calls = %d, waits = %d", err, calls, waits)
		}
	})

	for _, retryAfter := range []string{
		strconv.FormatUint(^uint64(0), 10),
		"18446744073709551616",
	} {
		t.Run("server delay exceeds remaining budget "+retryAfter, func(t *testing.T) {
			calls := 0
			client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					resp := runRawResponse(req, http.StatusTooManyRequests, `{"message":"slow down"}`)
					resp.Header.Set("Retry-After", retryAfter)
					return resp, nil
				}),
			})
			client.rateLimitPolicy = rateLimitRetryPolicy{
				budget:      time.Second,
				maxAttempts: 3,
				now:         time.Now,
				wait:        func(context.Context, time.Duration) error { t.Fatal("unexpected wait"); return nil },
				jitter:      func(time.Duration) time.Duration { return 0 },
			}

			err := client.Get("/resource", &map[string]any{})
			if !IsHTTPStatus(err, http.StatusTooManyRequests) || calls != 1 {
				t.Fatalf("error = %v, calls = %d, want one terminal 429", err, calls)
			}
		})
	}

	t.Run("consecutive responses exhaust attempts", func(t *testing.T) {
		calls := 0
		client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				resp := runRawResponse(req, http.StatusTooManyRequests, `{"message":"still limited"}`)
				resp.Header.Set("Retry-After", "0")
				return resp, nil
			}),
		})
		client.rateLimitPolicy.wait = func(context.Context, time.Duration) error { return nil }
		client.rateLimitPolicy.jitter = func(time.Duration) time.Duration { return 0 }

		err := client.Get("/resource", &map[string]any{})
		if !IsHTTPStatus(err, http.StatusTooManyRequests) || calls != defaultRateLimitMaxAttempts {
			t.Fatalf("error = %v, calls = %d", err, calls)
		}
	})
}

func TestRateLimitRetryHonorsCancellationAndWholeRequestBudget(t *testing.T) {
	t.Run("cancellation while waiting", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		waitStarted := make(chan struct{})
		client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				resp := runRawResponse(req, http.StatusTooManyRequests, "limited")
				resp.Header.Set("Retry-After", "1")
				return resp, nil
			}),
		}).WithContext(ctx)
		client.rateLimitPolicy.wait = func(ctx context.Context, _ time.Duration) error {
			close(waitStarted)
			<-ctx.Done()
			return ctx.Err()
		}

		result := make(chan error, 1)
		go func() { result <- client.Get("/resource", &map[string]any{}) }()
		<-waitStarted
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context canceled", err)
		}
	})

	t.Run("budget includes request time", func(t *testing.T) {
		calls := 0
		client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				<-req.Context().Done()
				return nil, req.Context().Err()
			}),
		})
		client.rateLimitPolicy.budget = 20 * time.Millisecond

		started := time.Now()
		err := client.Get("/resource", &map[string]any{})
		if !errors.Is(err, context.DeadlineExceeded) || calls != 1 || time.Since(started) > time.Second {
			t.Fatalf("error = %v, calls = %d, elapsed = %s", err, calls, time.Since(started))
		}
	})
}

func TestRateLimitRetryDoesNotReplayDisabledWritesOrStreams(t *testing.T) {
	for _, tt := range []struct {
		name string
		call func(*Client) error
	}{
		{
			name: "explicitly disabled GET",
			call: func(client *Client) error {
				return client.doJSONRequest(http.MethodGet, "/resource", nil, "", "application/json", RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}, nil)
			},
		},
		{name: "PUT", call: func(client *Client) error { return client.Put("/resource", map[string]string{"value": "x"}, nil) }},
		{
			name: "streaming GET",
			call: func(client *Client) error {
				resp, err := client.DoRequestRawStreamingWithAccept(http.MethodGet, "/resource", "*/*")
				if resp != nil {
					resp.Body.Close()
				}
				if err != nil {
					return err
				}
				return fmt.Errorf("status %d", resp.StatusCode)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					resp := runRawResponse(req, http.StatusTooManyRequests, "limited")
					resp.Header.Set("Retry-After", "0")
					return resp, nil
				}),
			})
			if err := tt.call(client); err == nil {
				t.Fatal("expected terminal 429 error")
			}
			if calls != 1 {
				t.Fatalf("calls = %d, want 1", calls)
			}
		})
	}
}
