package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
