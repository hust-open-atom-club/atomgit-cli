package key

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestSSHKeyDeleteConfirmedAndYes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		yes   bool
	}{
		{name: "confirmed with y", input: "y\n"},
		{name: "confirmed with yes", input: "yes\n"},
		{name: "yes flag", yes: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var methods []string
			factory := sshKeyFactory(sshKeyTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
				methods = append(methods, req.Method+" "+req.URL.Path)
				if got := req.Header.Get("Authorization"); got != "Bearer token" {
					t.Fatalf("Authorization = %q", got)
				}
				switch req.Method {
				case http.MethodGet:
					return sshKeyResponse(http.StatusOK, `{"id":7,"title":"Work Laptop","key":"ssh-ed25519 AQIDBA=="}`), nil
				case http.MethodDelete:
					return sshKeyResponse(http.StatusNoContent, ""), nil
				default:
					t.Fatalf("unexpected request = %s %s", req.Method, req.URL.Path)
					return nil, nil
				}
			})
			cmd := newCmdSSHKeyDelete(factory)
			if tt.yes {
				_ = cmd.Flags().Set("yes", "true")
			}
			cmd.SetIn(strings.NewReader(tt.input))
			var out bytes.Buffer
			cmd.SetOut(&out)

			if err := cmd.RunE(cmd, []string{"7"}); err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(methods, ","); got != "GET /api/v5/user/keys/7,DELETE /api/v5/user/keys/7" {
				t.Fatalf("methods = %s", got)
			}
			if !strings.Contains(out.String(), "Deleted SSH key 7 (Work Laptop).") {
				t.Fatalf("output = %s", out.String())
			}
			if !tt.yes {
				for _, want := range []string{"Work Laptop", "SHA256:n2SnR+G5fxMfq7a0Rylsm28CAeefs8U1bmx36JtqgGo"} {
					if !strings.Contains(out.String(), want) {
						t.Fatalf("prompt missing %q: %s", want, out.String())
					}
				}
			}
		})
	}
}

func TestSSHKeyDeleteCancellationDoesNotDelete(t *testing.T) {
	deletes := 0
	cmd := newCmdSSHKeyDelete(sshKeyFactory(sshKeyTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodDelete {
			deletes++
		}
		return sshKeyResponse(http.StatusOK, `{"id":7,"title":"Work Laptop","fingerprint":"SHA256:test"}`), nil
	}))
	cmd.SetIn(strings.NewReader("no\n"))
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.RunE(cmd, []string{"7"}); err != nil {
		t.Fatal(err)
	}
	if deletes != 0 {
		t.Fatalf("delete requests = %d", deletes)
	}
	if !strings.Contains(out.String(), "cancelled") {
		t.Fatalf("output = %s", out.String())
	}
}

func TestSSHKeyDeleteRejectsInvalidIDBeforeRequest(t *testing.T) {
	for _, id := range []string{"0", "-1", "abc"} {
		t.Run(id, func(t *testing.T) {
			requests := 0
			cmd := newCmdSSHKeyDelete(sshKeyFactory(sshKeyTestConfig{token: "token"}, func(*http.Request) (*http.Response, error) {
				requests++
				return sshKeyResponse(http.StatusOK, `{}`), nil
			}))
			err := cmd.RunE(cmd, []string{id})
			if err == nil || !strings.Contains(err.Error(), "positive integer") {
				t.Fatalf("error = %v", err)
			}
			if requests != 0 {
				t.Fatalf("requests = %d", requests)
			}
		})
	}
}

func TestSSHKeyDeleteReportsGetAndDeleteErrors(t *testing.T) {
	tests := []struct {
		name       string
		failMethod string
		status     int
		want       string
	}{
		{name: "not found", failMethod: http.MethodGet, status: http.StatusNotFound, want: "failed to get SSH key 7"},
		{name: "get forbidden", failMethod: http.MethodGet, status: http.StatusForbidden, want: "failed to get SSH key 7"},
		{name: "delete forbidden", failMethod: http.MethodDelete, status: http.StatusForbidden, want: "failed to delete SSH key 7"},
		{name: "delete server error", failMethod: http.MethodDelete, status: http.StatusInternalServerError, want: "failed to delete SSH key 7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdSSHKeyDelete(sshKeyFactory(sshKeyTestConfig{token: "token"}, func(req *http.Request) (*http.Response, error) {
				if req.Method == tt.failMethod {
					return sshKeyResponse(tt.status, `{"message":"error"}`), nil
				}
				return sshKeyResponse(http.StatusOK, `{"id":7,"title":"Work Laptop","fingerprint":"SHA256:test"}`), nil
			}))
			_ = cmd.Flags().Set("yes", "true")
			err := cmd.RunE(cmd, []string{"7"})
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), fmt.Sprint(tt.status)) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestSSHKeyCommandsReportAuthenticationErrorsWithoutRequest(t *testing.T) {
	requests := 0
	factory := sshKeyFactory(sshKeyTestConfig{tokenErr: errors.New("missing token")}, func(*http.Request) (*http.Response, error) {
		requests++
		return sshKeyResponse(http.StatusOK, `{}`), nil
	})

	list := newCmdSSHKeyList(factory)
	if err := list.RunE(list, nil); err == nil || !strings.Contains(err.Error(), "missing token") {
		t.Fatalf("list error = %v", err)
	}
	deleteCmd := newCmdSSHKeyDelete(factory)
	_ = deleteCmd.Flags().Set("yes", "true")
	if err := deleteCmd.RunE(deleteCmd, []string{"7"}); err == nil || !strings.Contains(err.Error(), "missing token") {
		t.Fatalf("delete error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d", requests)
	}
}
