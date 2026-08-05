package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	client := NewClient("test-token")
	client.baseURL = "https://example.test"
	client.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder.Result(), nil
	})}
	return client
}

func TestNewClient(t *testing.T) {
	client := NewClient("secret")
	if client.baseURL != BaseURL+APIVersion {
		t.Fatalf("baseURL = %q", client.baseURL)
	}
	if client.token != "secret" {
		t.Fatalf("token = %q", client.token)
	}
	if client.httpClient.Timeout != 30*time.Second {
		t.Fatalf("timeout = %s", client.httpClient.Timeout)
	}
}

func TestNewClientWithHTTPClient(t *testing.T) {
	want := &http.Client{Timeout: time.Second}
	client := NewClientWithHTTPClient("secret", want)
	if client.httpClient != want {
		t.Fatal("custom HTTP client was not retained")
	}

	client = NewClientWithHTTPClient("secret", nil)
	if client.httpClient == nil || client.httpClient.Timeout != 30*time.Second {
		t.Fatalf("nil HTTP client timeout = %v", client.httpClient)
	}
}

func TestNewClientWithBaseURL(t *testing.T) {
	client := NewClientWithBaseURL("secret", "https://example.test/api/v8/", nil)
	if client.baseURL != "https://example.test/api/v8" {
		t.Fatalf("baseURL = %q", client.baseURL)
	}
	if client.token != "secret" {
		t.Fatalf("token = %q", client.token)
	}
}

func TestGetSendsHeadersAndDecodesResponse(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/resource" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.Header.Get("User-Agent"); got != "AtomCode-CLI-v0.4" {
			t.Fatalf("User-Agent = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "demo"})
	})

	var result map[string]string
	if err := client.Get("/resource", &result); err != nil {
		t.Fatal(err)
	}
	if result["name"] != "demo" {
		t.Fatalf("result = %#v", result)
	}
}

func TestRawRequestSupportsCustomAcceptAndEmptyToken(t *testing.T) {
	client := NewClientWithBaseURL("", "https://example.test/api/v8", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v8/download" {
				t.Fatalf("path = %q", req.URL.Path)
			}
			if got := req.Header.Get("Accept"); got != "*/*" {
				t.Fatalf("Accept = %q", got)
			}
			if got := req.Header.Get("Authorization"); got != "" {
				t.Fatalf("Authorization = %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("content")),
				Request:    req,
			}, nil
		}),
	})

	resp, err := client.DoRequestRawWithAccept(http.MethodGet, "/download", "*/*")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "content" {
		t.Fatalf("body = %q", body)
	}
}

func TestDoRequestRawWithBody(t *testing.T) {
	tests := []struct {
		method      string
		body        string
		contentType string
		accept      string
	}{
		{method: http.MethodGet, accept: "application/json"},
		{method: http.MethodPost, body: `{"name":"demo"}`, contentType: "application/json", accept: "application/vnd.atomgit+json"},
		{method: http.MethodPatch, body: "patch"},
		{method: http.MethodPut, body: "put"},
		{method: http.MethodDelete, body: "delete"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotBody, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				if r.Method != tt.method || string(gotBody) != tt.body {
					t.Fatalf("request = %s body %q", r.Method, gotBody)
				}
				if got := r.Header.Get("Content-Type"); got != tt.contentType {
					t.Fatalf("Content-Type = %q", got)
				}
				if got := r.Header.Get("Accept"); got != tt.accept {
					t.Fatalf("Accept = %q", got)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
					t.Fatalf("Authorization = %q", got)
				}
				if got := r.Header.Get("User-Agent"); got != "AtomCode-CLI-v0.4" {
					t.Fatalf("User-Agent = %q", got)
				}
				w.WriteHeader(http.StatusAccepted)
			})

			resp, err := client.DoRequestRawWithBody(tt.method, "/resource", []byte(tt.body), tt.contentType, tt.accept)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusAccepted {
				t.Fatalf("status = %d", resp.StatusCode)
			}
		})
	}
}

