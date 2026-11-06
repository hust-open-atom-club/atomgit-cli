package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestListRepositoryCLAs(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid      bool
		count      int
	}{
		{"agreements", `{"data":[{"id":"cla-123","name":"Contributor Agreement"}]}`, true, 1},
		{"empty", `{"data":[]}`, true, 0},
		{"missing", `{}`, false, 0}, {"null", `{"data":null}`, false, 0},
		{"wrong wrapper", `[]`, false, 0}, {"malformed", `{`, false, 0},
		{"missing identity", `{"data":[{"name":"CLA"}]}`, false, 0},
		{"numeric ID", `{"data":[{"id":1,"name":"CLA"}]}`, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.EscapedPath() != "/repos/team%20space/demo%23/clas" {
					t.Fatalf("request %s %s", r.Method, r.URL)
				}
				_, _ = io.WriteString(w, tc.body)
			})
			got, err := ListRepositoryCLAs(client, "team space", "demo#")
			if (err == nil) != tc.valid {
				t.Fatalf("result=%v error=%v", got, err)
			}
			if tc.valid && (got == nil || len(got) != tc.count) {
				t.Fatalf("result=%v", got)
			}
		})
	}
}

func TestRepositoryCLAWrites(t *testing.T) {
	yes, no := true, false
	for _, tc := range []struct {
		name    string
		enforce *bool
		disable bool
		want    map[string]any
	}{
		{"omitted check", nil, false, map[string]any{"cla_id": "opaque-id"}},
		{"true", &yes, false, map[string]any{"cla_id": "opaque-id", "enforce_check": true}},
		{"false", &no, false, map[string]any{"cla_id": "opaque-id", "enforce_check": false}},
		{"disable", nil, true, map[string]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "PUT" || r.URL.EscapedPath() != "/repos/team%20space/demo%23/cla" || r.Header.Get("Content-Type") != "application/json" {
					t.Fatalf("request %s %s headers=%v", r.Method, r.URL, r.Header)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(body, tc.want) {
					t.Fatalf("body=%v want=%v", body, tc.want)
				}
				w.WriteHeader(200) // The documented response has no JSON body.
			})
			var err error
			if tc.disable {
				err = DisableRepositoryCLA(client, "team space", "demo#")
			} else {
				err = SetRepositoryCLA(client, "team space", "demo#", "opaque-id", tc.enforce)
			}
			if err != nil || calls != 1 {
				t.Fatalf("error=%v calls=%d", err, calls)
			}
		})
	}
}

func TestRepositoryCLAWriteFailuresDoNotReplay(t *testing.T) {
	for _, status := range []int{201, 301, 302, 303, 307, 308, 403, 404, 429, 500, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			client := NewClientWithBaseURL("test", "https://example.test/api/v5", &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls > 1 {
					t.Fatal("write replayed")
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://other.test/cla"}, "Retry-After": {"0"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			})})
			err := DisableRepositoryCLA(client, "team", "demo")
			if !IsHTTPStatus(err, status) || calls != 1 {
				t.Fatalf("error=%v calls=%d", err, calls)
			}
		})
	}
	cause := errors.New("connection lost")
	client := NewClientWithHTTPClient("test", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, cause })})
	if err := SetRepositoryCLA(client, "team", "demo", "id", nil); !errors.Is(err, cause) {
		t.Fatalf("error=%v", err)
	}
}

func TestRepositoryCLARejectsEmptyID(t *testing.T) {
	client := NewClientWithHTTPClient("test", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("request sent with empty ID"); return nil, nil })})
	for _, id := range []string{"", " \t"} {
		if err := SetRepositoryCLA(client, "team", "demo", id, nil); err == nil {
			t.Fatal("accepted empty CLA ID")
		}
	}
}
