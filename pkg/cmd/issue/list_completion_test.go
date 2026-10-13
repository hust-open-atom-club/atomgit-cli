package issue

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// --state completion is served from the issue list's own static enum: an empty
// prefix lists every issue state, a typed prefix narrows the candidates, and
// an unmatched prefix keeps file completion disabled instead of listing files.
// Neither the long option nor the -s shorthand needs authentication or sends
// network requests.
func TestIssueListStateCompletion(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "empty prefix lists all issue states",
			args: []string{"__complete", "issue", "list", "--state", ""},
			want: []string{"open", "closed", "all"},
		},
		{
			name: "prefix narrows candidates",
			args: []string{"__complete", "issue", "list", "--state", "cl"},
			want: []string{"closed"},
		},
		{
			name: "unmatched prefix keeps file completion disabled",
			args: []string{"__complete", "issue", "list", "--state", "zzz"},
			want: nil,
		},
		{
			name: "equals form completes like the separated form",
			args: []string{"__complete", "issue", "list", "--state=al"},
			want: []string{"all"},
		},
		{
			name: "shorthand completes like the long option",
			args: []string{"__complete", "issue", "list", "-s", ""},
			want: []string{"open", "closed", "all"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			root := &cobra.Command{Use: "ag-cli"}
			root.AddCommand(NewCmdIssue(&cmdutil.Factory{
				HttpClient: func() (*http.Client, error) {
					requests++
					return nil, nil
				},
			}))
			var out, errOut bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&errOut)
			root.SetArgs(tt.args)

			if err := root.Execute(); err != nil {
				t.Fatalf("execute completion: %v (stderr: %s)", err, errOut.String())
			}
			if requests != 0 {
				t.Fatalf("completion issued %d requests, want 0", requests)
			}

			completions, directive := parseCompletionOutput(t, out.String())
			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Fatalf("directive = %d, want %d (NoFileComp)", directive, cobra.ShellCompDirectiveNoFileComp)
			}
			if strings.Join(completions, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("completions = %q, want %q", completions, tt.want)
			}
		})
	}
}

// parseCompletionOutput splits cobra __complete output into candidate lines and
// the trailing directive line (":4" for ShellCompDirectiveNoFileComp).
func parseCompletionOutput(t *testing.T, output string) ([]string, cobra.ShellCompDirective) {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	directiveLine := lines[len(lines)-1]
	if !strings.HasPrefix(directiveLine, ":") {
		t.Fatalf("last line %q is not a directive line, output: %q", directiveLine, output)
	}
	directive, err := strconv.Atoi(strings.TrimPrefix(directiveLine, ":"))
	if err != nil {
		t.Fatalf("parse directive %q: %v", directiveLine, err)
	}
	return lines[:len(lines)-1], cobra.ShellCompDirective(directive)
}
