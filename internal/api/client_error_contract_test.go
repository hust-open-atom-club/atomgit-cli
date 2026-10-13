package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type failingJSONValue struct{ err error }

func (v failingJSONValue) MarshalJSON() ([]byte, error) { return nil, v.err }
func (v *failingJSONValue) UnmarshalJSON([]byte) error  { return v.err }

// dataAndErrorReader exercises a legal Read returning data and an error together.
type dataAndErrorReader struct {
	data string
	err  error
}

func (r *dataAndErrorReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	if r.data != "" {
		return n, nil
	}
	return n, r.err
}

type responseTimeoutError struct{}

func (responseTimeoutError) Error() string   { return "response timeout" }
func (responseTimeoutError) Timeout() bool   { return true }
func (responseTimeoutError) Temporary() bool { return false }

func TestMethodsPreserveJSONEncodingFailures(t *testing.T) {
	cause := errors.New("custom marshal failure")
	bodies := []struct {
		name  string
		value any
		cause error
	}{
		{name: "unsupported type", value: make(chan int)},
		{name: "custom marshaler", value: failingJSONValue{err: cause}, cause: cause},
	}
	methods := []struct {
		name string
		call func(*Client, any) error
	}{
		{name: http.MethodPost, call: func(c *Client, v any) error { return c.Post("/resource", v, nil) }},
		{name: http.MethodPut, call: func(c *Client, v any) error { return c.Put("/resource", v, nil) }},
		{name: http.MethodPatch, call: func(c *Client, v any) error { return c.Patch("/resource", v, nil) }},
		{name: http.MethodDelete, call: func(c *Client, v any) error { return c.DeleteWithBody("/resource", v) }},
	}
	for _, method := range methods {
		for _, body := range bodies {
			t.Run(method.name+"/"+body.name, func(t *testing.T) {
				calls := 0
				client := NewClientWithHTTPClient("fixture", &http.Client{
					Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
						calls++
						return nil, errors.New("unexpected request")
					}),
				})
				err := method.call(client, body.value)
				if !errors.Is(err, ErrRequestPreparation) || calls != 0 {
					t.Fatalf("error = %v, calls = %d; want preparation failure before execution", err, calls)
				}
				if !strings.Contains(err.Error(), "API request "+method.name+" /resource") {
					t.Fatalf("missing operation context: %v", err)
				}
				if body.cause != nil {
					if !errors.Is(err, body.cause) {
						t.Fatalf("lost custom marshal cause: %v", err)
					}
					if _, ok := errors.AsType[*json.MarshalerError](err); !ok {
						t.Fatalf("lost marshaler error: %v", err)
					}
				} else if _, ok := errors.AsType[*json.UnsupportedTypeError](err); !ok {
					t.Fatalf("lost unsupported type error: %v", err)
				}
			})
		}
	}
}

func TestDecodeJSONResponsePreservesDecodeFailures(t *testing.T) {
	cause := errors.New("custom unmarshal failure")
	tests := []struct {
		name          string
		body          string
		result        any
		cause         error
		parserType    string
		invalidTarget bool
	}{
		{name: "syntax", body: `{"x":]}`, result: new(map[string]any), parserType: "syntax"},
		{name: "field type", body: `{"x":"wrong"}`, result: new(map[string]int), parserType: "type"},
		{name: "truncated JSON", body: `{"x":`, result: new(map[string]any), cause: io.ErrUnexpectedEOF},
		{name: "empty JSON", result: new(map[string]any), cause: io.EOF},
		{name: "custom unmarshal", body: `{}`, result: &failingJSONValue{err: cause}, cause: cause},
		{name: "invalid target", body: `{}`, invalidTarget: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := decodeJSONResponse(http.MethodGet, "/resource", strings.NewReader(tt.body), tt.result)
			wantInvalid := !tt.invalidTarget
			if err == nil || errors.Is(err, ErrInvalidResponse) != wantInvalid {
				t.Fatalf("wrong invalid response contract: %v", err)
			}
			if _, ok := errors.AsType[*ResponseReadError](err); ok {
				t.Fatalf("decoder failure misclassified as read failure: %v", err)
			}
			if tt.cause != nil && !errors.Is(err, tt.cause) {
				t.Fatalf("lost decoder cause %v: %v", tt.cause, err)
			}
			switch tt.parserType {
			case "syntax":
				if _, ok := errors.AsType[*json.SyntaxError](err); !ok {
					t.Fatalf("lost syntax error: %v", err)
				}
			case "type":
				if _, ok := errors.AsType[*json.UnmarshalTypeError](err); !ok {
					t.Fatalf("lost type error: %v", err)
				}
			}
			if tt.invalidTarget {
				if _, ok := errors.AsType[*json.InvalidUnmarshalError](err); !ok {
					t.Fatalf("lost invalid target error: %v", err)
				}
			}
		})
	}
}

