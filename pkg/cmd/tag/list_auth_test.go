package tag

import (
	"io"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type recordingConfig struct {
	getTokenCalls int
}

func (r *recordingConfig) GetToken() (string, error) {
	r.getTokenCalls++
	return "token", nil
}
func (*recordingConfig) GetUser() (string, error) { return "alice", nil }
func (*recordingConfig) GetHost() string          { return "atomgit.com" }

// Issue #49 requires arguments/flags -> authentication -> execution, so a
// repository context that cannot be resolved locally must fail before
// GetToken is ever called; an unauthenticated user outside a Git repository
// sees the resolution error instead of a login prompt.
func TestTagListRejectsUnresolvableRepositoryBeforeAuth(t *testing.T) {
	cfg := &recordingConfig{}
	factory := &cmdutil.Factory{Config: cfg}
	cmd := newCmdTagList(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.RunE(cmd, nil)

	if err == nil || !strings.Contains(err.Error(), "unable to determine repository") {
		t.Fatalf("error = %v, want 'unable to determine repository'", err)
	}
	if cfg.getTokenCalls != 0 {
		t.Fatalf("GetToken was called %d times; repository resolution must happen before authentication", cfg.getTokenCalls)
	}
}

func TestTagMutationsRejectInvalidInputBeforeAuth(t *testing.T) {
	commands := []struct {
		name string
		new  func(*cmdutil.Factory) *cobra.Command
	}{
		{name: "create", new: newCmdTagCreate},
		{name: "delete", new: newCmdTagDelete},
	}
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{name: "repository", args: []string{"invalid-repo", "v1"}, wantError: "invalid repository format"},
		{name: "blank name", args: []string{"owner/repo", " \t "}, wantError: "tag name is required"},
	}

	for _, command := range commands {
		for _, tt := range tests {
			t.Run(command.name+"/"+tt.name, func(t *testing.T) {
				cfg := &recordingConfig{}
				factory := &cmdutil.Factory{Config: cfg}
				cmd := command.new(factory)
				cmd.SetOut(io.Discard)
				cmd.SetErr(io.Discard)

				err := cmd.RunE(cmd, tt.args)

				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want %q", err, tt.wantError)
				}
				if cfg.getTokenCalls != 0 {
					t.Fatalf("GetToken was called %d times; invalid input must be rejected before authentication", cfg.getTokenCalls)
				}
			})
		}
	}
}
