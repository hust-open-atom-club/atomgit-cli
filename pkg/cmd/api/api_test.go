package api

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestAPICommandEndToEnd(t *testing.T) {
	temp := t.TempDir()
	inputFile := filepath.Join(temp, "body.bin")
	if err := os.WriteFile(inputFile, []byte("file-body"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args []string
		in   string
		want string
	}{
		{name: "basic GET", args: []string{"/user"}, want: `{"login":"alice"}`},
		{name: "GET fields", args: []string{"/items", "-f", "q=中文 space", "-f", "empty="}, want: "get-fields"},
		{name: "POST fields", args: []string{"/items", "-X", "POST", "-f", "name=demo"}, want: "POST"},
		{name: "PATCH file", args: []string{"/items/1", "-X", "PATCH", "--input", inputFile}, want: "PATCH"},
		{name: "PUT stdin", args: []string{"/items/1", "-X", "PUT", "--input", "-"}, in: "stdin-body", want: "PUT"},
		{name: "DELETE", args: []string{"/items/1", "-X", "DELETE"}, want: "DELETE"},
		{name: "empty", args: []string{"/empty"}},
		{name: "binary", args: []string{"/binary"}, want: "\x00\xff"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Authorization") != "Bearer secret" {
					t.Fatalf("Authorization = %q", req.Header.Get("Authorization"))
				}
				switch req.URL.Path {
				case "/api/v5/user":
					return apiTestResponse(req, http.StatusOK, `{"login":"alice"}`), nil
				case "/api/v5/empty":
					return apiTestResponse(req, http.StatusNoContent, ""), nil
				case "/api/v5/binary":
					return apiTestResponse(req, http.StatusOK, "\x00\xff"), nil
				}
				if req.Method == http.MethodGet {
					if req.URL.Query().Get("q") != "中文 space" {
						t.Fatalf("query = %q", req.URL.RawQuery)
					}
					return apiTestResponse(req, http.StatusOK, "get-fields"), nil
				}
				var body []byte
				if req.Body != nil {
					var err error
					body, err = io.ReadAll(req.Body)
					if err != nil {
						t.Fatal(err)
					}
				}
				if req.Method == http.MethodPatch && !bytes.Equal(body, []byte("file-body")) {
					t.Fatalf("file body = %q", body)
				}
				if req.Method == http.MethodPut && !bytes.Equal(body, []byte("stdin-body")) {
					t.Fatalf("stdin body = %q", body)
				}
				return apiTestResponse(req, http.StatusOK, req.Method), nil
			})
			fixture.stdin.WriteString(tt.in)
			cmd := NewCmdAPI(fixture.factory)
			cmd.SetIn(fixture.stdin)
			cmd.SetOut(fixture.stdout)
			cmd.SetArgs(tt.args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if got := fixture.stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("pagination and later failure", func(t *testing.T) {
		requests := 0
		fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
			requests++
			page, _ := strconv.Atoi(req.URL.Query().Get("page"))
			if page == 3 {
				return apiTestResponse(req, http.StatusBadGateway, "failed"), nil
			}
			resp := apiTestResponse(req, http.StatusOK, "["+strconv.Itoa(page)+"]")
			resp.Header.Set("total_page", "3")
			return resp, nil
		})
		cmd := NewCmdAPI(fixture.factory)
		cmd.SetOut(fixture.stdout)
		cmd.SetArgs([]string{"/items", "--paginate"})
		if err := cmd.Execute(); err == nil {
			t.Fatal("expected later page error")
		}
		if fixture.stdout.String() != "[1]\n[2]\n" || requests != 3 {
			t.Fatalf("requests = %d stdout = %q", requests, fixture.stdout.String())
		}
	})

	t.Run("redirect credentials", func(t *testing.T) {
		requests := 0
		fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
			requests++
			if requests == 1 {
				resp := apiTestResponse(req, http.StatusFound, "")
				resp.Header.Set("Location", "https://other.test/next")
				return resp, nil
			}
			if req.Header.Get("Authorization") != "" {
				t.Fatal("Authorization followed cross-origin redirect")
			}
			return apiTestResponse(req, http.StatusOK, "redirected"), nil
		})
		cmd := NewCmdAPI(fixture.factory)
		cmd.SetOut(fixture.stdout)
		cmd.SetArgs([]string{"/redirect"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if fixture.stdout.String() != "redirected" {
			t.Fatalf("stdout = %q", fixture.stdout.String())
		}
	})
}
