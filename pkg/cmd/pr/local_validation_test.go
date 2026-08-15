package pr

import (
	"io"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

// Issue #49 requires arguments/flags -> authentication -> execution, so
// invalid local input must be rejected before GetToken is ever called and an
// unauthenticated user sees a usage error instead of a login prompt.
// recordingConfig (defined in pr_test.go) counts GetToken calls.
func TestPREditRejectsMissingFieldsBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdPREdit(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.RunE(cmd, []string{"owner/repo", "1"})

	if err == nil || !strings.Contains(err.Error(), "at least one PR field") {
		t.Fatalf("error = %v, want 'at least one PR field'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; missing edit fields must be rejected before authentication", cfg.getTokenCalls)
	}
}

func TestPRMergeRejectsInvalidPRNumberBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdPRMerge(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.RunE(cmd, []string{"owner/repo", "bad"})

	if err == nil || !strings.Contains(err.Error(), "invalid PR number") {
		t.Fatalf("error = %v, want 'invalid PR number'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; invalid PR number must be rejected before authentication", cfg.getTokenCalls)
	}
}

func TestLinkIssuesRejectsMissingIssuesBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdLinkIssues(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.RunE(cmd, []string{"owner/repo", "1"})

	if err == nil || !strings.Contains(err.Error(), "at least one issue number is required") {
		t.Fatalf("error = %v, want 'at least one issue number is required'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; missing --issue must be rejected before authentication", cfg.getTokenCalls)
	}
}

func TestLinkIssuesRejectsInvalidIssueNumberBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdLinkIssues(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Flags().Set("issue", "bad"); err != nil {
		t.Fatalf("set issue: %v", err)
	}
	err := cmd.RunE(cmd, []string{"owner/repo", "1"})

	if err == nil || !strings.Contains(err.Error(), "invalid issue number") {
		t.Fatalf("error = %v, want 'invalid issue number'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; invalid issue number must be rejected before authentication", cfg.getTokenCalls)
	}
}

func TestUnlinkIssuesRejectsMissingIssuesBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdUnlinkIssues(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.RunE(cmd, []string{"owner/repo", "1"})

	if err == nil || !strings.Contains(err.Error(), "at least one issue number is required") {
		t.Fatalf("error = %v, want 'at least one issue number is required'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; missing --issue must be rejected before authentication", cfg.getTokenCalls)
	}
}
