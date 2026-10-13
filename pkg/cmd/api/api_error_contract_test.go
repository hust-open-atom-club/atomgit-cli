package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	internalapi "atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
)

type apiTimeoutError struct{}

func (apiTimeoutError) Error() string   { return "response timeout" }
func (apiTimeoutError) Timeout() bool   { return true }
func (apiTimeoutError) Temporary() bool { return false }

type apiErrorBody struct {
	io.Reader
	closed bool
}

func (b *apiErrorBody) Close() error { b.closed = true; return nil }

type apiFailingWriter struct{ err error }

func (w apiFailingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestAPIHTTPFailuresPreserveStatusAndReadRole(t *testing.T) {
	const token = "configured-token"
	for _, paginate := range []bool{false, true} {
		for _, status := range []int{403, 404, 429, 500, 502} {
			for _, readFailure := range []bool{false, true} {
				t.Run(fmt.Sprintf("paginate=%v/status=%d/read=%v", paginate, status, readFailure), func(t *testing.T) {
					cause := fmt.Errorf("password=read-sensitive; %s: %w", token, apiTimeoutError{})
					var reader io.Reader = strings.NewReader(`{"message":"denied ` + token + `"}`)
					if readFailure {
						reader = io.MultiReader(reader, failingReader{err: cause})
					}
					body := &apiErrorBody{Reader: reader}
					fixture := newAPITestFixture(token, func(req *http.Request) (*http.Response, error) {
						resp := apiTestResponse(req, status, "")
						resp.Header.Set("Retry-After", "3600")
						resp.Body = body
						return resp, nil
					})
					cmd := NewCmdAPI(fixture.factory)
					cmd.SetOut(fixture.stdout)
					cmd.SetErr(fixture.stderr)
					args := []string{"/items"}
					if paginate {
						args = append(args, "--paginate")
					}
					cmd.SetArgs(args)
					err := cmd.Execute()
					httpErr, ok := errors.AsType[*internalapi.HTTPError](err)
					if !ok || httpErr.StatusCode != status || !internalapi.IsHTTPStatus(err, status) {
						t.Fatalf("lost primary HTTP status: %v", err)
					}
					if httpErr.Status != fmt.Sprintf("%d %s", status, http.StatusText(status)) ||
						httpErr.RetryAfter != "3600" {
						t.Fatalf("lost HTTP metadata: %#v", httpErr)
					}
					if errors.Is(httpErr.ReadError, cause) != readFailure || errors.Is(err, cause) != readFailure {
						t.Fatalf("lost secondary reader cause: %v", err)
					}
					if _, read := errors.AsType[*internalapi.ResponseReadError](err); read ||
						errors.Is(err, internalapi.ErrInvalidResponse) {
						t.Fatalf("HTTP failure misclassified as successful response failure: %v", err)
					}
					operation := "GET /items"
					if paginate {
						operation = "request page 1"
					}
					wantPrefix := fmt.Sprintf("%s: API request failed (%d %s)", operation, status, http.StatusText(status))
					if !strings.HasPrefix(err.Error(), wantPrefix) || !strings.Contains(err.Error(), "retry after 3600") {
						t.Fatalf("lost established safe message: %v", err)
					}
					assertNoTokenLeak(t, token, err.Error(), fixture.stderr.String())
					if strings.Contains(err.Error(), "read-sensitive") {
						t.Fatalf("unsafe secondary cause text: %v", err)
					}
					if fixture.stdout.Len() != 0 || fixture.requests != 1 || !body.closed {
						t.Fatalf("stdout = %q, calls = %d, closed = %v", fixture.stdout, fixture.requests, body.closed)
					}
				})
			}
		}
	}
}

func TestAPIPaginationFailuresPreserveCauseAndCompletedPages(t *testing.T) {
	const token = "configured-token"
	tests := []struct {
		name       string
		body       string
		header     string
		fallback   bool
		readCause  error
		wantSyntax bool
		wantType   bool
		wantNum    bool
		wantCause  error
	}{
		{name: "syntax", body: `{!}`, wantSyntax: true},
		{name: "truncated", body: `[`, wantSyntax: true},
		{name: "second JSON value", body: `[2] [3]`, wantSyntax: true},
		{name: "trailing garbage", body: `[2] garbage`, wantSyntax: true},
		{name: "non array", body: `{}`, fallback: true, wantType: true},
		{name: "non array with credential", body: `{"access_token":"body-sensitive"}`,
			fallback: true, wantType: true},
		{name: "header syntax", body: `[2]`, header: "not-numeric", wantNum: true, wantCause: strconv.ErrSyntax},
		{name: "header overflow", body: `[2]`, header: strings.Repeat("9", 64),
			wantNum: true, wantCause: strconv.ErrRange},
		{name: "header credential", body: `[2]`, header: "password=header-sensitive; " + token,
			wantNum: true, wantCause: strconv.ErrSyntax},
		{name: "header zero", body: `[2]`, header: "0"},
		{name: "header negative", body: `[2]`, header: "-1"},
		{name: "inconsistent total", body: `[2]`, header: "3"},
		{name: "read partial", body: `[2]`, readCause: errors.New("password=read-sensitive; " + token)},
		{name: "read cancellation", body: `[2]`, readCause: context.Canceled},
		{name: "read deadline", body: `[2]`, readCause: context.DeadlineExceeded},
		{name: "read timeout", body: `[2]`, readCause: apiTimeoutError{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := &apiErrorBody{Reader: strings.NewReader(tt.body)}
			if tt.readCause != nil {
				body.Reader = io.MultiReader(body.Reader, failingReader{err: tt.readCause})
			}
			fixture := newAPITestFixture(token, func(req *http.Request) (*http.Response, error) {
				if req.URL.Query().Get("page") == "1" {
					resp := apiTestResponse(req, http.StatusOK, `[1]`)
					if !tt.fallback {
						resp.Header.Set("total_page", "2")
					}
					return resp, nil
				}
				resp := apiTestResponse(req, http.StatusOK, "")
				resp.Header.Set("total_page", tt.header)
				resp.Body = body
				return resp, nil
			})
			cmd := NewCmdAPI(fixture.factory)
			cmd.SetOut(fixture.stdout)
			cmd.SetErr(fixture.stderr)
			cmd.SetArgs([]string{"/items?per_page=1", "--paginate"})
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "page 2") {
				t.Fatalf("lost failure/page context: %v", err)
			}
			if errors.Is(err, internalapi.ErrInvalidResponse) != (tt.readCause == nil) {
				t.Fatalf("wrong invalid response role: %v", err)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Fatalf("lost parser cause: %v", err)
			}
			readErr, read := errors.AsType[*internalapi.ResponseReadError](err)
			if read != (tt.readCause != nil) {
				t.Fatalf("wrong read role: %v", err)
			}
			if read && (!errors.Is(err, tt.readCause) || !errors.Is(readErr.Err, tt.readCause)) {
				t.Fatalf("lost actual read cause: %v", err)
			}
			if read {
				wantTimeout := errors.Is(tt.readCause, context.DeadlineExceeded) || errors.Is(tt.readCause, apiTimeoutError{})
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
			if numErr, ok := errors.AsType[*strconv.NumError](err); ok != tt.wantNum {
				t.Fatalf("number cause = %v, want %v: %v", ok, tt.wantNum, err)
			} else if ok && numErr.Num != tt.header {
				t.Fatalf("lost original header in parser cause: %#v", numErr)
			}
			assertNoTokenLeak(t, token, err.Error(), fixture.stderr.String())
			for _, secret := range []string{"body-sensitive", "header-sensitive", "read-sensitive"} {
				if strings.Contains(err.Error(), secret) {
					t.Fatalf("unsafe parser/read message: %v", err)
				}
			}
			if fixture.stdout.String() != "[1]\n" || fixture.requests != 2 || !body.closed {
				t.Fatalf("stdout = %q, calls = %d, closed = %v", fixture.stdout, fixture.requests, body.closed)
			}
		})
	}
}

func TestAPIPaginationQueryPreservesNumericCauseBeforeSideEffects(t *testing.T) {
	for _, key := range []string{"page", "per_page"} {
		for _, dryRun := range []bool{false, true} {
			for _, tt := range []struct {
				name  string
				value string
				cause error
			}{
				{name: "syntax", value: "query-sensitive", cause: strconv.ErrSyntax},
				{name: "overflow", value: strings.Repeat("9", 64), cause: strconv.ErrRange},
				{name: "zero", value: "0"},
				{name: "negative", value: "-1"},
			} {
				t.Run(fmt.Sprintf("%s/dry-run=%v/%s", key, dryRun, tt.name), func(t *testing.T) {
					fixture := newAPITestFixture("configured-token", nil)
					cmd := NewCmdAPI(fixture.factory)
					cmd.SetOut(fixture.stdout)
					cmd.SetErr(fixture.stderr)
					args := []string{"/items?" + key + "=" + tt.value, "--paginate"}
					if dryRun {
						args = append(args, "--dry-run")
					}
					cmd.SetArgs(args)
					err := cmd.Execute()
					if err == nil || !strings.Contains(err.Error(), "positive integer") {
						t.Fatalf("lost validation fact: %v", err)
					}
					if numErr, ok := errors.AsType[*strconv.NumError](err); ok != (tt.cause != nil) {
						t.Fatalf("numeric cause = %v, want %v: %v", ok, tt.cause != nil, err)
					} else if ok && (numErr.Num != tt.value || !errors.Is(err, tt.cause)) {
						t.Fatalf("lost original numeric cause: %v", err)
					}
					if errors.Is(err, internalapi.ErrInvalidResponse) || strings.Contains(err.Error(), "query-sensitive") {
						t.Fatalf("user input misclassified or leaked: %v", err)
					}
					if fixture.config.tokenReads != 0 || fixture.clientReads != 0 || fixture.requests != 0 {
						t.Fatal("validation read credentials, acquired a client, or sent a request")
					}
				})
			}
		}
	}
}

// apiFinalErrorReader can return response bytes and a terminal error together.
type apiFinalErrorReader struct {
	body string
	err  error
}

func (r *apiFinalErrorReader) Read(p []byte) (int, error) {
	n := copy(p, r.body)
	r.body = r.body[n:]
	if r.body == "" {
		return n, r.err
	}
	return n, nil
}

func TestAPIRawResponseReadContract(t *testing.T) {
	const token = "configured-token"
	privateCause := errors.New("password=reader-secret; " + token)
	for _, tt := range []struct {
		name      string
		body      string
		cause     error
		sameRead  bool
		wantError bool
	}{
		{name: "immediate failure", cause: privateCause, wantError: true},
		{name: "partial response", body: "partial", cause: privateCause, wantError: true},
		{name: "data and error together", body: "partial", cause: privateCause, sameRead: true, wantError: true},
		{name: "canceled", cause: context.Canceled, wantError: true},
		{name: "deadline", cause: context.DeadlineExceeded, wantError: true},
		{name: "network timeout", cause: apiTimeoutError{}, wantError: true},
		{name: "empty EOF", cause: io.EOF},
		{name: "data and EOF together", body: "complete", cause: io.EOF, sameRead: true},
		{name: "complete response", body: "complete", cause: io.EOF},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var reader io.Reader = io.MultiReader(strings.NewReader(tt.body), failingReader{err: tt.cause})
			if tt.sameRead {
				reader = &apiFinalErrorReader{body: tt.body, err: tt.cause}
			}
			body := &apiErrorBody{Reader: reader}
			fixture := newAPITestFixture(token, func(req *http.Request) (*http.Response, error) {
				resp := apiTestResponse(req, http.StatusOK, "")
				resp.Body = body
				return resp, nil
			})
			cmd := NewCmdAPI(fixture.factory)
			cmd.SetOut(fixture.stdout)
			cmd.SetErr(fixture.stderr)
			cmd.SetArgs([]string{"/items"})

			err := cmd.Execute()
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, want error = %v", err, tt.wantError)
			}
			if tt.wantError {
				readErr, ok := errors.AsType[*internalapi.ResponseReadError](err)
				if !ok || readErr.Err != tt.cause || !errors.Is(err, tt.cause) {
					t.Fatalf("lost response read identity or cause: %v", err)
				}
				wantTimeout := errors.Is(tt.cause, context.DeadlineExceeded) || errors.Is(tt.cause, apiTimeoutError{})
				if readErr.Timeout() != wantTimeout {
					t.Fatalf("timeout = %v, want %v", readErr.Timeout(), wantTimeout)
				}
				if errors.Is(err, internalapi.ErrInvalidResponse) {
					t.Fatalf("read failure classified as invalid response: %v", err)
				}
				assertNoTokenLeak(t, token, err.Error())
			}
			if fixture.stdout.String() != tt.body || fixture.requests != 1 || !body.closed {
				t.Fatalf("stdout = %q, calls = %d, closed = %v", fixture.stdout, fixture.requests, body.closed)
			}
		})
	}
}