func TestDoRequestRawWithBodyReplaysBodyOnRetry(t *testing.T) {
	var calls int32
	client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			call := atomic.AddInt32(&calls, 1)
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != "replay me" {
				t.Fatalf("call %d body = %q", call, body)
			}
			if call == 1 {
				return nil, errors.New("temporary network failure")
			}
			return runRawResponse(req, http.StatusOK, "ok"), nil
		}),
	})

	resp, err := client.DoRequestRawWithBody(http.MethodPut, "/resource", []byte("replay me"), "text/plain", "*/*")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("calls = %d", calls)
	}
}

func runRawResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func TestMethodsEncodeBodies(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		statusCode int
		call       func(*Client, interface{}) error
	}{
		{name: "post", method: http.MethodPost, statusCode: http.StatusCreated, call: func(c *Client, result interface{}) error {
			return c.Post("/resource", map[string]string{"value": "post"}, result)
		}},
		{name: "put", method: http.MethodPut, statusCode: http.StatusOK, call: func(c *Client, result interface{}) error {
			return c.Put("/resource", map[string]string{"value": "put"}, result)
		}},
		{name: "patch", method: http.MethodPatch, statusCode: http.StatusOK, call: func(c *Client, result interface{}) error {
			return c.Patch("/resource", map[string]string{"value": "patch"}, result)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Fatalf("method = %s", r.Method)
				}
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["value"] != tt.name {
					t.Fatalf("body = %#v", body)
				}
				w.WriteHeader(tt.statusCode)
				_, _ = io.WriteString(w, `{"ok":true}`)
			})

			var result map[string]bool
			if err := tt.call(client, &result); err != nil {
				t.Fatal(err)
			}
			if !result["ok"] {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestPatchFormEncodesMultipartBody(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/resource" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{"repo": "demo", "state": "close", "title": "Issue title"} {
			if got := r.FormValue(key); got != want {
				t.Errorf("form field %s = %q, want %q", key, got, want)
			}
		}
		_, _ = io.WriteString(w, `{"state":"closed"}`)
	})

	var result map[string]string
	if err := client.PatchForm("/resource", map[string]string{
		"repo":  "demo",
		"state": "close",
		"title": "Issue title",
	}, &result); err != nil {
		t.Fatal(err)
	}
	if result["state"] != "closed" {
		t.Fatalf("result = %#v", result)
	}
}

func TestPostFormEncodesURLBody(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/resource" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("Content-Type = %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("name") != "bug" || r.Form.Get("color") != "#ff0000" {
			t.Fatalf("form = %#v", r.Form)
		}
		_, _ = io.WriteString(w, `{"name":"bug","color":"#ff0000"}`)
	})

	var result Label
	if err := client.PostForm("/resource", url.Values{
		"name":  {"bug"},
		"color": {"#ff0000"},
	}, &result); err != nil {
		t.Fatal(err)
	}
	if result.Name != "bug" || result.Color != "#ff0000" {
		t.Fatalf("result = %#v", result)
	}
}

func TestPatchAcceptsEmptySuccessResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "200 empty body", statusCode: http.StatusOK},
		{name: "204 empty body", statusCode: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch || r.URL.Path != "/resource" {
					t.Fatalf("request = %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			})

			result := map[string]bool{"existing": true}
			if err := client.Patch("/resource", map[string]string{"value": "patch"}, &result); err != nil {
				t.Fatalf("Patch() error = %v", err)
			}
			if !result["existing"] {
				t.Fatalf("Patch() replaced the existing result: %#v", result)
			}
		})
	}
}

func TestPostAllowsNilBodyAndResult(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			if len(body) != 0 {
				t.Fatalf("body = %q", body)
			}
		}
		w.WriteHeader(http.StatusCreated)
	})

	if err := client.Post("/resource", nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestPostAcceptsEmptySuccessResponseWithoutResult(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "200 empty body", statusCode: http.StatusOK},
		{name: "204 empty body", statusCode: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/resource" {
					t.Fatalf("request = %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			})

			if err := client.Post("/resource", map[string]string{"value": "post"}, nil); err != nil {
				t.Fatalf("Post() error = %v", err)
			}
		})
	}
}