func TestDecodeJSONResponsePreservesReadFailures(t *testing.T) {
	cause := errors.New("read interrupted")
	tests := []struct {
		name    string
		body    io.Reader
		cause   error
		syntax  bool
		timeout bool
	}{
		{name: "before data", body: apiErrorReader{err: cause}, cause: cause},
		{
			name: "after partial JSON", cause: cause,
			body: io.MultiReader(strings.NewReader(`{"x":`), apiErrorReader{err: cause}),
		},
		{
			name: "data and error together", cause: cause,
			body: &dataAndErrorReader{data: `{"x":`, err: cause},
		},
		{
			name: "syntax and read causes", cause: cause, syntax: true,
			body: &dataAndErrorReader{data: `{"x":]}`, err: cause},
		},
		{name: "canceled", body: apiErrorReader{err: context.Canceled}, cause: context.Canceled},
		{
			name: "deadline", cause: context.DeadlineExceeded, timeout: true,
			body: apiErrorReader{err: context.DeadlineExceeded},
		},
		{
			name: "transport truncation", cause: io.ErrUnexpectedEOF,
			body: &dataAndErrorReader{data: `{"x":`, err: io.ErrUnexpectedEOF},
		},
		{
			name: "timeout", cause: responseTimeoutError{}, timeout: true,
			body: apiErrorReader{err: responseTimeoutError{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]any
			err := decodeJSONResponse(http.MethodGet, "/resource", tt.body, &result)
			readErr, ok := errors.AsType[*ResponseReadError](err)
			if !ok || !errors.Is(readErr.Err, tt.cause) || !errors.Is(err, tt.cause) {
				t.Fatalf("lost read contract or cause: %v", err)
			}
			if errors.Is(err, ErrInvalidResponse) || readErr.Timeout() != tt.timeout {
				t.Fatalf("wrong read category or timeout: %v", err)
			}
			if tt.syntax {
				if _, ok := errors.AsType[*json.SyntaxError](err); !ok {
					t.Fatalf("lost accompanying syntax cause: %v", err)
				}
			}
		})
	}
}

func TestDecodeJSONResponseUsesReadCauseForTimeout(t *testing.T) {
	cause := errors.New("non-timeout failure")
	tests := []struct {
		name        string
		readCause   error
		decodeCause error
		timeout     bool
	}{
		{name: "parser timeout only", readCause: cause, decodeCause: responseTimeoutError{}},
		{name: "reader timeout", readCause: responseTimeoutError{}, decodeCause: cause, timeout: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := &dataAndErrorReader{data: `{}`, err: tt.readCause}
			result := &failingJSONValue{err: tt.decodeCause}
			err := decodeJSONResponse(http.MethodGet, "/resource", body, result)
			readErr, ok := errors.AsType[*ResponseReadError](err)
			if !ok || readErr.Timeout() != tt.timeout {
				t.Fatalf("wrong timeout role: %v", err)
			}
			if !errors.Is(err, tt.readCause) || !errors.Is(err, tt.decodeCause) {
				t.Fatalf("lost distinct read or decode cause: %v", err)
			}
		})
	}
}

