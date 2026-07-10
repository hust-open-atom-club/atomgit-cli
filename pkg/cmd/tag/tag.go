package tag

import (
	"fmt"
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

	return cmd
}

func newCmdTagList(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [<owner>/]<repo>",
		Short: "List tags",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client := api.NewClient(token)

			var owner, repo string
			if len(args) == 0 {
				return fmt.Errorf("repository required")
			}

			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
			}
			owner, repo = parts[0], parts[1]

			var tags []api.Tag
			path := fmt.Sprintf("/repos/%s/%s/tags", owner, repo)
			if err := client.Get(path, &tags); err != nil {
				return err
			}

			if len(tags) == 0 {
				fmt.Println("No tags found")
				return nil
			}

			for _, tag := range tags {
				fmt.Printf("%s\n", tag.Name)
			}

			return nil
		},
	}

	return cmd
}

func newCmdTagCreate(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Message string
		Ref     string
	}

	cmd := &cobra.Command{
		Use:   "create [<owner>/]<repo> <tag_name>",
		Short: "Create a tag",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client := api.NewClient(token)

			var owner, repo, tagName string
			if len(args) == 2 {
				parts := strings.Split(args[0], "/")
				if len(parts) != 2 {
					return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
				}
				owner, repo = parts[0], parts[1]
				tagName = args[1]
			} else {
				return fmt.Errorf("repository and tag name required")
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

			fmt.Printf("Created tag %s\n", tag.Name)

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Message, "message", "m", "", "Tag message")
	cmd.Flags().StringVar(&opts.Ref, "ref", "", "The SHA value or branch name to create the tag from")

	return cmd
}

func newCmdTagDelete(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [<owner>/]<repo> <tag_name>",
		Short: "Delete a tag",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			client := api.NewClient(token)

			var owner, repo, tagName string
			if len(args) == 2 {
				parts := strings.Split(args[0], "/")
				if len(parts) != 2 {
					return fmt.Errorf("invalid repository format: %s (expected owner/repo)", args[0])
				}
				owner, repo = parts[0], parts[1]
				tagName = args[1]
			} else {
				return fmt.Errorf("repository and tag name required")
			}

			path := fmt.Sprintf("/repos/%s/%s/tags/%s", owner, repo, tagName)
			if err := client.Delete(path); err != nil {
				return err
			}

			fmt.Printf("Deleted tag %s\n", tagName)

			return nil
		},
	}

	return cmd
}