func TestPostReturnsEOFWhenResultIsExpected(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "200 empty body", statusCode: http.StatusOK},
		{name: "201 empty body", statusCode: http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			var result map[string]bool
			err := client.Post("/resource", nil, &result)
			if !errors.Is(err, io.EOF) {
				t.Fatalf("Post() error = %v, want io.EOF", err)
			}
		})
	}
}

func TestPostAcceptsNoContentWithResult(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	result := map[string]bool{"existing": true}
	if err := client.Post("/resource", nil, &result); err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	if !result["existing"] {
		t.Fatalf("Post() replaced the existing result: %#v", result)
	}
}

func TestPostFormAcceptsNoContentWithResult(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	result := map[string]bool{"existing": true}
	if err := client.PostForm("/resource", url.Values{"key": {"value"}}, &result); err != nil {
		t.Fatalf("PostForm() error = %v", err)
	}
	if !result["existing"] {
		t.Fatalf("PostForm() replaced the existing result: %#v", result)
	}
}

func TestDeleteMethods(t *testing.T) {
	t.Run("delete", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Fatalf("method = %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		if err := client.Delete("/resource"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("delete with body", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			var body map[string]int
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["id"] != 42 {
				t.Fatalf("body = %#v", body)
			}
			w.WriteHeader(http.StatusOK)
		})
		if err := client.DeleteWithBody("/resource", map[string]int{"id": 42}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestMethodsReturnAPIError(t *testing.T) {
	tests := []struct {
		name string
		call func(*Client) error
	}{
		{name: "get", call: func(c *Client) error { return c.Get("/fail", &map[string]string{}) }},
		{name: "post", call: func(c *Client) error { return c.Post("/fail", nil, nil) }},
		{name: "post form", call: func(c *Client) error { return c.PostForm("/fail", url.Values{"name": {"bug"}}, nil) }},
		{name: "put", call: func(c *Client) error { return c.Put("/fail", nil, nil) }},
		{name: "patch", call: func(c *Client) error { return c.Patch("/fail", nil, nil) }},
		{name: "patch form", call: func(c *Client) error { return c.PatchForm("/fail", map[string]string{"state": "close"}, nil) }},
		{name: "delete", call: func(c *Client) error { return c.Delete("/fail") }},
		{name: "delete with body", call: func(c *Client) error { return c.DeleteWithBody("/fail", nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "denied", http.StatusForbidden)
			})
			err := tt.call(client)
			if err == nil || !strings.Contains(err.Error(), "403 Forbidden") || !strings.Contains(err.Error(), "denied") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestMethodsRejectUnencodableBody(t *testing.T) {
	badBody := map[string]interface{}{"channel": make(chan int)}
	client := NewClient("token")

	for name, call := range map[string]func() error{
		"post":             func() error { return client.Post("/", badBody, nil) },
		"put":              func() error { return client.Put("/", badBody, nil) },
		"patch":            func() error { return client.Patch("/", badBody, nil) },
		"delete with body": func() error { return client.DeleteWithBody("/", badBody) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); err == nil || !strings.Contains(err.Error(), "unsupported type") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

// newRetryTestClient 构造一个 client，其 transport 前 `failFirst` 次返回网络错误，
// 之后的请求返回 200 + 空 JSON body。用于验证幂等重试逻辑。
func newRetryTestClient(t *testing.T, failFirst int) (*Client, *int32) {
	t.Helper()
	var calls int32
	client := NewClient("test-token")
	client.baseURL = "https://example.test"
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			n := atomic.AddInt32(&calls, 1)
			if int(n) <= failFirst {
				return nil, errors.New("connection reset by peer")
			}
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Content-Type", "application/json")
			_, _ = recorder.WriteString("{}")
			return recorder.Result(), nil
		}),
	}
	return client, &calls
}

func TestClientRetryIdempotent(t *testing.T) {
	// 幂等方法：首次网络错误，第二次成功 → 应触发重试
	cases := []struct {
		name   string
		caller func(*Client) error
	}{
		{
			name:   "GET retries on network error",
			caller: func(c *Client) error { var v map[string]any; return c.Get("/", &v) },
		},
		{
			name:   "PUT retries on network error",
			caller: func(c *Client) error { return c.Put("/", map[string]string{"k": "v"}, nil) },
		},
		{
			name:   "DELETE retries on network error",
			caller: func(c *Client) error { return c.Delete("/") },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := newRetryTestClient(t, 1)
			if err := tc.caller(client); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if n := atomic.LoadInt32(calls); n != 2 {
				t.Fatalf("expected 2 calls (1 fail + 1 retry), got %d", n)
			}
		})
	}

	// 非幂等方法 POST/PATCH：首次失败不重试
	t.Run("POST does not retry", func(t *testing.T) {
		client, calls := newRetryTestClient(t, 1)
		err := client.Post("/", map[string]string{"k": "v"}, nil)
		if err == nil {
			t.Fatal("expected error from POST, got nil")
		}
		if n := atomic.LoadInt32(calls); n != 1 {
			t.Fatalf("expected 1 call (no retry for POST), got %d", n)
		}
	})

	t.Run("PATCH does not retry", func(t *testing.T) {
		client, calls := newRetryTestClient(t, 1)
		err := client.Patch("/", map[string]string{"k": "v"}, nil)
		if err == nil {
			t.Fatal("expected error from PATCH, got nil")
		}
		if n := atomic.LoadInt32(calls); n != 1 {
			t.Fatalf("expected 1 call (no retry for PATCH), got %d", n)
		}
	})

	// 幂等方法连续 2 次都失败：重试一次后返回错误
	t.Run("GET gives up after retry", func(t *testing.T) {
		client, calls := newRetryTestClient(t, 2)
		var v map[string]any
		err := client.Get("/", &v)
		if err == nil {
			t.Fatal("expected error after retry exhaustion, got nil")
		}
		if n := atomic.LoadInt32(calls); n != 2 {
			t.Fatalf("expected 2 calls (1 fail + 1 retry fail), got %d", n)
		}
	})
}

func TestDoJSONRequestExactStatus(t *testing.T) {
	tests := []struct {
		name         string
		allowed      []int
		serverStatus int
		wantError    bool
	}{
		{name: "exact 200 accepted", allowed: []int{http.StatusOK}, serverStatus: http.StatusOK},
		{name: "201 rejected when only 200 allowed", allowed: []int{http.StatusOK}, serverStatus: http.StatusCreated, wantError: true},
		{name: "200 rejected when only 201 allowed", allowed: []int{http.StatusCreated}, serverStatus: http.StatusOK, wantError: true},
		{name: "multiple allowed statuses", allowed: []int{http.StatusOK, http.StatusCreated}, serverStatus: http.StatusCreated},
		{name: "204 rejected when only 200 allowed", allowed: []int{http.StatusOK}, serverStatus: http.StatusNoContent, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.serverStatus)
				_, _ = io.WriteString(w, `{"ok":true}`)
			})

			var result map[string]bool
			err := client.doJSONRequest(http.MethodGet, "/resource", nil, "application/json", "application/json",
				RequestPolicy{AllowedStatuses: tt.allowed, CanRetry: true}, &result)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "API error") {
					t.Fatalf("error = %v, want API error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result["ok"] {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestDoJSONRequestRejectsEmptyJSONBody(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			})

			var result map[string]bool
			err := client.doJSONRequest(http.MethodGet, "/resource", nil, "application/json", "application/json",
				RequestPolicy{AllowedStatuses: []int{status}, CanRetry: true}, &result)
			if !errors.Is(err, io.EOF) {
				t.Fatalf("error = %v, want io.EOF", err)
			}
		})
	}
}

