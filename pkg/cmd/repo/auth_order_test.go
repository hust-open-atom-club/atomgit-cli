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
}

func (c *repoRecordingConfig) GetToken() (string, error) {
	c.getTokenCalls++
	return "token", nil
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
