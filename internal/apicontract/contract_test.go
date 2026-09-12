package apicontract

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
)

func sample() Fixture {
	item := Shape{Types: []string{"object"}, Required: []string{"id"}, Properties: map[string]Shape{"id": {Types: []string{"integer"}}, "label": {Types: []string{"string", "null"}}}}
	return Fixture{Name: "sample", Source: "synthetic", Version: "v8", Method: "GET", Path: "/repos/{owner}/{repo}/actions/workflows", Statuses: []int{200}, BodyPolicy: "required", HeaderTypes: map[string]string{"X-Page": "integer"}, Shape: &Shape{Types: []string{"object"}, Required: []string{"total_count", "items"}, Properties: map[string]Shape{"total_count": {Types: []string{"integer"}}, "items": {Types: []string{"array"}, Items: &item}}}, Examples: []Example{{Name: "page", Status: 200, Query: map[string]string{"page": "1", "per_page": "2"}, Body: json.RawMessage(`{"total_count":1,"items":[{"id":1}]}`)}}}
}

func TestContractDetectsDrift(t *testing.T) {
	tests := []struct {
		name, body string
		status     int
		header     string
		want       string
	}{
		{"valid", `{"total_count":1,"items":[{"id":1}]}`, 200, "1", ""},
		{"empty page", `{"total_count":0,"items":[]}`, 200, "", ""},
		{"nullable optional", `{"total_count":1,"items":[{"id":1,"label":null}]}`, 200, "", ""},
		{"additive field", `{"total_count":1,"items":[{"id":1,"future":true}],"new":42}`, 200, "", ""},
		{"unexpected success", `{}`, 201, "", "unexpected HTTP status 201"},
		{"failure", `{"access_token":"sensitive-canary"}`, 403, "", "unexpected HTTP status 403"},
		{"missing body", "", 200, "", "required response body is empty"},
		{"whitespace body", "  \n", 200, "", "required response body is empty"},
		{"envelope becomes array", `[]`, 200, "", "incompatible JSON type"},
		{"missing envelope", `{"total_count":1,"data":[]}`, 200, "", "required field missing"},
		{"null page", `{"total_count":0,"items":null}`, 200, "", "incompatible JSON type"},
		{"bad count", `{"total_count":"sensitive-canary","items":[]}`, 200, "", "incompatible JSON type"},
		{"bad item", `{"total_count":1,"items":[{"id":"sensitive-canary"}]}`, 200, "", "incompatible JSON type"},
		{"missing id", `{"total_count":1,"items":[{}]}`, 200, "", "required field missing"},
		{"bad optional field", `{"total_count":1,"items":[{"id":1,"label":false}]}`, 200, "", "incompatible JSON type"},
		{"fractional id", `{"total_count":1,"items":[{"id":1.5}]}`, 200, "", "incompatible JSON type"},
		{"trailing data", `{} sensitive-canary`, 200, "", "invalid JSON response"},
		{"malformed", `{"secret":"sensitive-canary`, 200, "", "invalid JSON response"},
		{"pagination header", `{}`, 200, "sensitive-canary", "expected non-negative integer"},
		{"oversize", strings.Repeat("x", MaxBodyBytes+1), 200, "", "size limit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			if tt.header != "" {
				h.Set("X-Page", tt.header)
			}
			err := sample().Validate(tt.status, h, []byte(tt.body))
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %s, got %v", tt.want, err)
			}
			if strings.Contains(err.Error(), "sensitive-canary") {
				t.Fatal("diagnostic leaked response data")
			}
		})
	}
}

func TestBodyPolicies(t *testing.T) {
	for _, policy := range []string{"required", "optional", "empty"} {
		for _, body := range []string{"", "{}", "null"} {
			t.Run(policy+"/"+body, func(t *testing.T) {
				f := sample()
				f.BodyPolicy = policy
				f.Shape = &Shape{Types: []string{"object"}}
				want := (body == "" && policy != "required") || (body == "{}" && policy != "empty")
				if got := f.Validate(200, nil, []byte(body)) == nil; got != want {
					t.Fatalf("accepted=%t, want %t", got, want)
				}
			})
		}
	}
}

