package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	baseapi "atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
)

type contractJSONValue struct{ err error }

func (v contractJSONValue) MarshalJSON() ([]byte, error) { return nil, v.err }
func (v *contractJSONValue) UnmarshalJSON([]byte) error  { return v.err }

type contractTimeoutError struct{}

func (contractTimeoutError) Error() string   { return "body timeout" }
func (contractTimeoutError) Timeout() bool   { return true }
func (contractTimeoutError) Temporary() bool { return false }

type contractBody struct {
	io.Reader
	closed bool
}

func (b *contractBody) Close() error { b.closed = true; return nil }

type contractDataReader struct {
	data string
	err  error
}

func (r *contractDataReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	if r.data != "" {
		return n, nil
	}
	return n, r.err
}

// Exercise both shared entry points with identical response and cause fixtures.
func callContractJSON(c *Client, method string, result any) error {
	if method == http.MethodGet {
		return c.getJSON("get fixture", "/fixture", result)
	}
	return c.postJSON("post fixture", "/fixture", map[string]string{"input": "value"}, result)
}

func TestActionsPostJSONPreservesEncodingFailures(t *testing.T) {
	cause := errors.New("marshal cause")
	for _, tt := range []struct {
		name  string
		value any
		cause error
	}{
		{name: "unsupported", value: make(chan int)},
		{name: "marshaler", value: contractJSONValue{err: cause}, cause: cause},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := NewClientWithHTTPClient("fixture", &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls++
					return nil, errors.New("unexpected request")
				}),
			})
			err := client.postJSON("post fixture", "/fixture", tt.value, nil)
			if !errors.Is(err, baseapi.ErrRequestPreparation) || calls != 0 {
				t.Fatalf("error = %v, calls = %d; want preparation failure before execution", err, calls)
			}
			for _, value := range []string{"post fixture", "POST", "/fixture", "marshal payload"} {
				if !strings.Contains(err.Error(), value) {
					t.Fatalf("missing context %q: %v", value, err)
				}
			}
			if tt.cause != nil {
				if _, ok := errors.AsType[*json.MarshalerError](err); !ok || !errors.Is(err, tt.cause) {
					t.Fatalf("lost marshaler cause: %v", err)
				}
			} else if _, ok := errors.AsType[*json.UnsupportedTypeError](err); !ok {
				t.Fatalf("lost unsupported type cause: %v", err)
			}
		})
	}
}

