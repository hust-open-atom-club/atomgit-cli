package api

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestAPIFieldsEncodeQueryAndJSON(t *testing.T) {
	t.Run("GET query", func(t *testing.T) {
		fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
			query := req.URL.Query()
			if got := query["existing"]; len(got) != 2 || got[0] != "first" || got[1] != "second" {
				t.Fatalf("existing = %#v", got)
			}
			if query.Get("unicode") != "中文 /?" || query.Get("empty") != "" || query.Get("equals") != "a=b" {
				t.Fatalf("query = %#v", query)
			}
			return apiTestResponse(req, http.StatusOK, "ok"), nil
		})
		cmd := NewCmdAPI(fixture.factory)
		cmd.SetOut(fixture.stdout)
		cmd.SetArgs([]string{"/items?existing=first", "-f", "existing=second", "-f", "unicode=中文 /?", "-f", "empty=", "-f", "equals=a=b"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	})

	for _, method := range []string{"POST", "PATCH", "PUT", "DELETE"} {
		t.Run(method+" JSON", func(t *testing.T) {
			fixture := newAPITestFixture("secret", func(req *http.Request) (*http.Response, error) {
				if req.Method != method || req.Header.Get("Content-Type") != "application/json" {
					t.Fatalf("request = %s Content-Type %q", req.Method, req.Header.Get("Content-Type"))
				}
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal(err)
				}
				var fields map[string]string
				if err := json.Unmarshal(body, &fields); err != nil {
					t.Fatal(err)
				}
				if fields["name"] != "last" || fields["body"] != "a=b" {
					t.Fatalf("fields = %#v", fields)
				}
				return apiTestResponse(req, http.StatusOK, "ok"), nil
			})
			cmd := NewCmdAPI(fixture.factory)
			cmd.SetOut(fixture.stdout)
			cmd.SetArgs([]string{"/items", "-X", method, "-f", "name=first", "-f", "name=last", "-f", "body=a=b"})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMalformedFieldHasNoSideEffects(t *testing.T) {
	for _, field := range []string{"missing", "=empty"} {
		fixture := newAPITestFixture("secret", nil)
		cmd := NewCmdAPI(fixture.factory)
		cmd.SetArgs([]string{"/items", "-f", field})
		if err := cmd.Execute(); err == nil {
			t.Fatalf("field %q unexpectedly succeeded", field)
		}
		if fixture.config.tokenReads != 0 || fixture.clientReads != 0 || fixture.requests != 0 {
			t.Fatalf("field %q caused side effects", field)
		}
	}
}
