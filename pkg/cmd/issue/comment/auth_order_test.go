package comment

import (
	"io"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

// issueCommentRecordingConfig counts GetToken calls so tests can assert that
// the issue number is parsed and rejected before authentication.
type issueCommentRecordingConfig struct {
	getTokenCalls int
}

func (c *issueCommentRecordingConfig) GetToken() (string, error) {
	c.getTokenCalls++
	return "token", nil
}
func (*issueCommentRecordingConfig) GetUser() (string, error) { return "alice", nil }
func (*issueCommentRecordingConfig) GetHost() string          { return "atomgit.com" }

// An invalid issue number must be rejected before GetToken, so an
// unauthenticated user sees the parse error instead of a login prompt or
// interactive body input (issue #49).
func TestIssueCommentCreateRejectsInvalidNumberBeforeAuth(t *testing.T) {
	cfg := &issueCommentRecordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdCreate(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	_ = cmd.Flags().Set("body", "hello")

	err := cmd.RunE(cmd, []string{"alice/demo", "not-a-number"})

	if err == nil || !strings.Contains(err.Error(), "invalid issue number") {
		t.Fatalf("error = %v, want 'invalid issue number'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; invalid issue number must be rejected before authentication", cfg.getTokenCalls)
	}
}
