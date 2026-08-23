package tag

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdTag(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage tags",
		Long:  `List, create, and delete tags.`,
	}

	cmd.AddCommand(newCmdTagList(f))
	cmd.AddCommand(newCmdTagCreate(f))
	cmd.AddCommand(newCmdTagDelete(f))
	cmdutil.AddRepositoryContextHelp(cmd)

	return cmd
}

func newCmdTagList(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	var limit int
	cmd := &cobra.Command{
		Use:     "list [<owner>/<repo>]",
		Short:   "List tags",
		Example: `  ag tag list owner/repo --limit 50`,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", limit)
			}

			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			tags, err := api.GetPaginated[api.Tag](client, limit, func(page, perPage int) string {
				return fmt.Sprintf("/repos/%s/%s/tags?page=%d&per_page=%d", owner, repo, page, perPage)
			})
			if err != nil {
				return fmt.Errorf("failed to list tags for %s/%s: %w", owner, repo, err)
			}
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), tagsJSON(tags))
			}

			out := cmd.OutOrStdout()
			if len(tags) == 0 {
				fmt.Fprintln(out, "No tags found")
				return nil
			}

			for _, tag := range tags {
				fmt.Fprintln(out, tag.Name)
			}

			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of tags to list")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output tags as JSON")

	return cmd
}

type tagJSON struct {
	Name      string `json:"name"`
	Message   string `json:"message"`
	CommitSHA string `json:"commitSha"`
	CommitURL string `json:"commitUrl"`
	Tagger    string `json:"tagger"`
	TaggedAt  string `json:"taggedAt"`
}

func tagsJSON(tags []api.Tag) []tagJSON {
	result := make([]tagJSON, len(tags))
	for index, tag := range tags {
		result[index] = tagJSON{Name: tag.Name, Message: tag.Message, CommitSHA: tag.Commit.SHA, CommitURL: tag.Commit.URL, Tagger: tag.Tagger.Name, TaggedAt: tag.Tagger.Date}
	}
	return result
}

func newCmdTagCreate(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Message string
		Ref     string
	}

	cmd := &cobra.Command{
		Use:   "create [<owner>/<repo>] <tag_name>",
		Short: "Create a tag",
		Args:  cobra.RangeArgs(1, 2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateTagCreateRef(opts.Ref)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			tagName := strings.TrimSpace(remaining[0])
			if tagName == "" {
				return fmt.Errorf("tag name is required")
			}
			if err := validateTagCreateRef(opts.Ref); err != nil {
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

			body := api.TagRequest{
				TagName: tagName,
				Message: opts.Message,
				Refs:    opts.Ref,
			}

			var tag api.Tag
			path := fmt.Sprintf("/repos/%s/%s/tags", owner, repo)
			if err := client.Post(path, body, &tag); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created tag %s\n", tag.Name)

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Message, "message", "m", "", "Tag message")
	cmd.Flags().StringVar(&opts.Ref, "ref", "", "Branch, tag, or commit SHA to create the tag from (required)")
	_ = cmd.MarkFlagRequired("ref")

	return cmd
}

func validateTagCreateRef(ref string) error {
	if strings.TrimSpace(ref) == "" {
		return fmt.Errorf("source ref is required; pass --ref with a branch, tag, or commit SHA")
	}
	return nil
}

func newCmdTagDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete [<owner>/<repo>] <tag_name>",
		Short: "Delete a tag",
		Long: `Delete a tag from AtomGit.

By default, you will be prompted to confirm the deletion. Use --yes to skip
the confirmation prompt.`,
		Example: `  ag tag delete owner/repo v1.0.0
  ag tag delete owner/repo v1.0.0 --yes`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			tagName := strings.TrimSpace(remaining[0])
			if tagName == "" {
				return fmt.Errorf("tag name is required")
			}

			out := cmd.OutOrStdout()
			if !yes {
				confirmed, err := confirmTagDelete(cmd.InOrStdin(), out, repository, tagName)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(out, "Deletion cancelled.")
					return nil
				}
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			// Tag names may contain slashes (e.g. v1.0/rc1), so escape the name
			// before splicing it into the request path.
			path := fmt.Sprintf("/repos/%s/%s/tags/%s", owner, repo, url.PathEscape(tagName))
			if err := client.Delete(path); err != nil {
				return err
			}

			fmt.Fprintf(out, "Deleted tag %s\n", tagName)

			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func confirmTagDelete(in io.Reader, out io.Writer, repository cmdutil.Repository, tagName string) (bool, error) {
	fmt.Fprintf(out, "Delete tag %s from %s? [y/N] ", tagName, repository.String())
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("read confirmation: %w", err)
		}
		return false, nil
	}
	answer := strings.TrimSpace(scanner.Text())
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes"), nil
}
