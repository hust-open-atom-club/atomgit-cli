package comment

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestListOutputsRowsAndPaginates(t *testing.T) {
	requests := 0
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet {
			t.Fatalf("method = %s", req.Method)
		}
		if got := req.URL.EscapedPath(); got != "/api/v5/repos/alice/demo/commits/feature%2Fone/comments" {
			t.Fatalf("path = %q", got)
		}
		if got := req.URL.Query().Get("per_page"); got != "100" {
			t.Fatalf("per_page = %q", got)
		}
		if got := req.URL.Query().Get("page"); got != fmt.Sprint(requests) {
			t.Fatalf("page = %q, want %d", got, requests)
		}
		if requests == 1 {
			return jsonResponse(req, http.StatusOK, commentsPage(1, 100)), nil
		}
		return jsonResponse(req, http.StatusOK, commentsPage(101, 2)), nil
	})

	var out bytes.Buffer
	cmd := newCmdList(newFactory(t, &testConfig{}, transport))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "feature/one", "--limit", "102"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 102 {
		t.Fatalf("lines = %d, want 102", len(lines))
	}
	if lines[0] != "1\talice\t2026-09-15 10:00\t(你) note 1" {
		t.Fatalf("first line = %q", lines[0])
	}
	if lines[101] != "102\talice\t2026-09-15 10:00\t(你) note 102" {
		t.Fatalf("last line = %q", lines[101])
	}
}

func TestListMarksCurrentUserAndEscapesCells(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(req, http.StatusOK, `[{"id":5,"body":"multi\nline\tbody","user":{"login":"alice"},"created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T10:00:00+08:00"}]`), nil
	})

	var out bytes.Buffer
	cmd := newCmdList(newFactory(t, &testConfig{}, transport))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "abc1234"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := "5\talice\t2026-09-15 10:00\t(你) multi\\nline\\tbody\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestListJSONOutputIsStable(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(req, http.StatusOK, `[{"id":9,"body":"hello","user":{"id":99,"login":"bob","name":"Bob"},"created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T11:00:00+08:00"}]`), nil
	})

	var out bytes.Buffer
	cmd := newCmdList(newFactory(t, &testConfig{}, transport))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "abc1234", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := `[
  {
    "id": "9",
    "body": "hello",
    "author": "bob",
    "created_at": "2026-09-15T10:00:00+08:00",
    "updated_at": "2026-09-15T11:00:00+08:00"
  }
]
`
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestListEmptyPrintsPlaceholder(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(req, http.StatusOK, `[]`), nil
	})

	var out bytes.Buffer
	cmd := newCmdList(newFactory(t, &testConfig{}, transport))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"alice/demo", "abc1234"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "No comments found.\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestListRejectsBadInputBeforeNetwork(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing args", args: []string{}, want: "accepts between 1 and 2 arg(s)"},
		{name: "zero limit", args: []string{"alice/demo", "abc1234", "--limit", "0"}, want: "invalid limit"},
		{name: "negative limit", args: []string{"alice/demo", "abc1234", "--limit", "-1"}, want: "invalid limit"},
		{name: "invalid ref", args: []string{"alice/demo", "bad~ref"}, want: "invalid commit ref"},
		{name: "empty ref", args: []string{"alice/demo", ""}, want: "cannot be empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &testConfig{}
			transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errNoRequests
			})
			cmd := newCmdList(newFactory(t, cfg, transport))
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
			if cfg.tokenCalls != 0 {
				t.Fatalf("token fetched %d times before validation failed", cfg.tokenCalls)
			}
		})
	}
}

func commentsPage(first, count int) string {
	items := make([]string, count)
	for i := range count {
		items[i] = fmt.Sprintf(`{"id":%d,"body":"note %d","user":{"id":1001,"login":"alice"},"created_at":"2026-09-15T10:00:00+08:00","updated_at":"2026-09-15T10:00:00+08:00"}`, first+i, first+i)
	}
	return "[" + strings.Join(items, ",") + "]"
}