func TestActionsJSONResponseContracts(t *testing.T) {
	readCause := errors.New("reader failed")
	decodeCause := errors.New("unmarshaler failed")
	tests := []struct {
		name        string
		data        string
		readCause   error
		decodeCause error
		wantCause   error
		wantInvalid bool
		wantSyntax  bool
		wantType    bool
		badTarget   bool
	}{
		{name: "syntax", data: `{!}`, wantInvalid: true, wantSyntax: true},
		{name: "type", data: `{"n":"text"}`, wantInvalid: true, wantType: true},
		{name: "empty", wantInvalid: true, wantCause: io.EOF},
		{name: "truncated", data: `{"n":`, wantInvalid: true, wantCause: io.ErrUnexpectedEOF},
		{name: "unmarshaler", data: `{}`, decodeCause: decodeCause, wantInvalid: true, wantCause: decodeCause},
		{name: "invalid target", data: `{}`, badTarget: true},
		{name: "read start", readCause: readCause, wantCause: readCause},
		{name: "read partial", data: `{"n":`, readCause: readCause, wantCause: readCause},
		{name: "syntax and read", data: `{!}`, readCause: readCause, wantCause: readCause, wantSyntax: true},
		{name: "canceled", readCause: context.Canceled, wantCause: context.Canceled},
		{name: "deadline", readCause: context.DeadlineExceeded, wantCause: context.DeadlineExceeded},
		{name: "timeout", readCause: contractTimeoutError{}, wantCause: contractTimeoutError{}},
		{name: "reader unexpected EOF", readCause: io.ErrUnexpectedEOF, wantCause: io.ErrUnexpectedEOF},
		{name: "wrapped EOF read", readCause: fmt.Errorf("read interrupted: %w", io.EOF), wantCause: io.EOF},
		{name: "decode timeout only", data: `{}`, decodeCause: contractTimeoutError{}, wantInvalid: true,
			wantCause: contractTimeoutError{}},
		{name: "read timeout and unmarshal", data: `{}`, readCause: contractTimeoutError{},
			decodeCause: decodeCause, wantCause: decodeCause},
	}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		for _, tt := range tests {
			t.Run(method+"/"+tt.name, func(t *testing.T) {
				var reader io.Reader = strings.NewReader(tt.data)
				if tt.readCause != nil {
					reader = &contractDataReader{data: tt.data, err: tt.readCause}
				}
				body := &contractBody{Reader: reader}
				calls := 0
				client := NewClientWithHTTPClient("fixture", &http.Client{
					Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						calls++
						resp := response(req, http.StatusOK, "")
						resp.Body = body
						return resp, nil
					}),
				})
				var value struct {
					N int `json:"n"`
				}
				var target any = &value
				if tt.decodeCause != nil {
					target = &contractJSONValue{err: tt.decodeCause}
				} else if tt.badTarget {
					target = 42
				}
				err := callContractJSON(client, method, target)
				if err == nil || errors.Is(err, baseapi.ErrInvalidResponse) != tt.wantInvalid {
					t.Fatalf("error = %v, want invalid = %v", err, tt.wantInvalid)
				}
				if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
					t.Fatalf("lost cause %v: %v", tt.wantCause, err)
				}
				readErr, read := errors.AsType[*baseapi.ResponseReadError](err)
				if read != (tt.readCause != nil) {
					t.Fatalf("wrong response read role: %v", err)
				}
				if read {
					if !errors.Is(readErr.Err, tt.readCause) {
						t.Fatalf("lost actual reader cause: %v", err)
					}
					wantTimeout := errors.Is(tt.readCause, context.DeadlineExceeded) ||
						errors.Is(tt.readCause, contractTimeoutError{})
					if readErr.Timeout() != wantTimeout {
						t.Fatalf("read timeout = %v, want %v", readErr.Timeout(), wantTimeout)
					}
				}
				if _, ok := errors.AsType[*json.SyntaxError](err); ok != tt.wantSyntax {
					t.Fatalf("syntax cause = %v, want %v: %v", ok, tt.wantSyntax, err)
				}
				if _, ok := errors.AsType[*json.UnmarshalTypeError](err); ok != tt.wantType {
					t.Fatalf("type cause = %v, want %v: %v", ok, tt.wantType, err)
				}
				if _, ok := errors.AsType[*json.InvalidUnmarshalError](err); ok != tt.badTarget {
					t.Fatalf("target cause = %v, want %v: %v", ok, tt.badTarget, err)
				}
				if !strings.Contains(err.Error(), strings.ToLower(method)+" fixture: decode response") {
					t.Fatalf("lost operation/decode context: %v", err)
				}
				if calls != 1 || !body.closed {
					t.Fatalf("calls = %d, closed = %v", calls, body.closed)
				}
			})
		}
	}
}

func TestActionsJSONHTTPFailuresKeepStatusPrimary(t *testing.T) {
	cause := fmt.Errorf("password=read-secret: %w", contractTimeoutError{})
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		for _, status := range []int{403, 404, 429, 500, 503} {
			t.Run(fmt.Sprintf("%s/%d", method, status), func(t *testing.T) {
				body := &contractBody{Reader: io.MultiReader(
					strings.NewReader(`{"message":"server denied; access_token=body-secret"}`),
					errorReader{err: cause},
				)}
				calls := 0
				client := NewClientWithHTTPClient("fixture", &http.Client{
					Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						calls++
						resp := response(req, status, "")
						resp.Body = body
						// Exceed the GET retry budget so terminal 429 is inspected without sleeping.
						resp.Header.Set("Retry-After", "3600")
						return resp, nil
					}),
				})
				err := callContractJSON(client, method, new(any))
				httpErr, ok := errors.AsType[*HTTPError](err)
				if !ok || httpErr.StatusCode != status || httpErr.Operation != strings.ToLower(method)+" fixture" {
					t.Fatalf("lost primary HTTP status/operation: %v", err)
				}
				if !errors.Is(httpErr.ReadError, cause) || !errors.Is(err, cause) || httpErr.RetryAfter != "3600" {
					t.Fatalf("lost secondary read cause or retry metadata: %v", err)
				}
				if _, read := errors.AsType[*baseapi.ResponseReadError](err); read ||
					errors.Is(err, baseapi.ErrInvalidResponse) {
					t.Fatalf("HTTP failure misclassified as a successful response failure: %v", err)
				}
				if strings.Contains(err.Error(), "read-secret") || strings.Contains(err.Error(), "body-secret") ||
					!strings.Contains(err.Error(), "<redacted>") {
					t.Fatalf("unsafe HTTP rendering: %v", err)
				}
				if calls != 1 || !body.closed {
					t.Fatalf("calls = %d, closed = %v", calls, body.closed)
				}
			})
		}
	}
}

