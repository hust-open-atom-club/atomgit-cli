package pr

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func TestPRCommitsTextOutput(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return prResponse(http.StatusOK, `[{"sha":"abc1234567890","html_url":"https://u","commit":{"message":"fix: resolve issue\n\nbody","author":{"name":"Alice","email":"a@b.com","date":"2024-06-15T10:30:00Z","login":"alice"}}},{"sha":"def5678","html_url":"https://u2","commit":{"message":"chore: cleanup","author":{"name":"Bob","email":"","date":"2024-06-14T08:00:00Z","login":""}}}]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRCommits(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(lines))
	}
	if !strings.Contains(lines[0], "abc1234") || !strings.Contains(lines[0], "fix: resolve issue") || !strings.Contains(lines[0], "alice") || !strings.Contains(lines[0], "2024-06-15 10:30:00Z") {
		t.Errorf("line 0 = %s", lines[0])
	}
	if !strings.Contains(lines[1], "def5678") || !strings.Contains(lines[1], "chore: cleanup") || !strings.Contains(lines[1], "Bob") {
		t.Errorf("line 1 = %s", lines[1])
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "UTC",
			input: "2026-08-05T10:00:00Z",
			want:  "2026-08-05 10:00:00Z",
		},
		{
			name:  "positive offset",
			input: "2026-08-05T10:00:00+08:00",
			want:  "2026-08-05 10:00:00+08:00",
		},
		{
			name:  "negative offset with fractional seconds",
			input: "2026-08-05T10:00:00.123456789-04:30",
			want:  "2026-08-05 10:00:00-04:30",
		},
		{
			name:    "invalid calendar date",
			input:   "2026-13-40T25:61:61Z",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimestamp(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseTimestamp(%q) = %q, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTimestamp(%q): %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("parseTimestamp(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPRCommitsJSON(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return prResponse(http.StatusOK, `[{"sha":"abc1234","html_url":"https://url","commit":{"message":"fix bug","author":{"name":"Alice","email":"alice@example.com","date":"2024-01-01T00:00:00Z","login":"alice"}}}]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRCommits(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal(output.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0]["sha"] != "abc1234" || items[0]["authoredAt"] != "2024-01-01T00:00:00Z" {
		t.Fatalf("items = %#v", items)
	}
	author, _ := items[0]["author"].(map[string]any)
	if author["login"] != "alice" || author["name"] != "Alice" || author["email"] != "alice@example.com" {
		t.Fatalf("author = %#v", author)
	}
	if items[0]["url"] != "https://url" {
		t.Fatalf("url = %v", items[0]["url"])
	}
}

func TestPRCommitsEmptyResponse(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return prResponse(http.StatusOK, `[]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRCommits(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	trimmed := strings.TrimSpace(output.String())
	if trimmed != "[]" {
		t.Fatalf("output = %q, want []", trimmed)
	}
}

func TestPRCommitsPagination(t *testing.T) {
	var pageQuery string
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				pageQuery = req.URL.RawQuery
				return prResponse(http.StatusOK, `[]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRCommits(factory)
	_ = cmd.Flags().Set("limit", "50")
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pageQuery, "page=1") || !strings.Contains(pageQuery, "per_page=100") {
		t.Errorf("pageQuery = %s", pageQuery)
	}
}

func TestPRCommitsInvalidLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit string
	}{
		{name: "zero", limit: "0"},
		{name: "negative", limit: "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdPRCommits(&cmdutil.Factory{Config: prTestConfig{}})
			_ = cmd.Flags().Set("limit", tt.limit)
			err := cmd.RunE(cmd, []string{"alice/demo", "42"})
			if err == nil || !strings.Contains(err.Error(), "must be positive") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestPRCommitsInvalidPRNumberBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	f := &cmdutil.Factory{Config: cfg}
	cmd := newCmdPRCommits(f)
	err := cmd.RunE(cmd, []string{"alice/demo", "0"})
	if err == nil || !strings.Contains(err.Error(), "invalid PR number") {
		t.Fatalf("error = %v", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken called %d times", cfg.getTokenCalls)
	}
}

func TestPRFilesTextOutput(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return prResponse(http.StatusOK, `[{"sha":"a","filename":"main.go","status":"modified","additions":10,"deletions":2,"too_large":false,"blob_url":"https://b","raw_url":"https://r","patch":{"old_path":"main.go","new_path":"main.go","added_lines":10,"removed_lines":2,"too_large":false}},{"sha":"b","filename":"old.go","status":"renamed","additions":0,"deletions":0,"too_large":false,"blob_url":"https://b2","raw_url":"https://r2","patch":{"old_path":"old.go","new_path":"new.go","added_lines":0,"removed_lines":0,"too_large":false}}]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRFiles(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2: %q", len(lines), output.String())
	}
	if !strings.Contains(lines[0], "modified +10 -2 main.go") {
		t.Errorf("line 0 = %s", lines[0])
	}
	if !strings.Contains(lines[1], "renamed") && !strings.Contains(lines[1], "old.go -> new.go") {
		t.Errorf("line 1 = %s", lines[1])
	}
}

func TestPRFilesJSON(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return prResponse(http.StatusOK, `[{"sha":"a","filename":"main.go","status":"modified","additions":10,"deletions":2,"too_large":false,"blob_url":"https://b","raw_url":"https://r","patch":{"old_path":"main.go","new_path":"main.go","added_lines":10,"removed_lines":2,"too_large":false}}]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRFiles(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal(output.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	f := items[0]
	if f["oldPath"] != "main.go" || f["newPath"] != "main.go" || f["changeType"] != "modified" {
		t.Fatalf("item = %#v", f)
	}
	if f["additions"].(float64) != 10 || f["deletions"].(float64) != 2 {
		t.Fatalf("additions/deletions = %v/%v", f["additions"], f["deletions"])
	}
}

func TestPRFilesEmptyResponse(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return prResponse(http.StatusOK, `[]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRFiles(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(output.String()) != "[]" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestPRFilesPatchFallback(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return prResponse(http.StatusOK, `[{"sha":"a","filename":"renamed.go","status":"renamed","additions":0,"deletions":0,"too_large":false,"blob_url":"https://b","raw_url":"https://r","patch":{"old_path":"old_name.go","new_path":"renamed.go","added_lines":5,"removed_lines":3,"too_large":true}}]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRFiles(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal(output.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	f := items[0]
	if f["oldPath"] != "old_name.go" || f["newPath"] != "renamed.go" {
		t.Fatalf("oldPath/newPath = %v/%v", f["oldPath"], f["newPath"])
	}
	if f["additions"].(float64) != 5 || f["deletions"].(float64) != 3 {
		t.Fatalf("additions/deletions = %v/%v", f["additions"], f["deletions"])
	}
	if f["tooLarge"] != true {
		t.Fatalf("tooLarge = %v", f["tooLarge"])
	}
}

func TestPRFilesInvalidPRNumberBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	f := &cmdutil.Factory{Config: cfg}
	cmd := newCmdPRFiles(f)
	err := cmd.RunE(cmd, []string{"alice/demo", "0"})
	if err == nil || !strings.Contains(err.Error(), "invalid PR number") {
		t.Fatalf("error = %v", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken called %d times", cfg.getTokenCalls)
	}
}

func TestPRFilesGETOnlyRequestCount(t *testing.T) {
	var count int
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				count++
				if req.Method != http.MethodGet {
					t.Errorf("method = %s, want GET", req.Method)
				}
				return prResponse(http.StatusOK, `[]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRFiles(factory)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("requests = %d, want 1", count)
	}
}

func TestPRReactionsTextOutput(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return prResponse(http.StatusOK, `[{"id":1,"user":{"login":"alice"},"content":"+1","created_at":"2024-06-15T10:30:00Z"},{"id":2,"user":{"login":"bob"},"content":"heart","created_at":"2024-06-14T08:00:00Z"}]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRReactions(factory)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d", len(lines))
	}
	if !strings.Contains(lines[0], "+1") || !strings.Contains(lines[0], "alice") || !strings.Contains(lines[0], "2024-06-15 10:30:00Z") {
		t.Errorf("line 0 = %s", lines[0])
	}
	if !strings.Contains(lines[1], "heart") || !strings.Contains(lines[1], "bob") {
		t.Errorf("line 1 = %s", lines[1])
	}
}

func TestPRReactionsJSON(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return prResponse(http.StatusOK, `[{"id":1,"user":{"login":"alice"},"content":"+1","created_at":"2024-01-01T00:00:00Z"}]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRReactions(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal(output.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	r := items[0]
	if r["id"].(float64) != 1 || r["author"] != "alice" || r["content"] != "+1" || r["createdAt"] != "2024-01-01T00:00:00Z" {
		t.Fatalf("reaction = %#v", r)
	}
}

func TestPRReactionsEmptyResponse(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return prResponse(http.StatusOK, `[]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRReactions(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(output.String()) != "[]" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestPRReactionsInvalidPRNumberBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	f := &cmdutil.Factory{Config: cfg}
	cmd := newCmdPRReactions(f)
	err := cmd.RunE(cmd, []string{"alice/demo", "0"})
	if err == nil || !strings.Contains(err.Error(), "invalid PR number") {
		t.Fatalf("error = %v", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken called %d times", cfg.getTokenCalls)
	}
}

func TestPRReactionsGETOnlyRequestCount(t *testing.T) {
	var count int
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				count++
				if req.Method != http.MethodGet {
					t.Errorf("method = %s, want GET", req.Method)
				}
				return prResponse(http.StatusOK, `[]`), nil
			})}, nil
		},
	}
	cmd := newCmdPRReactions(factory)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("requests = %d, want 1", count)
	}
}

func TestPRDetailsHelp(t *testing.T) {
	tests := []struct {
		cmd  func(*cmdutil.Factory) *cobra.Command
		name string
	}{
		{newCmdPRCommits, "commits"},
		{newCmdPRFiles, "files"},
		{newCmdPRReactions, "reactions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmd(&cmdutil.Factory{})
			if cmd.Use != tt.name+" [<owner>/<repo>] <number>" {
				t.Errorf("Use = %q", cmd.Use)
			}
			_ = cmd.Args(cmd, []string{"42"})
			_ = cmd.Args(cmd, []string{"owner/repo", "42"})
			if err := cmd.Args(cmd, []string{}); err == nil {
				t.Fatal("expected error for zero args")
			}
		})
	}
}

func TestPRDetailsServerOrderPreserved(t *testing.T) {
	t.Run("commits preserve server order", func(t *testing.T) {
		factory := &cmdutil.Factory{
			Config: prTestConfig{},
			HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return prResponse(http.StatusOK, `[{"sha":"c","html_url":"","commit":{"message":"third","author":{"name":"","email":"","date":""}}},{"sha":"b","html_url":"","commit":{"message":"second","author":{"name":"","email":"","date":""}}},{"sha":"a","html_url":"","commit":{"message":"first","author":{"name":"","email":"","date":""}}}]`), nil
				})}, nil
			},
		}
		cmd := newCmdPRCommits(factory)
		var output bytes.Buffer
		cmd.SetOut(&output)
		if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		if len(lines) < 3 {
			t.Fatalf("lines = %d", len(lines))
		}
		if !strings.Contains(lines[0], "third") || !strings.Contains(lines[1], "second") || !strings.Contains(lines[2], "first") {
			t.Errorf("order wrong: %q", output.String())
		}
	})

	t.Run("files preserve server order", func(t *testing.T) {
		factory := &cmdutil.Factory{
			Config: prTestConfig{},
			HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return prResponse(http.StatusOK, `[{"sha":"a","filename":"z.go","status":"modified"},{"sha":"b","filename":"a.go","status":"added"}]`), nil
				})}, nil
			},
		}
		cmd := newCmdPRFiles(factory)
		var output bytes.Buffer
		cmd.SetOut(&output)
		if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		if len(lines) < 2 {
			t.Fatalf("lines = %d", len(lines))
		}
		if !strings.Contains(lines[0], "z.go") || !strings.Contains(lines[1], "a.go") {
			t.Errorf("order wrong: %q", output.String())
		}
	})

	t.Run("reactions preserve server order", func(t *testing.T) {
		factory := &cmdutil.Factory{
			Config: prTestConfig{},
			HttpClient: func() (*http.Client, error) {
				return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					return prResponse(http.StatusOK, `[{"id":2,"user":{"login":"bob"},"content":"-1","created_at":""},{"id":1,"user":{"login":"alice"},"content":"+1","created_at":""}]`), nil
				})}, nil
			},
		}
		cmd := newCmdPRReactions(factory)
		var output bytes.Buffer
		cmd.SetOut(&output)
		if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		if len(lines) < 2 {
			t.Fatalf("lines = %d", len(lines))
		}
		if !strings.Contains(lines[0], "bob") || !strings.Contains(lines[1], "alice") {
			t.Errorf("order wrong: %q", output.String())
		}
	})
}

func TestPRFilesZeroValueFallbacks(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: prTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: prRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				body := `[{"sha":"a","filename":"new_file.go","status":"added","additions":0,"deletions":0,"too_large":false,"blob_url":"","raw_url":"","patch":{"old_path":"","new_path":"","added_lines":0,"removed_lines":0,"too_large":false}}]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			})}, nil
		},
	}
	cmd := newCmdPRFiles(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "42"}); err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal(output.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	f := items[0]
	if f["additions"].(float64) != 0 {
		t.Fatalf("additions = %v, want 0", f["additions"])
	}
	if f["deletions"].(float64) != 0 {
		t.Fatalf("deletions = %v, want 0", f["deletions"])
	}
	if f["tooLarge"] != false {
		t.Fatalf("tooLarge = %v", f["tooLarge"])
	}
}
