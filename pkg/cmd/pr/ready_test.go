package pr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestPRReadyChangesDraftState(t *testing.T) {
	for _, tt := range []struct {
		name  string
		undo  bool
		start bool
	}{
		{name: "ready", start: true},
		{name: "undo", undo: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reads, writes := 0, 0
			factory := &cmdutil.Factory{
				Config: prTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						if req.URL.Path != "/api/v5/repos/alice/demo/pulls/42" || req.URL.RawQuery != "" || req.Header.Get("Authorization") != "Bearer token" {
							t.Fatalf("request = %s %s", req.Method, req.URL)
						}
						switch req.Method {
						case http.MethodGet:
							reads++
							draft := tt.start
							if writes > 0 {
								draft = tt.undo
							}
							return prResponse(http.StatusOK, fmt.Sprintf(`{"draft":%t,"state":"open","html_url":"https://atomgit.com/alice/demo/pull/42"}`, draft)), nil
						case http.MethodPatch:
							writes++
							var body map[string]any
							if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
								t.Fatal(err)
							}
							if len(body) != 1 || body["draft"] != tt.undo {
								t.Fatalf("body = %v", body)
							}
							return prResponse(http.StatusOK, `{}`), nil
						default:
							t.Fatalf("method = %s", req.Method)
							return nil, nil
						}
					})}, nil
				},
			}
			cmd := newCmdPRReady(factory)
			if tt.undo {
				_ = cmd.Flags().Set("undo", "true")
			}
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
				t.Fatal(err)
			}
			if reads != 2 || writes != 1 || !strings.Contains(out.String(), "PR #42") {
				t.Fatalf("reads=%d writes=%d output=%q", reads, writes, out.String())
			}
		})
	}
}

func TestPRReadySkipsWriteWhenAlreadyInTargetState(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{Config: prTestConfig{}, HttpClient: func() (*http.Client, error) {
		return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			if req.Method != http.MethodGet {
				t.Fatalf("unexpected %s", req.Method)
			}
			return prResponse(http.StatusOK, `{"draft":false,"state":"open"}`), nil
		})}, nil
	}}
	cmd := newCmdPRReady(factory)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil || requests != 1 {
		t.Fatalf("err=%v requests=%d", err, requests)
	}
}

func TestPRReadyRejectsInvalidStateBeforeWrite(t *testing.T) {
	for _, tt := range []struct {
		name, body, want string
	}{
		{"closed", `{"draft":true,"state":"closed"}`, "not open"},
		{"merged", `{"draft":true,"state":"merged"}`, "already merged"},
		{"missing draft", `{"state":"open"}`, "no draft status"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{Config: prTestConfig{}, HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodGet {
						t.Fatalf("unexpected %s", req.Method)
					}
					return prResponse(http.StatusOK, tt.body), nil
				})}, nil
			}}
			cmd := newCmdPRReady(factory)
			if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestPRReadyRejectsUnchangedReadback(t *testing.T) {
	factory := &cmdutil.Factory{Config: prTestConfig{}, HttpClient: func() (*http.Client, error) {
		return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPatch {
				return prResponse(http.StatusOK, `{}`), nil
			}
			return prResponse(http.StatusOK, `{"draft":true,"state":"open"}`), nil
		})}, nil
	}}
	cmd := newCmdPRReady(factory)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err == nil || !strings.Contains(err.Error(), "did not change") {
		t.Fatalf("error = %v", err)
	}
}
