package tag

import (
	"fmt"
	"net/url"

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
	cmd := &cobra.Command{
		Use:   "list [<owner>/<repo>]",
		Short: "List tags",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			var tags []api.Tag
			path := fmt.Sprintf("/repos/%s/%s/tags", owner, repo)
			if err := client.Get(path, &tags); err != nil {
				return err
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
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			tagName := remaining[0]

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
	cmd.Flags().StringVar(&opts.Ref, "ref", "", "The SHA value or branch name to create the tag from")

	return cmd
}

func newCmdTagDelete(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [<owner>/<repo>] <tag_name>",
		Short: "Delete a tag",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			owner, repo := repository.Owner, repository.Name
			tagName := remaining[0]

			// Tag names may contain slashes (e.g. v1.0/rc1), so escape the name
			// before splicing it into the request path.
			path := fmt.Sprintf("/repos/%s/%s/tags/%s", owner, repo, url.PathEscape(tagName))
			if err := client.Delete(path); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Deleted tag %s\n", tagName)

			return nil
		},
	}

	return cmd
}