func TestDoJSONRequestRetryPolicy(t *testing.T) {
	t.Run("CanRetry false does not retry on network error", func(t *testing.T) {
		var calls int32
		client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				atomic.AddInt32(&calls, 1)
				return nil, errors.New("network failure")
			}),
		})

		err := client.doJSONRequest(http.MethodPut, "/resource", bytes.NewReader([]byte("body")),
			"application/json", "application/json",
			RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "API request PUT /resource") {
			t.Fatalf("error = %v, want wrapped transport error", err)
		}
		if !errors.Is(err, errors.New("network failure")) {
			if !strings.Contains(err.Error(), "network failure") {
				t.Fatalf("error does not preserve cause: %v", err)
			}
		}
		if n := atomic.LoadInt32(&calls); n != 1 {
			t.Fatalf("calls = %d, want 1 (no retry)", n)
		}
	})

	t.Run("CanRetry true retries once on network error", func(t *testing.T) {
		var calls int32
		client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				n := atomic.AddInt32(&calls, 1)
				if n == 1 {
					return nil, errors.New("network failure")
				}
				recorder := httptest.NewRecorder()
				recorder.Header().Set("Content-Type", "application/json")
				_, _ = recorder.WriteString("{}")
				return recorder.Result(), nil
			}),
		})

		var result map[string]any
		err := client.doJSONRequest(http.MethodPut, "/resource", bytes.NewReader([]byte("body")),
			"application/json", "application/json",
			RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n := atomic.LoadInt32(&calls); n != 2 {
			t.Fatalf("calls = %d, want 2 (1 fail + 1 retry)", n)
		}
	})
}