func TestAPIRawResponseWriterFailureTakesPrecedence(t *testing.T) {
	readCause := errors.New("reader failed")
	writeCause := errors.New("writer failed")
	body := &apiErrorBody{Reader: &apiFinalErrorReader{body: "partial", err: readCause}}
	fixture := newAPITestFixture("configured-token", func(req *http.Request) (*http.Response, error) {
		resp := apiTestResponse(req, http.StatusOK, "")
		resp.Body = body
		return resp, nil
	})
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetOut(apiFailingWriter{err: writeCause})
	cmd.SetArgs([]string{"/items"})

	err := cmd.Execute()
	if !errors.Is(err, writeCause) || errors.Is(err, readCause) {
		t.Fatalf("lost copy writer failure precedence: %v", err)
	}
	if _, read := errors.AsType[*internalapi.ResponseReadError](err); read {
		t.Fatalf("writer failure incorrectly classified: %v", err)
	}
	if !body.closed {
		t.Fatal("response body was not closed")
	}
}

func TestAPIRawResponseCopyDoesNotMisclassifyWriterFailure(t *testing.T) {
	cause := errors.New("writer failed")
	fixture := newAPITestFixture("configured-token", func(req *http.Request) (*http.Response, error) {
		return apiTestResponse(req, http.StatusOK, "raw response"), nil
	})
	cmd := NewCmdAPI(fixture.factory)
	cmd.SetOut(apiFailingWriter{err: cause})
	cmd.SetArgs([]string{"/items"})
	err := cmd.Execute()
	if !errors.Is(err, cause) {
		t.Fatalf("lost writer cause: %v", err)
	}
	if _, read := errors.AsType[*internalapi.ResponseReadError](err); read ||
		errors.Is(err, internalapi.ErrInvalidResponse) {
		t.Fatalf("writer failure incorrectly classified: %v", err)
	}
}

func TestAPIHTTPTypeSurvivesShortTokenRedaction(t *testing.T) {
	for _, token := range []string{"b", "bo", "bob", "bob1"} {
		t.Run(token, func(t *testing.T) {
			resp := apiTestResponse(nil, http.StatusForbidden, `{"message":"denied `+token+`"}`)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Error(err)
				}
			}()
			err := responseError("GET /items", resp, token)
			if !internalapi.IsHTTPStatus(err, http.StatusForbidden) {
				t.Fatalf("redaction lost HTTP type: %v", err)
			}
			assertNoTokenLeak(t, token, err.Error())
		})
	}
}
