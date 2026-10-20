package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestUploadTransportFailurePreservesContextCause(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "deadline"}[deadline], func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			wantContext := context.Canceled
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), 100*time.Millisecond)
				wantContext = context.DeadlineExceeded
			}
			defer cancel()
			cause := errors.New("transport stopped")
			calls := 0
			client := newUploadClient(t, funcRoundTrip(func(req *http.Request) (*http.Response, error) {
				calls++
				if deadline {
					<-ctx.Done()
				} else {
					cancel()
				}
				_ = req.Body.Close()
				return nil, cause
			}))
			err := UploadReleaseAsset(ctx, client, ReleaseUploadURL{URL: "https://store.example/upload"}, bytes.NewReader([]byte("payload")))
			if !errors.Is(err, cause) || !errors.Is(err, wantContext) {
				t.Fatalf("lost transport or context cause: %v", err)
			}
			if _, ok := errors.AsType[*url.Error](err); !ok || calls != 1 {
				t.Fatalf("transport wrapper / calls = %v / %d", err, calls)
			}
		})
	}
}

func TestUploadBodyCloseTimeoutPreservesTransportCause(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cause := errors.New("transport failed without closing body")
	calls := 0
	client := newUploadClient(t, funcRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		// Model a transport that violates the body-close contract. Retain the
		// request for cleanup, but never allow a retry to reuse the open body.
		t.Cleanup(func() { _ = req.Body.Close() })
		cancel()
		return nil, cause
	}))
	err := UploadReleaseAsset(ctx, client, ReleaseUploadURL{URL: "https://store.example/upload"}, bytes.NewReader([]byte("payload")))
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("lost cause or retried open body: %v (calls %d)", err, calls)
	}
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok || len(joined.Unwrap()) != 2 || !errors.Is(joined.Unwrap()[0], cause) ||
		!strings.Contains(joined.Unwrap()[1].Error(), "timed out") {
		t.Fatalf("transport and close roles = %v", err)
	}
}

type cancelOnRetryReader struct {
	*bytes.Reader
	cancel context.CancelFunc
	seeks  int
}

func (r *cancelOnRetryReader) Seek(offset int64, whence int) (int64, error) {
	r.seeks++
	if r.seeks == 3 {
		r.cancel()
	}
	return r.Reader.Seek(offset, whence)
}

func TestUploadCanceledBeforeRetryPreservesTransportCause(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cause := errors.New("first attempt failed")
	calls := 0
	client := newUploadClient(t, funcRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		_ = req.Body.Close()
		return nil, cause
	}))
	body := &cancelOnRetryReader{Reader: bytes.NewReader([]byte("payload")), cancel: cancel}
	err := UploadReleaseAsset(ctx, client, ReleaseUploadURL{URL: "https://store.example/upload"}, body)
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("retry cancellation lost cause: %v (calls %d)", err, calls)
	}
}