func TestAPIErrorBoundedBody(t *testing.T) {
	largeBody := strings.Repeat("A", maxErrorBodyBytes+1000)
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, largeBody, http.StatusForbidden)
	})

	err := client.Get("/fail", &map[string]string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(err.Error()) > maxErrorBodyBytes+200 {
		t.Fatalf("error message too long: %d bytes", len(err.Error()))
	}
	if !strings.HasSuffix(err.Error(), "...") {
		t.Fatalf("error should be truncated, got suffix: ...%s", err.Error()[len(err.Error())-20:])
	}
}

func TestAPIErrorRedactsCredentials(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		secret string
	}{
		{
			name:   "bearer token",
			body:   `{"error":"Authorization: Bearer bearer-secret-123 failed"}`,
			secret: "bearer-secret-123",
		},
		{
			name:   "JSON access token",
			body:   `{"access_token":"access-secret-123","message":"failed"}`,
			secret: "access-secret-123",
		},
		{
			name:   "JSON refresh token",
			body:   `{"refresh_token":"refresh-secret-123"}`,
			secret: "refresh-secret-123",
		},
		{
			name:   "JSON client secret",
			body:   `{"client_secret":"client-secret-123"}`,
			secret: "client-secret-123",
		},
		{
			name:   "JSON authorization",
			body:   `{"authorization":"Basic authorization-secret-123"}`,
			secret: "authorization-secret-123",
		},
		{
			name:   "JSON password",
			body:   `{"password":"password-secret-123"}`,
			secret: "password-secret-123",
		},
		{
			name:   "JSON token",
			body:   `{"token":"generic-secret-123"}`,
			secret: "generic-secret-123",
		},
		{
			name:   "equals key value",
			body:   "access_token=equals-secret-123",
			secret: "equals-secret-123",
		},
		{
			name:   "colon key value",
			body:   "refresh_token: colon-secret-123",
			secret: "colon-secret-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, tt.body, http.StatusForbidden)
			})

			err := client.Get("/fail", &map[string]string{})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if strings.Contains(err.Error(), tt.secret) {
				t.Fatalf("error leaked credential: %s", err.Error())
			}
			if !strings.Contains(err.Error(), "<redacted>") {
				t.Fatalf("error did not redact credential: %s", err.Error())
			}
		})
	}
}

