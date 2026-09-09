package label

import (
	"io"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

// labelRecordingConfig counts GetToken calls so tests can assert that purely
// local validation (flags and repository parsing) runs before authentication.
type labelRecordingConfig struct {
	getTokenCalls int
}

func (c *labelRecordingConfig) GetToken() (string, error) {
	c.getTokenCalls++
	return "token", nil
}
func (*labelRecordingConfig) GetUser() (string, error) { return "alice", nil }
func (*labelRecordingConfig) GetHost() string          { return "atomgit.com" }

// An invalid --limit must be rejected before GetToken is ever called, so an
// unauthenticated user sees a usage error instead of a login prompt (issue #49).
func TestLabelListRejectsInvalidLimitBeforeAuth(t *testing.T) {
	cfg := &labelRecordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdLabelList(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Flags().Set("limit", "0"); err != nil {
		t.Fatalf("set limit: %v", err)
	}
	err := cmd.RunE(cmd, []string{"alice/demo"})

	if err == nil || !strings.Contains(err.Error(), "invalid limit") {
		t.Fatalf("error = %v, want 'invalid limit'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; invalid limit must be rejected before authentication", cfg.getTokenCalls)
	}
}
