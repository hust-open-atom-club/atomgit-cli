package issue

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestIssuePRsList(t *testing.T) {
	prsJSON := `[{"id":1,"number":7,"title":"Fix bug","body":"details","state":"open","html_url":"https://example.test/pulls/7","url":"https://api.example.test/pulls/7","created_at":"2026-01-01","updated_at":"2026-01-02"}]`

	tests := []struct {
		name       string
		body       string
		json       bool
		wantOutput string
	}{
		{
			name:       "text output",
			body:       prsJSON,
			wantOutput: "#7 Fix bug [open]\n",
		},
		{
			name:       "json output",
			body:       prsJSON,
			json:       true,
			wantOutput: "Fix bug",
		},
		{
			name:       "empty text",
			body:       `[]`,
			wantOutput: "",
		},
		{
			name:       "empty json",
			body:       `[]`,
			json:       true,
			wantOutput: "[]",
		},
		{
			name:       "null response json",
			body:       `null`,
			json:       true,
			wantOutput: "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{
				Config: issueTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/issues/7/pull_requests" {
							t.Fatalf("request = %s %s", req.Method, req.URL.Path)
						}
						return issueResponse(http.StatusOK, tt.body), nil
					})}, nil
				},
			}

			cmd := newCmdIssuePRS(factory)
			if tt.json {
				_ = cmd.Flags().Set("json", "true")
			}
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), tt.wantOutput) {
				t.Fatalf("output = %q, want containing %q", output.String(), tt.wantOutput)
			}
		})
	}
}

func TestIssuePRsJSONSchema(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return issueResponse(http.StatusOK, `[{"id":1,"number":7,"title":"Fix bug","body":"desc","state":"open","html_url":"https://ex.test/pulls/7","url":"https://api.ex.test/pulls/7","created_at":"2026-01-01","updated_at":"2026-01-02"}]`), nil
			})}, nil
		},
	}

	cmd := newCmdIssuePRS(factory)
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}

	var prs []struct {
		ID     int64  `json:"id"`
		Number string `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		State  string `json:"state"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal(output.Bytes(), &prs); err != nil {
		t.Fatal(err)
	}
	if len(prs) != 1 || prs[0].Number != "7" || prs[0].Title != "Fix bug" {
		t.Fatalf("prs = %#v", prs)
	}
}

func TestIssuePRsInvalidNumber(t *testing.T) {
	tests := []string{"0", "-1", "abc", ""}
	for _, num := range tests {
		t.Run("num="+num, func(t *testing.T) {
			cmd := newCmdIssuePRS(&cmdutil.Factory{Config: issueEditAuthErrorConfig{}})
			err := cmd.RunE(cmd, []string{"alice/demo", num})
			if err == nil || !strings.Contains(err.Error(), "positive integer") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestIssueBranchesList(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		json       bool
		wantOutput string
	}{
		{
			name:       "text output",
			body:       `["main","feature/x"]`,
			wantOutput: "main\nfeature/x\n",
		},
		{
			name:       "json output",
			body:       `["main","feature/x"]`,
			json:       true,
			wantOutput: "main",
		},
		{
			name:       "empty text",
			body:       `[]`,
			wantOutput: "",
		},
		{
			name:       "empty json",
			body:       `[]`,
			json:       true,
			wantOutput: "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{
				Config: issueTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						if req.Method != http.MethodGet || req.URL.Path != "/api/v5/repos/alice/demo/issues/7/related_branches" {
							t.Fatalf("request = %s %s", req.Method, req.URL.Path)
						}
						return issueResponse(http.StatusOK, tt.body), nil
					})}, nil
				},
			}

			cmd := newCmdIssueBranches(factory)
			if tt.json {
				_ = cmd.Flags().Set("json", "true")
			}
			var output bytes.Buffer
			cmd.SetOut(&output)
			if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), tt.wantOutput) {
				t.Fatalf("output = %q, want containing %q", output.String(), tt.wantOutput)
			}
		})
	}
}