func TestDecodeJSONResponseKeepsFirstValueSuccess(t *testing.T) {
	cause := errors.New("unused response tail")
	tests := []struct {
		name string
		body io.Reader
	}{
		{name: "another JSON value", body: strings.NewReader(`{"ok":true} {"ok":false}`)},
		{name: "invalid tail", body: strings.NewReader(`{"ok":true} invalid`)},
		{name: "data and EOF", body: &dataAndErrorReader{data: `{"ok":true}`, err: io.EOF}},
		{name: "data and failure", body: &dataAndErrorReader{data: `{"ok":true}`, err: cause}},
		{
			name: "unread tail failure",
			body: io.MultiReader(strings.NewReader(`{"ok":true}`), apiErrorReader{err: cause}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]bool
			if err := decodeJSONResponse(http.MethodGet, "/resource", tt.body, &result); err != nil {
				t.Fatalf("first JSON value no longer succeeds: %v", err)
			}
			if !result["ok"] {
				t.Fatalf("first result replaced: %#v", result)
			}
		})
	}
}

func TestJSONMethodsExposeResponseContracts(t *testing.T) {
	methods := []struct {
		name string
		call func(*Client, any) error
	}{
		{name: "get", call: func(c *Client, v any) error { return c.Get("/resource", v) }},
		{name: "post", call: func(c *Client, v any) error { return c.Post("/resource", nil, v) }},
		{name: "post form", call: func(c *Client, v any) error { return c.PostForm("/resource", url.Values{}, v) }},
		{name: "put", call: func(c *Client, v any) error { return c.Put("/resource", nil, v) }},
		{name: "put form", call: func(c *Client, v any) error { return c.PutForm("/resource", url.Values{}, v) }},
		{name: "patch", call: func(c *Client, v any) error { return c.Patch("/resource", nil, v) }},
		{
			name: "patch form",
			call: func(c *Client, v any) error { return c.PatchForm("/resource", map[string]string{}, v) },
		},
		{
			name: "exact status request",
			call: func(c *Client, v any) error {
				return c.doJSONRequest(http.MethodGet, "/resource", nil, "", "",
					RequestPolicy{AllowedStatuses: []int{http.StatusOK}}, v)
			},
		},
	}
	for _, method := range methods {
		for _, failure := range []string{"decode", "read", "HTTP"} {
			t.Run(method.name+"/"+failure, func(t *testing.T) {
				cause := errors.New("read failure")
				calls := 0
				client := NewClientWithHTTPClient("fixture", &http.Client{
					Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						calls++
						status := http.StatusOK
						var body io.Reader = strings.NewReader(`{"x":`)
						if failure != "decode" {
							body = apiErrorReader{err: cause}
						}
						if failure == "HTTP" {
							status = http.StatusForbidden
						}
						return &http.Response{
							StatusCode: status, Header: make(http.Header),
							Body: io.NopCloser(body), Request: req,
						}, nil
					}),
				})
				var result map[string]any
				err := method.call(client, &result)
				readErr, read := errors.AsType[*ResponseReadError](err)
				switch failure {
				case "decode":
					if !errors.Is(err, ErrInvalidResponse) || !errors.Is(err, io.ErrUnexpectedEOF) || read {
						t.Fatalf("wrong decode contract: %v", err)
					}
				case "read":
					if !read || !errors.Is(readErr.Err, cause) || errors.Is(err, ErrInvalidResponse) {
						t.Fatalf("wrong read contract: %v", err)
					}
				case "HTTP":
					httpErr, ok := errors.AsType[*HTTPError](err)
					if !ok || httpErr.StatusCode != http.StatusForbidden || !errors.Is(httpErr.ReadError, cause) {
						t.Fatalf("lost HTTP primary failure or read cause: %v", err)
					}
					if read || errors.Is(err, ErrInvalidResponse) {
						t.Fatalf("HTTP failure misclassified as successful response failure: %v", err)
					}
				}
				if calls != 1 {
					t.Fatalf("calls = %d, want one request", calls)
				}
			})
		}
	}
}
