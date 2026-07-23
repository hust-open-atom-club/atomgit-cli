package label

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

var labelColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{3}([0-9a-fA-F]{3})?$`)

func newCmdLabelCreate(f *cmdutil.Factory) *cobra.Command {
	var name, color string

	cmd := &cobra.Command{
		Use:     "create [<owner>/<repo>]",
		Short:   "Create a repository label",
		Long:    "Create a repository label. AtomGit API v5 accepts a name and color for label creation.",
		Example: `  ag label create owner/repo --name bug --color "#ff0000"`,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name = strings.TrimSpace(name)
			color = strings.TrimSpace(color)
			if err := validateLabelName(name); err != nil {
				return err
			}
			if err := validateLabelColor(color); err != nil {
				return err
			}

			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}
			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/repos/%s/%s/labels", repository.Owner, repository.Name)
			var created api.Label
			if err := client.PostForm(path, url.Values{
				"name":  {name},
				"color": {color},
			}, &created); err != nil {
				return fmt.Errorf("failed to create label: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created label %q [%s]\n", created.Name, created.Color)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Label name")
	cmd.Flags().StringVar(&color, "color", "", "Label color in #RGB or #RRGGBB format")
	return cmd
}

func newCmdLabelEdit(f *cmdutil.Factory) *cobra.Command {
	var name, color string

	cmd := &cobra.Command{
		Use:     "edit [<owner>/<repo>] <name>",
		Short:   "Edit a repository label",
		Long:    "Edit a repository label. AtomGit API v5 accepts a new name and color for label updates.",
		Example: `  ag label edit owner/repo bug --name defect --color "#d73a4a"`,
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameChanged := cmd.Flags().Changed("name")
			colorChanged := cmd.Flags().Changed("color")
			if !nameChanged && !colorChanged {
				return fmt.Errorf("at least one of --name or --color must be provided")
			}

			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			originalName := strings.TrimSpace(remaining[0])
			if err := validateLabelName(originalName); err != nil {
				return fmt.Errorf("invalid existing label name: %w", err)
			}

			fields := make(map[string]string)
			if nameChanged {
				name = strings.TrimSpace(name)
				if err := validateLabelName(name); err != nil {
					return err
				}
				fields["name"] = name
			}
			if colorChanged {
				color = strings.TrimSpace(color)
				if err := validateLabelColor(color); err != nil {
					return err
				}
				fields["color"] = color
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}
			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/repos/%s/%s/labels/%s", repository.Owner, repository.Name, url.PathEscape(originalName))
			if err := client.PatchForm(path, fields, nil); err != nil {
				return fmt.Errorf("failed to edit label: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updated label %q\n", originalName)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New label name")
	cmd.Flags().StringVar(&color, "color", "", "New label color in #RGB or #RRGGBB format")
	return cmd
}

func validateLabelName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("label name is required")
	}
	return nil
}

func validateLabelColor(color string) error {
	if strings.TrimSpace(color) == "" {
		return fmt.Errorf("label color is required")
	}
	if !labelColorPattern.MatchString(color) {
		return fmt.Errorf("invalid label color %q (expected #RGB or #RRGGBB)", color)
	}
	return nil
}