func TestIssueBranchesValidation(t *testing.T) {
	tests := []struct {
		name      string
		add       []string
		remove    []string
		wantError string
	}{
		{name: "empty branch name in add", add: []string{"  "}, wantError: "branch name cannot be empty"},
		{name: "empty branch name in remove", remove: []string{""}, wantError: "branch name cannot be empty"},
		{name: "duplicate in add", add: []string{"main", "main"}, wantError: "duplicate branch name in --add"},
		{name: "duplicate in remove", remove: []string{"x", "x"}, wantError: "duplicate branch name in --remove"},
		{name: "overlap add and remove", add: []string{"main"}, remove: []string{"main"}, wantError: "cannot be in both --add and --remove"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &cmdutil.Factory{Config: issueEditAuthErrorConfig{}}
			cmd := newCmdIssueBranches(factory)
			cmd.SetIn(strings.NewReader("yes\n"))
			for _, a := range tt.add {
				_ = cmd.Flags().Set("add", a)
			}
			for _, r := range tt.remove {
				_ = cmd.Flags().Set("remove", r)
			}
			_ = cmd.Flags().Set("yes", "true")
			err := cmd.RunE(cmd, []string{"alice/demo", "7"})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestIssueBranchesInvalidNumber(t *testing.T) {
	cmd := newCmdIssueBranches(&cmdutil.Factory{Config: issueEditAuthErrorConfig{}})
	err := cmd.RunE(cmd, []string{"alice/demo", "0"})
	if err == nil || !strings.Contains(err.Error(), "positive integer") {
		t.Fatalf("error = %v", err)
	}
}

func TestIssueBranchesAuthenticatesBeforeConfirmation(t *testing.T) {
	cmd := newCmdIssueBranches(&cmdutil.Factory{Config: issueEditAuthErrorConfig{}})
	if err := cmd.Flags().Set("remove", "main"); err != nil {
		t.Fatal(err)
	}
	cmd.SetIn(strings.NewReader("yes\n"))
	var output bytes.Buffer
	cmd.SetOut(&output)

	err := cmd.RunE(cmd, []string{"alice/demo", "7"})
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("error = %v, want authentication error", err)
	}
	if output.Len() != 0 {
		t.Fatalf("prompted before authentication: %q", output.String())
	}
}

func TestIssueBranchesMutationConfirmation(t *testing.T) {
	tests := []struct {
		name      string
		flags     map[string]string
		input     string
		wantCalls int
		wantErr   bool
	}{
		{name: "add no confirmation", flags: map[string]string{"add": "new-feature", "yes": "true"}, input: "", wantCalls: 2},
		{name: "add no prompt needed", flags: map[string]string{"add": "new-feature"}, input: "", wantCalls: 2},
		{name: "remove with yes", flags: map[string]string{"remove": "main", "yes": "true"}, input: "", wantCalls: 2},
		{name: "remove decline", flags: map[string]string{"remove": "main"}, input: "no\n", wantCalls: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			factory := &cmdutil.Factory{
				Config: issueTestConfig{},
				HttpClient: func() (*http.Client, error) {
					return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
						requests++
						if req.Method == http.MethodGet {
							return issueResponse(http.StatusOK, `["main"]`), nil
						}
						if req.Method == http.MethodPut {
							return issueResponse(http.StatusOK, ""), nil
						}
						t.Fatalf("unexpected method: %s", req.Method)
						return nil, nil
					})}, nil
				},
			}

			cmd := newCmdIssueBranches(factory)
			cmd.SetIn(strings.NewReader(tt.input))
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)
			for name, value := range tt.flags {
				_ = cmd.Flags().Set(name, value)
			}

			err := cmd.RunE(cmd, []string{"alice/demo", "7"})
			if tt.wantErr {
				if err != nil {
					t.Fatalf("expected nil (decline returns nil), got %v", err)
				}
				if requests != 0 {
					t.Fatalf("requests = %d, want 0", requests)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if requests != tt.wantCalls {
					t.Fatalf("requests = %d, want %d", requests, tt.wantCalls)
				}
			}
		})
	}
}

func TestIssueBranchesUnchangedSkipsPUT(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method == http.MethodGet {
					return issueResponse(http.StatusOK, `["main"]`), nil
				}
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				return nil, nil
			})}, nil
		},
	}

	cmd := newCmdIssueBranches(factory)
	_ = cmd.Flags().Set("add", "main")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1 (GET only, no PUT)", requests)
	}
	if !strings.Contains(output.String(), "No changes needed") {
		t.Fatalf("output = %q, want 'No changes needed'", output.String())
	}
}

