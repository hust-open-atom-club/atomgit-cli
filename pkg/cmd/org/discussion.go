package org

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmd/discussion"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdOrgDiscussion(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "discussion",
		Short:        "Manage organization discussions",
		Long:         "List, view, create, edit, and delete organization discussions and their comments. An explicit organization path is required for every operation.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newCmdOrgDiscussionList(f))
	cmd.AddCommand(newCmdOrgDiscussionView(f))
	cmd.AddCommand(newCmdOrgDiscussionCreate(f))
	cmd.AddCommand(newCmdOrgDiscussionEdit(f))
	cmd.AddCommand(newCmdOrgDiscussionDelete(f))
	cmd.AddCommand(newCmdOrgDiscussionComment(f))
	commandschema.Annotate(cmd, commandschema.Metadata{Output: "text", Effects: "none"})
	return cmd
}

func newCmdOrgDiscussionList(f *cmdutil.Factory) *cobra.Command {
	opts := &collectionOptions{Limit: defaultOrganizationCollectionLimit}
	cmd := &cobra.Command{
		Use:   "list <org>",
		Short: "List organization discussions",
		Example: `  ag-cli org discussion list my-organization
  ag-cli org discussion list my-organization --limit 100
  ag-cli org discussion list my-organization --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseReadDiscussionOrganization(args[0])
			if err != nil {
				return err
			}
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			client, err := f.OptionalAPIClient()
			if err != nil {
				return err
			}
			items, err := api.ListOrganizationDiscussions(client, organization, opts.Limit)
			if err != nil {
				return fmt.Errorf("failed to list discussions for organization %q: %w", organization, err)
			}
			return discussion.WriteList(cmd.OutOrStdout(), items, opts.JSON)
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultOrganizationCollectionLimit, "Maximum number of discussions to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output discussions as JSON")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 1, Description: "An explicit organization path; Git repository context is not used."},
		Output:      "text,json", Effects: "read",
		Notes: []string{
			"Uses authentication when configured; public organization discussions may be read anonymously.",
			"--limit must be positive and caps the total number of discussions across pages.",
		},
	})
	return cmd
}

func newCmdOrgDiscussionView(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Comments bool
		JSON     bool
	}
	cmd := &cobra.Command{
		Use:   "view <org> <number>",
		Short: "View an organization discussion",
		Long: `View a single organization discussion with its Markdown body.

With --comments, fetch comments and their nested replies in server order.
Hidden, deleted, and empty bodies are shown as explicit placeholders.`,
		Example: `  ag-cli org discussion view my-organization 1
  ag-cli org discussion view my-organization 1 --comments
  ag-cli org discussion view my-organization 1 --comments --json`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseReadDiscussionOrganization(args[0])
			if err != nil {
				return err
			}
			number, err := strconv.Atoi(args[1])
			if err != nil || number <= 0 {
				return fmt.Errorf("invalid discussion number %q: must be a positive integer", args[1])
			}
			client, err := f.OptionalAPIClient()
			if err != nil {
				return err
			}
			detail, err := api.GetOrganizationDiscussion(client, organization, number)
			if err != nil {
				return fmt.Errorf("failed to view discussion #%d for organization %q: %w", number, organization, err)
			}
			var threads []api.DiscussionCommentThread
			if opts.Comments {
				threads, err = api.ListOrganizationDiscussionThreads(client, organization, number)
				if err != nil {
					return fmt.Errorf("failed to list comments of discussion #%d for organization %q: %w", number, organization, err)
				}
			}
			return discussion.WriteView(cmd.OutOrStdout(), detail, threads, opts.JSON)
		},
	}
	cmd.Flags().BoolVar(&opts.Comments, "comments", false, "Also fetch and show the comment thread")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output the discussion as JSON")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 2, MaxCount: 2, Description: "An explicit organization path and a positive discussion number; Git repository context is not used."},
		Output:      "text,json", Effects: "read",
		Notes: []string{
			"Uses authentication when configured; public organization discussions may be read anonymously.",
			"--comments fetches all comments and nested replies. JSON omits comments unless requested, and includes an empty array when there are none.",
		},
	})
	return cmd
}

func parseReadDiscussionOrganization(value string) (string, error) {
	organization, err := parseOrganization(value)
	if err != nil {
		return "", err
	}
	if organization == "." || organization == ".." {
		return "", fmt.Errorf("invalid organization: %q", value)
	}
	for _, char := range organization {
		if unicode.IsSpace(char) || unicode.IsControl(char) {
			return "", fmt.Errorf("invalid organization: %q", value)
		}
	}
	return organization, nil
}

func parseDiscussionOrganization(value string) (string, error) {
	if !validDiscussionPathID(value) {
		return "", fmt.Errorf("invalid organization: %q", value)
	}
	return value, nil
}

func orgDiscussionNumber(value string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 || strconv.Itoa(number) != value {
		return 0, fmt.Errorf("invalid discussion number %q: must be a positive integer", value)
	}
	return number, nil
}

func validDiscussionPathID(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-._~", r)) {
			return false
		}
	}
	return true
}
