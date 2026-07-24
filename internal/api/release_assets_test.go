package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newUploadClient(t *testing.T, transport http.RoundTripper) *Client {
	t.Helper()
	client := NewClient("test-token")
	client.httpClient = &http.Client{Transport: transport}
	return client
}

func TestGetReleaseUploadURLBuildsPathAndDecodesResponse(t *testing.T) {
	var gotMethod, gotPath, gotFileName string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotFileName = r.URL.Query().Get("file_name")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"url":"https://store.example.com/upload","headers":{"Content-Type":"application/octet-stream","X-MD5":"deadbeef"}}`))
	})

	got, err := GetReleaseUploadURL(client, "atom club", "atom/git-cli", "v1.2 rc", "release file+?.tar.gz")
	if err != nil {
		t.Fatalf("GetReleaseUploadURL: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want GET", gotMethod)
	}
	const wantPath = "/repos/atom%20club/atom%2Fgit-cli/releases/v1.2%20rc/upload_url"
	if gotPath != wantPath {
		t.Fatalf("path = %q, want %q", gotPath, wantPath)
	}
	if gotFileName != "release file+?.tar.gz" {
		t.Fatalf("file_name = %q, want %q", gotFileName, "release file+?.tar.gz")
	}
	if got.URL != "https://store.example.com/upload" {
		t.Fatalf("url = %q", got.URL)
	}
	if got.Headers["Content-Type"] != "application/octet-stream" {
		t.Fatalf("Content-Type = %q", got.Headers["Content-Type"])
	}
	if got.Headers["X-MD5"] != "deadbeef" {
		t.Fatalf("X-MD5 = %q", got.Headers["X-MD5"])
	}
}

func TestUploadReleaseAssetRejectsNilBody(t *testing.T) {
	upload := ReleaseUploadURL{URL: "https://store.example.com/upload"}
	if err := UploadReleaseAsset(context.Background(), newUploadClient(t, nil), upload, nil); err == nil {
		t.Fatal("expected error for nil body, got nil")
	} else if !strings.Contains(err.Error(), "nil") {
		t.Fatalf("error %q does not mention nil body", err)
	}
}

func TestUploadReleaseAssetRejectsNilContext(t *testing.T) {
	upload := ReleaseUploadURL{URL: "https://store.example.com/upload"}
	err := UploadReleaseAsset(nil, newUploadClient(t, nil), upload, bytes.NewReader([]byte("payload")))
	if err == nil || !strings.Contains(err.Error(), "context") {
		t.Fatalf("error = %v, want context error", err)
	}
}

type untouchedReadSeeker struct {
	readCalls int
	seekCalls int
}

func (r *untouchedReadSeeker) Read([]byte) (int, error) {
	r.readCalls++
	return 0, io.EOF
}

func (r *untouchedReadSeeker) Seek(int64, int) (int64, error) {
	r.seekCalls++
	return 0, nil
}

func TestUploadReleaseAssetRejectsUnsafeTargetBeforeReadingBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		url  string
	}{
		{name: "plaintext", url: "http://store.example.com/upload"},
		{name: "relative", url: "/upload"},
		{name: "missing host", url: "https:///upload"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var transportCalls int32
			transport := funcRoundTrip(func(*http.Request) (*http.Response, error) {
				atomic.AddInt32(&transportCalls, 1)
				return nil, errors.New("unexpected transport call")
			})
			body := &untouchedReadSeeker{}

			err := UploadReleaseAsset(
				context.Background(),
				newUploadClient(t, transport),
				ReleaseUploadURL{URL: tc.url},
				body,
			)
			if err == nil || !strings.Contains(err.Error(), "HTTPS") {
				t.Fatalf("error = %v, want HTTPS validation error", err)
			}
			if got := atomic.LoadInt32(&transportCalls); got != 0 {
				t.Fatalf("transport calls = %d, want 0", got)
			}
			if body.readCalls != 0 || body.seekCalls != 0 {
				t.Fatalf("body was accessed before URL validation: reads=%d seeks=%d", body.readCalls, body.seekCalls)
			}
		})
	}
}

func TestUploadReleaseAssetDoesNotRetryAmbiguousZeroByteUpload(t *testing.T) {
	var calls int32
	transport := funcRoundTrip(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		_ = req.Body.Close()
		return nil, errors.New("response lost")
	})

	client := newUploadClient(t, transport)
	err := UploadReleaseAsset(
		context.Background(),
		client,
		ReleaseUploadURL{URL: "https://store.example.com/upload"},
		bytes.NewReader(nil),
	)
	if err == nil || !strings.Contains(err.Error(), "remote state may be unknown") {
		t.Fatalf("error = %v, want ambiguous remote-state error", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("transport calls = %d, want 1", got)
	}
}

func TestUploadReleaseAssetStopsWhenContextExpires(t *testing.T) {
	var calls int32
	transport := funcRoundTrip(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		<-req.Context().Done()
		_ = req.Body.Close()
		return nil, req.Context().Err()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	client := newUploadClient(t, transport)
	err := UploadReleaseAsset(
		ctx,
		client,
		ReleaseUploadURL{URL: "https://store.example.com/upload"},
		bytes.NewReader([]byte("payload")),
	)
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("error = %v, want cancellation error", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("transport calls = %d, want 1", got)
	}
}

type funcRoundTrip func(*http.Request) (*http.Response, error)

func (f funcRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestUploadReleaseAssetSuccess(t *testing.T) {
	upload := ReleaseUploadURL{
		URL: "https://store.example.com/upload?policy=abc",
		Headers: map[string]string{
			"Content-Type": "application/gzip",
			"X-Test":       "ok",
		},
	}
	wantBody := []byte("release-binary-payload")

	var gotReq *http.Request
	var gotBody []byte
	transport := funcRoundTrip(func(req *http.Request) (*http.Response, error) {
		gotReq = req
		if req.Body != nil {
			b, err := io.ReadAll(req.Body)
			_ = req.Body.Close()
			if err != nil {
				return nil, err
			}
			gotBody = b
		}
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusOK)
		return rec.Result(), nil
	})

	client := newUploadClient(t, transport)
	if err := UploadReleaseAsset(context.Background(), client, upload, bytes.NewReader(wantBody)); err != nil {
		t.Fatalf("UploadReleaseAsset: %v", err)
	}
	if gotReq == nil {
		t.Fatal("transport was not called")
	}
	if gotReq.Method != http.MethodPut {
		t.Fatalf("method = %q, want PUT", gotReq.Method)
	}
	if gotReq.URL.String() != upload.URL {
		t.Fatalf("url = %q, want %q", gotReq.URL.String(), upload.URL)
	}
	if gotReq.Header.Get("Content-Type") != "application/gzip" {
		t.Fatalf("Content-Type = %q", gotReq.Header.Get("Content-Type"))
	}
	if gotReq.Header.Get("X-Test") != "ok" {
		t.Fatalf("X-Test = %q", gotReq.Header.Get("X-Test"))
	}
	if gotReq.Header.Get("Authorization") != "" {
		t.Fatalf("Authorization leaked to external host: %q", gotReq.Header.Get("Authorization"))
	}
	if gotReq.ContentLength != int64(len(wantBody)) {
		t.Fatalf("Content-Length = %d, want %d", gotReq.ContentLength, len(wantBody))
	}
	if !bytes.Equal(gotBody, wantBody) {
		t.Fatalf("body = %q, want %q", gotBody, wantBody)
	}
}

func TestUploadReleaseAssetDoesNotUseWholeRequestTimeout(t *testing.T) {
	wantBody := []byte("slow-upload-payload")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read upload body: %v", err)
			return
		}
		if !bytes.Equal(got, wantBody) {
			t.Errorf("body = %q, want %q", got, wantBody)
		}
		time.Sleep(40 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = 10 * time.Millisecond
	client := NewClientWithHTTPClient("token", httpClient)
	err := UploadReleaseAsset(context.Background(), client, ReleaseUploadURL{URL: server.URL}, bytes.NewReader(wantBody))
	if err != nil {
		t.Fatalf("UploadReleaseAsset: %v", err)
	}
}

func TestUploadReleaseAssetRejectsRedirects(t *testing.T) {
	for _, statusCode := range []int{
		http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect,
	} {
		t.Run(fmt.Sprintf("%d_%s", statusCode, http.StatusText(statusCode)), func(t *testing.T) {
			var targetCalls int32
			target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&targetCalls, 1)
				w.WriteHeader(http.StatusOK)
			}))
			defer target.Close()

			redirect := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, target.URL, statusCode)
			}))
			defer redirect.Close()

			client := NewClientWithHTTPClient("token", redirect.Client())
			err := UploadReleaseAsset(
				context.Background(),
				client,
				ReleaseUploadURL{
					URL:     redirect.URL,
					Headers: map[string]string{"X-Upload-Token": "signed-value"},
				},
				bytes.NewReader([]byte("redirect-payload")),
			)
			if err == nil {
				t.Fatal("expected redirect to fail, got nil")
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("%d", statusCode)) {
				t.Fatalf("error %q does not contain status %d", err.Error(), statusCode)
			}
			if got := atomic.LoadInt32(&targetCalls); got != 0 {
				t.Fatalf("redirect target calls = %d, want 0", got)
			}
		})
	}
}

func TestUploadReleaseAssetRetriesOneNetworkError(t *testing.T) {
	upload := ReleaseUploadURL{
		URL:     "https://store.example.com/upload?policy=abc",
		Headers: map[string]string{"Content-Type": "application/gzip"},
	}
	wantBody := []byte("retry-payload")

	var calls int32
	var body []byte
	transport := funcRoundTrip(func(req *http.Request) (*http.Response, error) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			_ = req.Body.Close()
			return nil, errors.New("connection reset by peer")
		}
		var err error
		body, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusOK)
		return rec.Result(), nil
	})

	client := newUploadClient(t, transport)
	if err := UploadReleaseAsset(context.Background(), client, upload, bytes.NewReader(wantBody)); err != nil {
		t.Fatalf("UploadReleaseAsset: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("transport calls = %d, want 2", got)
	}
	if !bytes.Equal(body, wantBody) {
		t.Fatalf("retried body = %q, want %q", body, wantBody)
	}
}

func TestUploadReleaseAssetWaitsForRequestBodyCloseBeforeRetry(t *testing.T) {
	upload := ReleaseUploadURL{URL: "https://store.example.com/upload"}
	wantBody := []byte("async-close-retry-payload")

	var calls int32
	firstBodyClosed := make(chan struct{})
	transport := funcRoundTrip(func(req *http.Request) (*http.Response, error) {
		switch n := atomic.AddInt32(&calls, 1); n {
		case 1:
			go func(body io.ReadCloser) {
				time.Sleep(300 * time.Millisecond)
				_ = body.Close()
				close(firstBodyClosed)
			}(req.Body)
			return nil, errors.New("temporary connection failure")
		case 2:
			select {
			case <-firstBodyClosed:
			default:
				return nil, errors.New("second request started before first body closed")
			}
			gotBody, err := io.ReadAll(req.Body)
			_ = req.Body.Close()
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(gotBody, wantBody) {
				return nil, fmt.Errorf("retried body = %q, want %q", gotBody, wantBody)
			}
			recorder := httptest.NewRecorder()
			recorder.WriteHeader(http.StatusOK)
			return recorder.Result(), nil
		default:
			return nil, fmt.Errorf("unexpected transport call %d", n)
		}
	})

	client := newUploadClient(t, transport)
	if err := UploadReleaseAsset(context.Background(), client, upload, bytes.NewReader(wantBody)); err != nil {
		t.Fatalf("UploadReleaseAsset: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("transport calls = %d, want 2", got)
	}
}

// Regression test: transport Close must not close the underlying upload file.
func TestUploadReleaseAssetRetriesRealFile(t *testing.T) {
	wantBody := []byte("real-file-payload-for-retry")

	dir := t.TempDir()
	path := filepath.Join(dir, "asset.bin")
	if err := os.WriteFile(path, wantBody, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	upload := ReleaseUploadURL{
		URL:     "https://store.example.com/upload?policy=abc",
		Headers: map[string]string{"Content-Type": "application/octet-stream"},
	}

	var calls int32
	var body []byte
	transport := funcRoundTrip(func(req *http.Request) (*http.Response, error) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			_ = req.Body.Close()
			return nil, errors.New("connection reset by peer")
		}
		var readErr error
		body, readErr = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusOK)
		return rec.Result(), nil
	})

	client := newUploadClient(t, transport)
	if err := UploadReleaseAsset(context.Background(), client, upload, file); err != nil {
		t.Fatalf("UploadReleaseAsset: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("transport calls = %d, want 2", got)
	}
	if !bytes.Equal(body, wantBody) {
		t.Fatalf("retried body = %q, want %q", body, wantBody)
	}
}

func TestUploadReleaseAssetDoesNotRetryHTTPError(t *testing.T) {
	upload := ReleaseUploadURL{
		URL:     "https://store.example.com/upload?policy=abc",
		Headers: map[string]string{"Content-Type": "application/gzip"},
	}
	wantBody := []byte("http-error-payload")

	var calls int32
	transport := funcRoundTrip(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		if req.Body != nil {
			_, _ = io.ReadAll(req.Body)
			_ = req.Body.Close()
		}
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusInternalServerError)
		_, _ = rec.WriteString("internal server error")
		return rec.Result(), nil
	})

	client := newUploadClient(t, transport)
	err := UploadReleaseAsset(context.Background(), client, upload, bytes.NewReader(wantBody))
	if err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("error %q does not contain status 500", err.Error())
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("transport calls = %d, want 1 (no HTTP retry)", got)
	}
}

func TestDeleteReleaseAttachment(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	})

	if err := DeleteReleaseAttachment(client, "atom club", "atom/git-cli", "v1.2 rc", 42); err != nil {
		t.Fatalf("DeleteReleaseAttachment: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method = %q, want DELETE", gotMethod)
	}
	const wantPath = "/repos/atom%20club/atom%2Fgit-cli/releases/v1.2%20rc/attach_files/42"
	if gotPath != wantPath {
		t.Fatalf("path = %q, want %q", gotPath, wantPath)
	}
}

func TestDeleteReleaseAttachmentDoesNotRetryTransportError(t *testing.T) {
	var calls int32
	client := NewClientWithHTTPClient("token", &http.Client{Transport: funcRoundTrip(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return nil, errors.New("response lost")
	})})

	err := DeleteReleaseAttachment(client, "owner", "repo", "v1.0", 42)
	if err == nil || !strings.Contains(err.Error(), "response lost") {
		t.Fatalf("error = %v, want response-lost context", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("transport calls = %d, want 1", got)
	}
}

func TestDeleteReleaseAttachmentRejectsNonPositiveID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.EscapedPath())
	})
	for _, id := range []int64{0, -1, -42} {
		if err := DeleteReleaseAttachment(client, "owner", "repo", "v1.0", id); err == nil {
			t.Fatalf("id %d: expected error, got nil", id)
		}
	}
}

type closeTracker struct {
	rc     io.ReadCloser
	closed int32
}

func (c *closeTracker) Read(p []byte) (int, error) { return c.rc.Read(p) }
func (c *closeTracker) Close() error {
	atomic.AddInt32(&c.closed, 1)
	return c.rc.Close()
}

func TestDownloadReleaseAttachmentSuccess(t *testing.T) {
	want := []byte{0x00, 0x01, 'b', 'i', 'n', 0xFF, 0xFE, 'x', 0x0A, 0x00}
	var gotMethod, gotAccept, gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAccept = r.Header.Get("Accept")
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write(want)
	})

	rc, err := DownloadReleaseAttachment(context.Background(), client, "atom club", "atom/git-cli", "v1.2 rc", "release file.tar.gz")
	if err != nil {
		t.Fatalf("DownloadReleaseAttachment: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want GET", gotMethod)
	}
	if gotAccept != "*/*" {
		t.Fatalf("Accept = %q, want %q", gotAccept, "*/*")
	}
	const wantPath = "/repos/atom%20club/atom%2Fgit-cli/releases/v1.2%20rc/attach_files/release%20file.tar.gz/download"
	if gotPath != wantPath {
		t.Fatalf("path = %q, want %q", gotPath, wantPath)
	}
}

func TestDownloadReleaseAttachmentDoesNotUseMetadataTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("first-"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		time.Sleep(40 * time.Millisecond)
		_, _ = w.Write([]byte("second"))
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = 10 * time.Millisecond
	client := NewClientWithBaseURL("token", server.URL, httpClient)

	body, err := DownloadReleaseAttachment(context.Background(), client, "owner", "repo", "v1.0", "asset.bin")
	if err != nil {
		t.Fatalf("DownloadReleaseAttachment: %v", err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read download: %v", err)
	}
	if string(got) != "first-second" {
		t.Fatalf("body = %q, want %q", got, "first-second")
	}
}

func TestDownloadReleaseAttachmentRejectsNonOKAndClosesBody(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusPartialContent, http.StatusNotFound} {
		t.Run(fmt.Sprintf("status_%d", status), func(t *testing.T) {
			tracker := &closeTracker{rc: io.NopCloser(strings.NewReader("response body"))}
			transport := funcRoundTrip(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: status,
					Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
					Header:     make(http.Header),
					Body:       tracker,
				}, nil
			})
			client := newUploadClient(t, transport)

			rc, err := DownloadReleaseAttachment(context.Background(), client, "owner", "repo", "v1.0", "asset.bin")
			if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%d", status)) {
				t.Fatalf("error = %v, want status %d", err, status)
			}
			if rc != nil {
				t.Fatalf("reader = %v, want nil", rc)
			}
			if got := atomic.LoadInt32(&tracker.closed); got != 1 {
				t.Fatalf("body closes = %d, want 1", got)
			}
		})
	}
}

func TestDownloadReleaseAttachmentStopsWhenContextExpires(t *testing.T) {
	var calls int32
	transport := funcRoundTrip(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	client := newUploadClient(t, transport)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	body, err := DownloadReleaseAttachment(ctx, client, "owner", "repo", "v1.0", "asset.bin")
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
	if body != nil {
		t.Fatalf("body = %v, want nil", body)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("transport calls = %d, want 1", got)
	}
}

func TestUploadReleaseAssetExhaustsRetryOnRepeatedInterruption(t *testing.T) {
	upload := ReleaseUploadURL{
		URL:     "https://store.example.com/upload?policy=abc",
		Headers: map[string]string{"Content-Type": "application/octet-stream"},
	}
	wantBody := []byte("repeated-interruption-payload")

	var calls int32
	var body []byte
	transport := funcRoundTrip(func(req *http.Request) (*http.Response, error) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			_ = req.Body.Close()
			return nil, errors.New("first interruption: connection reset by peer")
		}
		if n != 2 {
			return nil, fmt.Errorf("unexpected transport call %d", n)
		}
		var readErr error
		body, readErr = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		return nil, errors.New("second interruption: response lost")
	})

	client := newUploadClient(t, transport)
	err := UploadReleaseAsset(context.Background(), client, upload, bytes.NewReader(wantBody))
	if err == nil {
		t.Fatal("expected error after exhausting retry, got nil")
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("transport calls = %d, want 2", got)
	}
	if !bytes.Equal(body, wantBody) {
		t.Fatalf("second-attempt body = %q, want %q", body, wantBody)
	}
	if !strings.Contains(err.Error(), "remote state may be unknown") {
		t.Fatalf("error %q does not explain the ambiguous remote state", err.Error())
	}
	if !strings.Contains(err.Error(), "second interruption") {
		t.Fatalf("error %q does not contain the second interruption context", err.Error())
	}
}
