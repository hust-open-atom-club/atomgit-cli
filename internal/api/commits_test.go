package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type commitRoundTripFunc func(*http.Request) (*http.Response, error)

func (f commitRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func commitTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClientWithBaseURL("token", server.URL+APIVersion, server.Client())
}

func TestCompareCommitsEscapesRefs(t *testing.T) {
	client := commitTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("method = %s", req.Method)
		}
		want := "/api/v5/repos/alice/demo/compare/release%2F1.0...feature%2Fnew%20ui"
		if req.URL.EscapedPath() != want {
			t.Fatalf("path = %q, want %q", req.URL.EscapedPath(), want)
		}
		if req.Header.Get("Accept") != "application/json" {
			t.Fatalf("Accept = %q", req.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"base_commit":{"sha":"base-sha"},"merge_base_commit":{"sha":"merge-sha"},"commits":null,"files":null,"truncated":false}`))
	})

	comparison, err := CompareCommits(context.Background(), client, "alice", "demo", "release/1.0", "feature/new ui")
	if err != nil {
		t.Fatal(err)
	}
	if comparison.Commits == nil || comparison.Files == nil {
		t.Fatalf("nil slices: %#v", comparison)
	}
}

func TestCompareCommitsReturnsAPIError(t *testing.T) {
	client := commitTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"compare failed"}`))
	})

	_, err := CompareCommits(context.Background(), client, "alice", "demo", "main", "feature")
	if err == nil || !strings.Contains(err.Error(), "API error: 500 Internal Server Error") || !strings.Contains(err.Error(), "compare failed") {
		t.Fatalf("error = %v", err)
	}
}

func TestCompareCommitsStopsWhenContextExpires(t *testing.T) {
	requestStarted := make(chan struct{})
	client := NewClientWithHTTPClient("token", &http.Client{Transport: commitRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		close(requestStarted)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := CompareCommits(ctx, client, "alice", "demo", "main", "feature")
		done <- err
	}()
	<-requestStarted
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestGetCommitText(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		status     int
		body       string
		wantBody   string
		wantError  string
		wantAccept string
	}{
		{name: "diff", format: "diff", status: http.StatusOK, body: "diff --git a/a b/a\n", wantBody: "diff --git a/a b/a\n", wantAccept: "text/plain, */*"},
		{name: "patch", format: "patch", status: http.StatusOK, body: "From abc Mon Sep 17 00:00:00 2001\n", wantBody: "From abc Mon Sep 17 00:00:00 2001\n", wantAccept: "text/plain, */*"},
		{name: "not found", format: "diff", status: http.StatusNotFound, body: `{"message":"commit not found"}`, wantError: "API error: 404 Not Found"},
		{name: "server error", format: "patch", status: http.StatusInternalServerError, body: "backend unavailable", wantError: "API error: 500 Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := commitTestClient(t, func(w http.ResponseWriter, req *http.Request) {
				wantPath := "/api/v5/repos/alice/demo/commit/feature%2Fone/" + tt.format
				if req.URL.EscapedPath() != wantPath {
					t.Fatalf("path = %q, want %q", req.URL.EscapedPath(), wantPath)
				}
				if tt.wantAccept != "" && req.Header.Get("Accept") != tt.wantAccept {
					t.Fatalf("Accept = %q", req.Header.Get("Accept"))
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})

			body, err := GetCommitText(context.Background(), client, "alice", "demo", "feature/one", tt.format)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want %q", err, tt.wantError)
				}
				if body != nil {
					t.Fatal("body is non-nil on error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer body.Close()
			got, err := io.ReadAll(body)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestGetCommitTextRejectsUnsupportedFormat(t *testing.T) {
	_, err := GetCommitText(context.Background(), NewClient("token"), "alice", "demo", "abc", "raw")
	if err == nil || !strings.Contains(err.Error(), "unsupported commit text format") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetCommitTextWrapsNetworkError(t *testing.T) {
	want := errors.New("network down")
	client := NewClientWithHTTPClient("token", &http.Client{Transport: commitRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, want
	})})

	_, err := GetCommitText(context.Background(), client, "alice", "demo", "abc", "diff")
	if !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}

func TestGetCommitTextDoesNotUseWholeRequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		time.Sleep(30 * time.Millisecond)
		_, _ = w.Write([]byte("diff"))
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = 10 * time.Millisecond
	client := NewClientWithBaseURL("token", server.URL, httpClient)
	body, err := GetCommitText(context.Background(), client, "alice", "demo", "abc", "diff")
	if err != nil {
		t.Fatalf("GetCommitText: %v", err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "diff" {
		t.Fatalf("body = %q", got)
	}
}

func TestCommitAPIsRejectNilContext(t *testing.T) {
	client := NewClient("token")
	if _, err := CompareCommits(nil, client, "alice", "demo", "main", "feature"); err == nil || !strings.Contains(err.Error(), "context is nil") {
		t.Fatalf("CompareCommits error = %v", err)
	}
	if _, err := GetCommitText(nil, client, "alice", "demo", "abc", "diff"); err == nil || !strings.Contains(err.Error(), "context is nil") {
		t.Fatalf("GetCommitText error = %v", err)
	}
}

func TestCommitAPIsRejectNilClient(t *testing.T) {
	if _, err := CompareCommits(context.Background(), nil, "alice", "demo", "main", "feature"); err == nil || !strings.Contains(err.Error(), "API client is nil") {
		t.Fatalf("CompareCommits error = %v", err)
	}
	if _, err := GetCommitText(context.Background(), nil, "alice", "demo", "abc", "diff"); err == nil || !strings.Contains(err.Error(), "API client is nil") {
		t.Fatalf("GetCommitText error = %v", err)
	}
}
