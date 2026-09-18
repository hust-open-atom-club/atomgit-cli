package user

import (
	"fmt"
	"net/url"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type editOptions struct {
	Avatar        string
	Nickname      string
	Company       string
	Description   string
	Email         string
	GitHubAccount string
	Website       string
	Location      string
	JSON          bool
}

func newCmdUserEdit(f *cmdutil.Factory) *cobra.Command {
	opts := &editOptions{}
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit the authenticated user's profile",
		Long: `Edit supported profile fields for the authenticated AtomGit user.

Only flags explicitly provided are sent to AtomGit. Pass an empty string to
clear a supported field. This command does not upload avatar files, verify
email addresses, or rename the account login.`,
		Example: `  ag user edit --nickname "Alice" --company "Example Inc."
  ag user edit --description "" --location "Wuhan"
  ag user edit --website "https://example.com" --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			request, err := buildUserEditRequest(cmd, opts)
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("failed to update user profile: %w", cmdutil.AuthenticationError(err))
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return fmt.Errorf("failed to update user profile: %w", err)
			}

			var updated api.UpdatedUserProfile
			if err := client.Patch("/user", request, &updated); err != nil {
				return fmt.Errorf("failed to update user profile: %w", err)
			}

			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), updated)
			}

			login := strings.TrimSpace(updated.Login)
			if login == "" {
				return fmt.Errorf("failed to update user profile: API response did not include a login")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Updated user profile for %s\n", login)
			fmt.Fprintf(cmd.OutOrStdout(), "  URL: %s\n", updatedUserProfileURL(f.Config.GetHost(), login))
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Avatar, "avatar", "", "Avatar URL")
	cmd.Flags().StringVar(&opts.Nickname, "nickname", "", "Profile nickname")
	cmd.Flags().StringVar(&opts.Company, "company", "", "Company name")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Profile description")
	cmd.Flags().StringVar(&opts.Email, "email", "", "Public email address")
	cmd.Flags().StringVar(&opts.GitHubAccount, "github-account", "", "GitHub account name")
	cmd.Flags().StringVar(&opts.Website, "website", "", "Website URL")
	cmd.Flags().StringVar(&opts.Location, "location", "", "Profile location")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output the updated profile as JSON")
	return cmd
}

func buildUserEditRequest(cmd *cobra.Command, opts *editOptions) (api.UpdateUserProfileRequest, error) {
	request := api.UpdateUserProfileRequest{}
	changed := false
	set := func(flag string, value string, target **string) {
		if cmd.Flags().Changed(flag) {
			valueCopy := value
			*target = &valueCopy
			changed = true
		}
	}

	set("avatar", opts.Avatar, &request.Avatar)
	set("nickname", opts.Nickname, &request.Nickname)
	set("company", opts.Company, &request.Company)
	set("description", opts.Description, &request.Description)
	set("email", opts.Email, &request.Email)
	set("github-account", opts.GitHubAccount, &request.GitHubAccount)
	set("website", opts.Website, &request.Website)
	set("location", opts.Location, &request.Location)
	if !changed {
		return api.UpdateUserProfileRequest{}, fmt.Errorf("at least one of --avatar, --nickname, --company, --description, --email, --github-account, --website, or --location must be provided")
	}
	return request, nil
}

func updatedUserProfileURL(host, login string) string {
	if strings.TrimSpace(host) == "" {
		host = "atomgit.com"
	}
	return "https://" + host + "/" + url.PathEscape(login)
}