func TestActionsJSONSuccessCompatibility(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		for _, tt := range []struct {
			name string
			data string
			err  error
		}{
			{name: "first value", data: `{"n":7} {"n":8}`},
			{name: "trailing garbage", data: `{"n":7} garbage`},
			{name: "data and EOF", data: `{"n":7}`, err: io.EOF},
			{name: "data and failure", data: `{"n":7}`, err: errors.New("tail failed")},
		} {
			t.Run(method+"/"+tt.name, func(t *testing.T) {
				body := &contractBody{Reader: &contractDataReader{data: tt.data, err: tt.err}}
				client := NewClientWithHTTPClient("fixture", &http.Client{
					Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						resp := response(req, http.StatusOK, "")
						resp.Body = body
						return resp, nil
					}),
				})
				var result struct {
					N int `json:"n"`
				}
				if err := callContractJSON(client, method, &result); err != nil || result.N != 7 || !body.closed {
					t.Fatalf("error = %v, result = %v, closed = %v", err, result, body.closed)
				}
			})
		}
	}
}

func TestActionsPostJSONEmptyResponsePolicy(t *testing.T) {
	for _, status := range []int{200, 201, 202, 204} {
		for _, failure := range []string{"empty", "invalid", "wrapped EOF", "nil result"} {
			t.Run(fmt.Sprintf("%d/%s", status, failure), func(t *testing.T) {
				var reader io.Reader = strings.NewReader("")
				switch failure {
				case "invalid":
					reader = strings.NewReader(`{"n":`)
				case "wrapped EOF", "nil result":
					reader = errorReader{err: fmt.Errorf("reader failed: %w", io.EOF)}
				}
				body := &contractBody{Reader: reader}
				calls := 0
				client := NewClientWithHTTPClient("fixture", &http.Client{
					Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						calls++
						resp := response(req, status, "")
						resp.Body = body
						return resp, nil
					}),
				})
				var target any = new(any)
				if failure == "nil result" {
					target = nil
				}
				err := callContractJSON(client, http.MethodPost, target)
				wantErr := status != http.StatusNoContent && failure != "nil result" &&
					(failure != "empty" || status == http.StatusOK)
				if (err != nil) != wantErr {
					t.Fatalf("error = %v, want error = %v", err, wantErr)
				}
				if wantErr {
					_, read := errors.AsType[*baseapi.ResponseReadError](err)
					if read != (failure == "wrapped EOF") ||
						errors.Is(err, baseapi.ErrInvalidResponse) != (failure != "wrapped EOF") {
						t.Fatalf("wrong empty/decode/read contract: %v", err)
					}
				}
				if calls != 1 || !body.closed {
					t.Fatalf("calls = %d, closed = %v", calls, body.closed)
				}
			})
		}
	}
}

func TestActionsJSONRequestFailures(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			client := NewClientWithHTTPClient("fixture", &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return nil, context.Canceled
				}),
			})
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err := callContractJSON(client.WithContext(ctx), method, new(any))
			if _, ok := errors.AsType[*baseapi.RequestExecutionError](err); !ok ||
				!errors.Is(err, context.Canceled) {
				t.Fatalf("lost request execution contract: %v", err)
			}
			if _, read := errors.AsType[*baseapi.ResponseReadError](err); read ||
				errors.Is(err, baseapi.ErrInvalidResponse) || errors.Is(err, baseapi.ErrRequestPreparation) {
				t.Fatalf("execution failure misclassified: %v", err)
			}
		})
	}
}

func TestGetJobPreservesEmptyResponseCause(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
	}{
		{name: "normal EOF", err: io.EOF},
		{name: "reader wrapped EOF", err: fmt.Errorf("reader interrupted: %w", io.EOF)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithHTTPClient("fixture", &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					resp := response(req, http.StatusOK, "")
					resp.Body = io.NopCloser(errorReader{err: tt.err})
					return resp, nil
				}),
			})
			_, err := client.GetJob("team", "demo", "run", "job")
			_, read := errors.AsType[*baseapi.ResponseReadError](err)
			if !errors.Is(err, io.EOF) || read != (tt.name != "normal EOF") ||
				errors.Is(err, baseapi.ErrInvalidResponse) != !read {
				t.Fatalf("lost empty/read cause contract: %v", err)
			}
			missing, missingJob := errors.AsType[*JobNotFoundError](err)
			if missingJob == read || missingJob && missing.JobID != "job" {
				t.Fatalf("missing-job boundary = %#v (read failure %t)", missing, read)
			}
			if strings.Contains(err.Error(), "not found") == read {
				t.Fatalf("reader failure translated as missing job, or missing empty-response hint: %v", err)
			}
		})
	}
}