func TestAPIErrorRedactsCredentialTruncatedMidValue(t *testing.T) {
	const visibleSecret = "truncated"
	const credentialPrefix = `","access_token":"`
	const bodyPrefix = `{"padding":"`
	paddingLength := maxErrorBodyBytes - len(bodyPrefix) - len(credentialPrefix) - len(visibleSecret)
	body := bodyPrefix + strings.Repeat("A", paddingLength) +
		credentialPrefix + visibleSecret + "-secret-value\"}"

	resp := &http.Response{
		Status: http.StatusText(http.StatusForbidden),
		Body:   io.NopCloser(strings.NewReader(body)),
	}
	err := newAPIError(resp)
	if strings.Contains(err.Error(), visibleSecret) {
		t.Fatalf("error leaked truncated credential prefix: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "<redacted>") {
		t.Fatalf("error did not redact truncated credential: %s", err.Error())
	}
}

func TestAPIErrorSanitizesControlChars(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "error \x1b[31mred\x1b[0m text", http.StatusForbidden)
	})

	err := client.Get("/fail", &map[string]string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if strings.Contains(err.Error(), "\x1b") {
		t.Fatalf("error contained raw escape: %q", err.Error())
	}
	if !strings.Contains(err.Error(), `\x1b`) {
		t.Fatalf("error did not sanitize escape: %q", err.Error())
	}
}

func TestAPIErrorSanitizesUnicodeDirectionControls(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "error \u202ereversed\u2066isolated", http.StatusForbidden)
	})

	err := client.Get("/fail", &map[string]string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	for _, control := range []string{"\u202e", "\u2066"} {
		if strings.Contains(err.Error(), control) {
			t.Fatalf("error contained raw direction control %q: %q", control, err.Error())
		}
	}
	for _, escaped := range []string{`\u202e`, `\u2066`} {
		if !strings.Contains(err.Error(), escaped) {
			t.Fatalf("error did not expose sanitized control %q: %q", escaped, err.Error())
		}
	}
}

func TestAPIErrorWrapsTransportError(t *testing.T) {
	originalErr := errors.New("connection reset by peer")
	client := NewClientWithBaseURL("token", "https://example.test", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, originalErr
		}),
	})

	err := client.Get("/resource", &map[string]string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "API request GET /resource") {
		t.Fatalf("error missing context: %v", err)
	}
	if !errors.Is(err, originalErr) {
		t.Fatalf("error does not wrap cause: %v", err)
	}
}

func TestLegacyHelpersPreserveStatuses(t *testing.T) {
	t.Run("Get accepts 200 and 201", func(t *testing.T) {
		for _, status := range []int{http.StatusOK, http.StatusCreated} {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"ok":true}`)
			})
			var result map[string]bool
			if err := client.Get("/resource", &result); err != nil {
				t.Fatalf("Get with status %d: %v", status, err)
			}
		}
	})

	t.Run("Get rejects 204", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		err := client.Get("/resource", &map[string]bool{})
		if err == nil {
			t.Fatal("Get with 204 should fail")
		}
	})

	t.Run("Post accepts 200 201 204", func(t *testing.T) {
		for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusNoContent} {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				if status != http.StatusNoContent {
					_, _ = io.WriteString(w, `{"ok":true}`)
				}
			})
			if err := client.Post("/resource", map[string]string{"k": "v"}, nil); err != nil {
				t.Fatalf("Post with status %d: %v", status, err)
			}
		}
	})

	t.Run("Put accepts only 200", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		})
		err := client.Put("/resource", map[string]string{"k": "v"}, nil)
		if err == nil {
			t.Fatal("Put with 201 should fail")
		}
	})

	t.Run("Patch accepts 200 and 204", func(t *testing.T) {
		for _, status := range []int{http.StatusOK, http.StatusNoContent} {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				if status != http.StatusNoContent {
					_, _ = io.WriteString(w, `{"ok":true}`)
				}
			})
			if err := client.Patch("/resource", map[string]string{"k": "v"}, nil); err != nil {
				t.Fatalf("Patch with status %d: %v", status, err)
			}
		}
	})

	t.Run("Delete accepts 200 and 204", func(t *testing.T) {
		for _, status := range []int{http.StatusOK, http.StatusNoContent} {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			})
			if err := client.Delete("/resource"); err != nil {
				t.Fatalf("Delete with status %d: %v", status, err)
			}
		}
	})
}
