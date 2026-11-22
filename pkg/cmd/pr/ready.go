package pr

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type prDraftState struct {
	Draft    *bool  `json:"draft"`
	State    string `json:"state"`
	Merged   bool   `json:"merged"`
	MergedAt string `json:"merged_at"`
	HTMLURL  string `json:"html_url"`
}

func newCmdPRReady(f *cmdutil.Factory) *cobra.Command {
	var undo bool
	cmd := &cobra.Command{
		Use:   "ready [<owner>/<repo>] <number>",
		Short: "Mark a draft pull request as ready for review",
		Long:  "Mark a draft pull request as ready for review. Use --undo to make it a draft again.",
		Example: `  ag-cli pr ready owner/repo 123
  ag-cli pr ready 123
  ag-cli pr ready owner/repo 123 --undo`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			number, err := parsePRNumber(remaining[0])
			if err != nil {
				return err
			}
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repos/%s/%s/pulls/%s", repository.Owner, repository.Name, number)
			var current prDraftState
			if err := client.Get(path, &current); err != nil {
				return fmt.Errorf("get PR #%s: %w", number, err)
			}
			if err := validatePRDraftState(number, current); err != nil {
				return err
			}
			if *current.Draft != undo {
				if err := api.SetPullRequestDraft(client, repository.Owner, repository.Name, number, undo); err != nil {
					return fmt.Errorf("update PR #%s: %w", number, err)
				}
				var updated prDraftState
				if err := client.Get(path, &updated); err != nil {
					return fmt.Errorf("verify PR #%s draft status: %w; status may have changed", number, err)
				}
				if err := validatePRDraftState(number, updated); err != nil {
					return fmt.Errorf("verify PR #%s draft status: %w; status may have changed", number, err)
				}
				if *updated.Draft != undo {
					return fmt.Errorf("PR #%s draft status did not change; check AtomGit before retrying", number)
				}
				current = updated
			}
			url := pullRequestResultURL(current.HTMLURL, f.Config.GetHost(), repository.Owner, repository.Name, number)
			if undo {
				fmt.Fprintf(cmd.OutOrStdout(), "PR #%s is a draft: %s\n", number, url)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "PR #%s is ready for review: %s\n", number, url)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&undo, "undo", false, "Convert the pull request to a draft")
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func validatePRDraftState(number string, pr prDraftState) error {
	if pr.Draft == nil {
		return fmt.Errorf("PR #%s response has no draft status", number)
	}
	if pr.Merged || pr.MergedAt != "" || strings.EqualFold(pr.State, "merged") {
		return fmt.Errorf("PR #%s is already merged", number)
	}
	if !strings.EqualFold(pr.State, "open") && !strings.EqualFold(pr.State, "opened") {
		return fmt.Errorf("PR #%s is not open", number)
	}
	return nil
}
