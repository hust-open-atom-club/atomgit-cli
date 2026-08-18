package pr

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
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

func TestPRViewRejectsInvalidNumberBeforeAuth(t *testing.T) {
	for _, number := range []string{"bad", "0", "-1"} {
		t.Run(number, func(t *testing.T) {
			cfg := &recordingConfig{}
			factory := &cmdutil.Factory{Config: cfg}
			cmd := newCmdPRView(factory)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			err := cmd.RunE(cmd, []string{"owner/repo", number})

			if err == nil || !strings.Contains(err.Error(), "invalid PR number") {
				t.Fatalf("error = %v, want 'invalid PR number'", err)
			}
			if cfg.getTokenCalls != 0 {
				t.Fatalf("GetToken was called %d times; invalid PR number must be rejected before authentication", cfg.getTokenCalls)
			}
		})
	}
}

func TestPRViewWebRejectsNonPositiveNumber(t *testing.T) {
	opened := 0
	factory := &cmdutil.Factory{
		Config: &recordingConfig{},
		BrowserOpener: func(string) error {
			opened++
			return nil
		},
	}
	cmd := newCmdPRView(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Flags().Set("web", "true"); err != nil {
		t.Fatalf("set web flag: %v", err)
	}

	err := cmd.RunE(cmd, []string{"owner/repo", "0"})

	if err == nil || !strings.Contains(err.Error(), "invalid PR number") {
		t.Fatalf("error = %v, want 'invalid PR number'", err)
	}
	if opened != 0 {
		t.Fatalf("BrowserOpener was called %d times; invalid PR number must not open a URL", opened)
	}
}

func TestPRCreateRejectsLocalErrorsBeforeAuth(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*testing.T, *cobra.Command)
		args      []string
		wantError string
	}{
		{
			name:      "missing title",
			wantError: "title is required",
		},
		{
			name: "body file",
			configure: func(t *testing.T, cmd *cobra.Command) {
				t.Helper()
				if err := cmd.Flags().Set("title", "test"); err != nil {
					t.Fatal(err)
				}
				if err := cmd.Flags().Set("body-file", filepath.Join(t.TempDir(), "missing.md")); err != nil {
					t.Fatal(err)
				}
			},
			wantError: "failed to read body file",
		},
		{
			name: "repository context",
			configure: func(t *testing.T, cmd *cobra.Command) {
				t.Helper()
				for name, value := range map[string]string{"title": "test", "base": "main", "head": "feature"} {
					if err := cmd.Flags().Set(name, value); err != nil {
						t.Fatal(err)
					}
				}
			},
			wantError: "unable to determine repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &recordingConfig{}
			factory := &cmdutil.Factory{Config: cfg}
			cmd := newCmdPRCreate(factory)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			if tt.configure != nil {
				tt.configure(t, cmd)
			}

			err := cmd.RunE(cmd, tt.args)

			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want %q", err, tt.wantError)
			}
			if cfg.getTokenCalls != 0 {
				t.Fatalf("GetToken was called %d times; local errors must be rejected before authentication", cfg.getTokenCalls)
			}
		})
	}
}

func TestPRStateCommandsRejectInvalidInputBeforeAuth(t *testing.T) {
	commands := []struct {
		name string
		new  func(*cmdutil.Factory) *cobra.Command
	}{
		{name: "close", new: newCmdPRClose},
		{name: "reopen", new: newCmdPRReopen},
	}
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{name: "repository", args: []string{"invalid-repo", "1"}, wantError: "invalid repository format"},
		{name: "number", args: []string{"owner/repo", "bad"}, wantError: "invalid PR number"},
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
