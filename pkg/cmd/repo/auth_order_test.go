package repo

import (
	"io"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

// repoRecordingConfig counts GetToken calls so tests can assert that purely
// local validation (flags and repository parsing) runs before authentication.
type repoRecordingConfig struct {
	repoCommandConfig
	getTokenCalls int
	getUserCalls  int
}

func (c *repoRecordingConfig) GetToken() (string, error) {
	c.getTokenCalls++
	return "token", nil
}

func (c *repoRecordingConfig) GetUser() (string, error) {
	c.getUserCalls++
	return "alice", nil
}

// An invalid --limit must be rejected before GetToken is ever called, so an
// unauthenticated user sees a usage error instead of a login prompt (issue #49).
func TestRepoListRejectsInvalidLimitBeforeAuth(t *testing.T) {
	cfg := &repoRecordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdRepoList(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Flags().Set("limit", "0"); err != nil {
		t.Fatalf("set limit: %v", err)
	}
	err := cmd.RunE(cmd, nil)

	if err == nil || !strings.Contains(err.Error(), "invalid limit") {
		t.Fatalf("error = %v, want 'invalid limit'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; invalid limit must be rejected before authentication", cfg.getTokenCalls)
	}
}

func TestRepoDeleteRejectsMalformedExplicitRepositoryBeforeAuth(t *testing.T) {
	cfg := &repoRecordingConfig{}
	cmd := newCmdRepoDelete(&cmdutil.Factory{Config: cfg})
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatal(err)
	}

	err := cmd.RunE(cmd, []string{"invalid/repo/extra"})
	if err == nil || !strings.Contains(err.Error(), "invalid repository format") {
		t.Fatalf("error = %v, want invalid repository format", err)
	}
	if cfg.getUserCalls != 0 || cfg.getTokenCalls != 0 {
		t.Fatalf("GetUser calls = %d, GetToken calls = %d; malformed explicit repository must be rejected locally", cfg.getUserCalls, cfg.getTokenCalls)
	}
}

func TestRepoCreateRejectsMalformedExplicitRepositoryBeforeAuth(t *testing.T) {
	cfg := &repoRecordingConfig{}
	err := runCreate(strings.NewReader(""), io.Discard, io.Discard, &cmdutil.Factory{Config: cfg}, &CreateOptions{
		Name:   "invalid/repo/extra",
		Public: true,
	})
	if err == nil || !strings.Contains(err.Error(), "invalid repository format") {
		t.Fatalf("error = %v, want invalid repository format", err)
	}
	if cfg.getUserCalls != 0 || cfg.getTokenCalls != 0 {
		t.Fatalf("GetUser calls = %d, GetToken calls = %d; malformed explicit repository must be rejected locally", cfg.getUserCalls, cfg.getTokenCalls)
	}
}

func TestRepoForkRejectsLocalErrorsBeforeAuth(t *testing.T) {
	tests := []struct {
		name      string
		opts      ForkOptions
		repo      string
		wantError string
	}{
		{name: "malformed repository", repo: "invalid/repo/extra", wantError: "invalid repository format"},
		{name: "conflicting visibility", opts: ForkOptions{Public: true, Private: true}, repo: "alice/demo", wantError: "mutually exclusive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &repoRecordingConfig{}
			err := runFork(io.Discard, &cmdutil.Factory{Config: cfg}, &tt.opts, tt.repo)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want %q", err, tt.wantError)
			}
			if cfg.getUserCalls != 0 || cfg.getTokenCalls != 0 {
				t.Fatalf("GetUser calls = %d, GetToken calls = %d; local errors must be rejected before authentication", cfg.getUserCalls, cfg.getTokenCalls)
			}
		})
	}
}
