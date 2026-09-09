package key

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestSSHKeyListHandlesZeroOneAndMultipleKeys(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		contains []string
	}{
		{name: "zero", body: `[]`, contains: []string{"No SSH keys found."}},
		{
			name: "one",
			body: `[{
				"id": 7,
				"title": "Work Laptop",
				"key": "ssh-ed25519 AQIDBA== secret-comment",
				"created_at": "2026-07-20T12:00:00Z"
			}]`,
			contains: []string{"ID", "TITLE", "FINGERPRINT", "CREATED", "7", "Work Laptop", "SHA256:n2SnR+G5fxMfq7a0Rylsm28CAeefs8U1bmx36JtqgGo", "2026-07-20T12:00:00Z"},
		},
		{
			name: "multiple and API fingerprint",
			body: `[
				{"id": 7, "title": "First", "key": "ssh-ed25519 AQIDBA==", "created_at": "2026-07-20T12:00:00Z"},
				{"id": 8, "title": "Second", "key": "not-a-public-key", "fingerprint": "SHA256:from-api", "created_at": "2026-07-21T12:00:00Z"}
			]`,
			contains: []string{"7", "First", "8", "Second", "SHA256:from-api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := sshKeyFactory(sshKeyTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodGet || req.URL.Path != "/api/v5/user/keys" {
					t.Fatalf("request = %s %s", req.Method, req.URL.Path)
				}
				if req.URL.Query().Get("page") != "1" || req.URL.Query().Get("per_page") != "100" {
					t.Fatalf("query = %q", req.URL.RawQuery)
				}
				if got := req.Header.Get("Authorization"); got != "Bearer token" {
					t.Fatalf("Authorization = %q", got)
				}
				return sshKeyResponse(http.StatusOK, tt.body), nil
			})
			cmd := newCmdSSHKeyList(factory)
			var out bytes.Buffer
			cmd.SetOut(&out)

			if err := cmd.RunE(cmd, nil); err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests = %d", requests)
			}
			for _, want := range tt.contains {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("output missing %q:\n%s", want, out.String())
				}
			}
			if strings.Contains(out.String(), "AQIDBA==") || strings.Contains(out.String(), "secret-comment") {
				t.Fatalf("output leaked public key material:\n%s", out.String())
			}
		})
	}
}

func TestSSHKeyListPaginatesAndHonorsLimit(t *testing.T) {
	requests := 0
	factory := sshKeyFactory(sshKeyTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Query().Get("page") != fmt.Sprint(requests) || req.URL.Query().Get("per_page") != "100" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		count := 100
		start := 1
		if requests == 2 {
			count = 2
			start = 101
		}
		keys := make([]map[string]any, count)
		for i := range keys {
			keys[i] = map[string]any{"id": start + i, "title": fmt.Sprintf("key-%d", start+i), "fingerprint": "SHA256:test"}
		}
		body, err := json.Marshal(keys)
		if err != nil {
			t.Fatal(err)
		}
		return sshKeyResponse(http.StatusOK, string(body)), nil
	})
	cmd := newCmdSSHKeyList(factory)
	if err := cmd.Flags().Set("limit", "101"); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d", requests)
	}
	if !strings.Contains(out.String(), "key-101") || strings.Contains(out.String(), "key-102") {
		t.Fatalf("output did not honor limit:\n%s", out.String())
	}
}

func TestSSHKeyListRejectsInvalidLimitBeforeRequest(t *testing.T) {
	requests := 0
	cmd := newCmdSSHKeyList(sshKeyFactory(sshKeyTestConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
		requests++
		return sshKeyResponse(http.StatusOK, `[]`), nil
	}))
	if err := cmd.Flags().Set("limit", "0"); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestSSHKeyListReportsAPIErrors(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cmd := newCmdSSHKeyList(sshKeyFactory(sshKeyTestConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
				return sshKeyResponse(status, `{"message":"error"}`), nil
			}))
			err := cmd.RunE(cmd, nil)
			if err == nil || !strings.Contains(err.Error(), "failed to list SSH keys") || !strings.Contains(err.Error(), fmt.Sprint(status)) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
