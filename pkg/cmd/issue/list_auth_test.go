package issue

import (
	"io"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

// An invalid --limit must be rejected before GetToken is ever called, so an
// unauthenticated user sees a usage error instead of a login prompt (issue #49).
// issueRecordingConfig (defined in issue_test.go) counts GetToken calls.
func TestIssueListRejectsInvalidLimitBeforeAuth(t *testing.T) {
	cfg := &issueRecordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdIssueList(factory)
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
