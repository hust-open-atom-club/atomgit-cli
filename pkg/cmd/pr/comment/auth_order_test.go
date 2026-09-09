package comment

import (
	"io"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

// prCommentRecordingConfig counts GetToken calls so tests can assert that the
// PR number is parsed and rejected before authentication.
type prCommentRecordingConfig struct {
	getTokenCalls int
}

func (c *prCommentRecordingConfig) GetToken() (string, error) {
	c.getTokenCalls++
	return "token", nil
}
func (*prCommentRecordingConfig) GetUser() (string, error) { return "alice", nil }
func (*prCommentRecordingConfig) GetHost() string          { return "atomgit.com" }

// An invalid PR number must be rejected before GetToken, so an unauthenticated
// user sees the parse error instead of a login prompt or interactive body input
// (issue #49).
func TestPRCommentCreateRejectsInvalidNumberBeforeAuth(t *testing.T) {
	cfg := &prCommentRecordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdCreate(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	_ = cmd.Flags().Set("body", "hello")

	err := cmd.RunE(cmd, []string{"alice/demo", "not-a-number"})

	if err == nil || !strings.Contains(err.Error(), "invalid PR number") {
		t.Fatalf("error = %v, want 'invalid PR number'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; invalid PR number must be rejected before authentication", cfg.getTokenCalls)
	}
}

// reply must reject an invalid PR number before GetToken and before any
// interactive reply-body prompt.
func TestPRCommentReplyRejectsInvalidNumberBeforeAuth(t *testing.T) {
	cfg := &prCommentRecordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdReply(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	_ = cmd.Flags().Set("body", "hello")

	err := cmd.RunE(cmd, []string{"alice/demo", "not-a-number", "deadbeef"})

	if err == nil || !strings.Contains(err.Error(), "invalid PR number") {
		t.Fatalf("error = %v, want 'invalid PR number'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; invalid PR number must be rejected before authentication", cfg.getTokenCalls)
	}
}
