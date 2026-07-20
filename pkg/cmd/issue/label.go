package issue

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdIssueLabel(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "label [<owner>/<repo>] <number> <labels>",
		Short:   "Add labels to an issue",
		Example: `  ag issue label owner/repo 42 "bug, help wanted,priority/high"`,
		Args:    cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 2)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			number := remaining[0]
			labels, err := parseIssueLabels(remaining[1])
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}
			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/repos/%s/%s/issues/%s/labels", owner, repo, number)
			if err := client.Post(path, labels, nil); err != nil {
				return fmt.Errorf("failed to add labels to issue: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Added labels to issue #%s: %s\n", number, strings.Join(labels, ", "))
			return nil
		},
	}

	return cmd
}

func parseIssueLabels(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		label := strings.TrimSpace(part)
		if label == "" {
			return nil, fmt.Errorf("label cannot be empty")
		}
		labels = append(labels, label)
	}
	return labels, nil
}