func TestIssueBranchesChangedPUTCounts(t *testing.T) {
	requests := 0
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if requests == 1 && req.Method != http.MethodGet {
					t.Fatalf("call 1: method = %s, want GET", req.Method)
				}
				if requests == 2 && req.Method != http.MethodPut {
					t.Fatalf("call 2: method = %s, want PUT", req.Method)
				}
				if requests > 2 {
					t.Fatalf("unexpected call %d: %s", requests, req.Method)
				}
				return issueResponse(http.StatusOK, `["main"]`), nil
			})}, nil
		},
	}

	cmd := newCmdIssueBranches(factory)
	_ = cmd.Flags().Set("add", "feature/x")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2 (1 GET + 1 PUT)", requests)
	}
}

func TestIssueBranchesComputeDesired(t *testing.T) {
	tests := []struct {
		name    string
		current []string
		add     []string
		remove  []string
		want    []string
	}{
		{
			name:    "add new",
			current: []string{"main"},
			add:     []string{"feature/x"},
			want:    []string{"main", "feature/x"},
		},
		{
			name:    "remove existing",
			current: []string{"main", "feature/x"},
			remove:  []string{"main"},
			want:    []string{"feature/x"},
		},
		{
			name:    "add and remove",
			current: []string{"main", "old-feature"},
			add:     []string{"new-feature"},
			remove:  []string{"old-feature"},
			want:    []string{"main", "new-feature"},
		},
		{
			name:    "preserve order",
			current: []string{"z", "a", "m"},
			add:     []string{"n"},
			want:    []string{"z", "a", "m", "n"},
		},
		{
			name:    "skip already present",
			current: []string{"main"},
			add:     []string{"main"},
			want:    []string{"main"},
		},
		{
			name:    "empty result",
			current: []string{"main"},
			remove:  []string{"main"},
			want:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeDesiredBranches(tt.current, tt.add, tt.remove)
			if got == nil {
				t.Fatalf("got nil, want non-nil slice %v", tt.want)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestIssueBranchesJSONMutation(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					return issueResponse(http.StatusOK, `["main"]`), nil
				}
				return issueResponse(http.StatusOK, `["main","feature/x"]`), nil
			})}, nil
		},
	}

	cmd := newCmdIssueBranches(factory)
	_ = cmd.Flags().Set("add", "feature/x")
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}

	var result []string
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || result[0] != "main" || result[1] != "feature/x" {
		t.Fatalf("result = %#v", result)
	}
}

func TestIssueBranchesRemoveLastBranchJSONUsesEmptyArray(t *testing.T) {
	factory := &cmdutil.Factory{
		Config: issueTestConfig{},
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: issueRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					return issueResponse(http.StatusOK, `["main"]`), nil
				}
				var body map[string]json.RawMessage
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if got := string(body["branch_names"]); got != "[]" {
					t.Fatalf("branch_names = %s, want []", got)
				}
				return issueResponse(http.StatusOK, ""), nil
			})}, nil
		},
	}

	cmd := newCmdIssueBranches(factory)
	_ = cmd.Flags().Set("remove", "main")
	_ = cmd.Flags().Set("yes", "true")
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice/demo", "7"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != "[]" {
		t.Fatalf("output = %q, want []", got)
	}
}

func TestIssueBranchesHelpHasConcurrencyWarning(t *testing.T) {
	cmd := newCmdIssueBranches(&cmdutil.Factory{})
	if !strings.Contains(cmd.Long, "Concurrent") {
		t.Fatal("branches help missing concurrency warning")
	}
}

func TestBranchesEqual(t *testing.T) {
	if !branchesEqual(nil, nil) {
		t.Fatal("nil slices should be equal")
	}
	if !branchesEqual([]string{}, []string{}) {
		t.Fatal("empty slices should be equal")
	}
	if branchesEqual([]string{"a"}, []string{"b"}) {
		t.Fatal("different values should not be equal")
	}
	if branchesEqual([]string{"a"}, []string{"a", "b"}) {
		t.Fatal("different lengths should not be equal")
	}
}