func TestMalformedFixtures(t *testing.T) {
	tests := map[string]func(*Fixture){
		"version":            func(f *Fixture) { f.Version = "v9" },
		"method":             func(f *Fixture) { f.Method = "TRACE" },
		"source":             func(f *Fixture) { f.Source = "" },
		"path boundary":      func(f *Fixture) { f.Path = "/repos/{owner}/{repo}-other" },
		"ignored properties": func(f *Fixture) { f.Shape.Types = []string{"string"} },
		"ignored items":      func(f *Fixture) { f.Shape.Items = &Shape{Types: []string{"string"}} },
		"origin":             func(f *Fixture) { f.Path = "https://example.com" },
		"traversal":          func(f *Fixture) { f.Path = "/repos/{owner}/{repo}/../user" },
		"credential query":   func(f *Fixture) { f.Examples[0].Query = map[string]string{"access_token": "synthetic"} },
		"negative page":      func(f *Fixture) { f.Examples[0].Query = map[string]string{"page": "-1"} },
		"statuses":           func(f *Fixture) { f.Statuses = nil },
		"failure status":     func(f *Fixture) { f.Statuses = []int{500} },
		"body policy":        func(f *Fixture) { f.BodyPolicy = "anything" },
		"schema":             func(f *Fixture) { f.Shape = nil },
		"unknown type":       func(f *Fixture) { f.Shape.Types = []string{"map"} },
		"required":           func(f *Fixture) { f.Shape.Required = []string{"missing"} },
		"array":              func(f *Fixture) { f.Shape = &Shape{Types: []string{"array"}} },
		"headers":            func(f *Fixture) { f.Examples[0].Headers = map[string]string{"Authorization": "synthetic"} },
		"examples":           func(f *Fixture) { f.Examples = nil },
		"duplicate examples": func(f *Fixture) { f.Examples = append(f.Examples, f.Examples[0]) },
		"example drift":      func(f *Fixture) { f.Examples[0].Body = json.RawMessage(`[]`) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			f := sample()
			mutate(&f)
			data, err := json.Marshal(f)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Load(fstest.MapFS{"sample.json": {Data: data}}, "*.json"); err == nil {
				t.Fatal("malformed fixture accepted")
			}
		})
	}
	for _, data := range []string{`{}`, `{"unknown":true}`, `{} {}`} {
		if _, err := Load(fstest.MapFS{"sample.json": {Data: []byte(data)}}, "*.json"); err == nil {
			t.Fatal("malformed JSON accepted")
		}
	}
	data, _ := json.Marshal(sample())
	if _, err := Load(fstest.MapFS{"a.json": {Data: data}, "b.json": {Data: data}}, "*.json"); err == nil {
		t.Fatal("duplicate fixtures accepted")
	}
	if _, err := Load(fstest.MapFS{}, "*.json"); err == nil {
		t.Fatal("empty suite accepted")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestProbeSafety(t *testing.T) {
	f := sample()
	calls := 0
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != "GET" || req.URL.Host != "api.atomgit.com" || req.URL.Scheme != "https" || req.Header.Get("Authorization") != "Bearer dedicated-canary" {
			t.Fatal("unsafe probe request")
		}
		if strings.Contains(req.URL.String(), "dedicated-canary") {
			t.Fatal("credential in URL")
		}
		if _, ok := req.Context().Deadline(); !ok {
			t.Fatal("no probe timeout")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(f.Examples[0].Body)))}, nil
	})
	if err := Probe(t.Context(), transport, f, "fixture-owner", "fixture-repo", "dedicated-canary"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("expected a single bounded request")
	}
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		f.Method = method
		if err := Probe(t.Context(), transport, f, "fixture-owner", "fixture-repo", "dedicated-canary"); err == nil {
			t.Fatal("mutation allowed")
		}
	}
	f.Method = "GET"
	for _, owner := range []string{"..", "owner@example.com", "owner?token=secret", "owner/other"} {
		if err := Probe(t.Context(), transport, f, owner, "fixture-repo", "dedicated-canary"); err == nil {
			t.Fatal("unsafe repository allowed")
		}
	}
	if calls != 1 {
		t.Fatal("invalid probe reached transport")
	}
}

func TestProbeDiscardsSensitiveFailures(t *testing.T) {
	for _, kind := range []string{"transport", "response", "type", "oversize", "redirect", "read"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				status := 200
				body := `{"total_count":"private-canary","items":[]}`
				h := http.Header{}
				switch kind {
				case "transport":
					return nil, errors.New("https://private-canary/?access_token=private-canary")
				case "response":
					status = 401
					body = `{"access_token":"private-canary"}`
				case "oversize":
					body = strings.Repeat("x", MaxBodyBytes+1)
				case "redirect":
					status = 302
					h.Set("Location", "https://example.com/?token=private-canary")
				case "read":
					return &http.Response{StatusCode: 200, Header: h, Body: failingBody{}}, nil
				}
				return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			err := Probe(t.Context(), transport, sample(), "fixture-owner", "fixture-repo", "private-canary")
			if err == nil || strings.Contains(err.Error(), "private-canary") {
				t.Fatalf("unsafe or missing error: %v", err)
			}
			if calls != 1 {
				t.Fatal("probe followed a redirect or retried")
			}
		})
	}
}

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("private-canary") }
func (failingBody) Close() error             { return nil }

func TestProbeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Context().Err() == nil {
			t.Fatal("cancelled context not propagated")
		}
		return nil, r.Context().Err()
	})
	if err := Probe(ctx, transport, sample(), "fixture-owner", "fixture-repo", "dedicated-canary"); err == nil {
		t.Fatal("cancelled request succeeded")
	}
}

func TestReplayChecksRequestsAndConsumption(t *testing.T) {
	for _, change := range []string{"method", "version", "path", "query", "host", "none"} {
		t.Run(change, func(t *testing.T) {
			f := sample()
			replay := &Replay{Fixture: f, Examples: f.Examples}
			target, _ := f.URL("fixture-owner", "fixture-repo", f.Examples[0].Query)
			req, _ := http.NewRequest("GET", target, nil)
			switch change {
			case "method":
				req.Method = "POST"
			case "version":
				req.URL.Path = strings.Replace(req.URL.Path, "v8", "v5", 1)
			case "path":
				req.URL.Path += "/other"
			case "query":
				req.URL.RawQuery = "page=2"
			case "host":
				req.URL.Host = "example.com"
			}
			if replay.Check() == nil {
				t.Fatal("unconsumed fixture passed")
			}
			resp, err := replay.RoundTrip(req)
			if change != "none" {
				if err == nil {
					t.Fatal("mismatched request accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if err := replay.Check(); err != nil {
				t.Fatal(err)
			}
			if _, err := replay.RoundTrip(req); err == nil {
				t.Fatal("extra request accepted")
			}
		})
	}
}
