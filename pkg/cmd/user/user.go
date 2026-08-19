package user

import (
	"fmt"
	"net/url"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdUser(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "View AtomGit user profiles",
		Long:  "View the authenticated user's profile or a public profile.",
	}
	cmd.AddCommand(newCmdUserView(f))
	return cmd
}

func newCmdUserView(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	var webOutput bool
	cmd := &cobra.Command{
		Use:   "view [<login>]",
		Short: "View the current user or a public user profile",
		Example: `  ag user view
  ag user view alice
  ag user view alice --json
  ag user view alice --web`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			login := ""
			if len(args) == 1 {
				login = strings.TrimSpace(args[0])
				if !validLogin(login) {
					return fmt.Errorf("invalid login %q: expected a non-empty user name without path or query separators", args[0])
				}
			}

			// A public profile (/users/:login) is anonymously accessible, so
			// only the authenticated profile (/user) requires a token.
			var token string
			if login == "" {
				var err error
				token, err = f.Config.GetToken()
				if err != nil {
					return err
				}
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			var user api.User
			if login == "" {
				if err := client.Get("/user", &user); err != nil {
					return fmt.Errorf("view current user: %w", err)
				}
			} else {
				escapedLogin := url.PathEscape(login)
				if err := client.Get("/users/"+escapedLogin, &user); err != nil {
					return fmt.Errorf("view user %q: %w", login, err)
				}
			}
			if strings.TrimSpace(user.Login) == "" {
				return fmt.Errorf("view user: API response did not include a login")
			}

			if webOutput {
				profileURL := userProfileURL(user)
				fmt.Fprintf(cmd.OutOrStdout(), "Opening %s in your browser.\n", profileURL)
				if f.BrowserOpener != nil {
					if err := f.BrowserOpener(profileURL); err != nil {
						return fmt.Errorf("failed to open browser: %w", err)
					}
				}
				return nil
			}

			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), newUserJSON(user))
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Login: %s\n", user.Login)
			fmt.Fprintf(out, "Name:  %s\n", user.Name)
			fmt.Fprintf(out, "Email: %s\n", user.Email)
			fmt.Fprintf(out, "URL:   %s\n", user.HTMLURL)
			fmt.Fprintf(out, "Type:  %s\n", user.Type)
			fmt.Fprintf(out, "Bio:   %s\n", user.Bio)
			fmt.Fprintf(out, "Company: %s\n", user.Company)
			fmt.Fprintf(out, "Website: %s\n", user.Website)
			fmt.Fprintf(out, "Location: %s\n", user.Location)
			fmt.Fprintf(out, "Followers: %d\n", user.Followers)
			fmt.Fprintf(out, "Following: %d\n", user.Following)
			if len(user.TopLanguages) > 0 {
				fmt.Fprintf(out, "Top languages: %s\n", strings.Join(user.TopLanguages, ", "))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output the user profile as JSON")
	cmd.Flags().BoolVarP(&webOutput, "web", "w", false, "Open the user profile in the browser")
	cmd.MarkFlagsMutuallyExclusive("web", "json")
	return cmd
}

// validLogin reports whether login is safe to embed in a URL path segment.
// Whitespace, control characters, and path/query separators are rejected
// before any HTTP request is made so a crafted argument cannot redirect the
// request to a different endpoint.
func validLogin(login string) bool {
	if login == "" {
		return false
	}
	for _, r := range login {
		if r <= ' ' || r == 0x7f {
			return false
		}
	}
	return !strings.ContainsAny(login, "/?#")
}

// userProfileURL returns the canonical profile page for user, falling back to
// a constructed URL when the API response lacks html_url.
func userProfileURL(user api.User) string {
	if u := strings.TrimSpace(user.HTMLURL); u != "" {
		return u
	}
	return "https://atomgit.com/" + url.PathEscape(user.Login)
}

// userJSON is the stable, documented JSON schema emitted by `ag user view
// --json`. Fields are always present so automation can distinguish a real
// zero/empty value from a missing field.
type userJSON struct {
	ID           string   `json:"id"`
	Login        string   `json:"login"`
	Name         string   `json:"name"`
	Email        string   `json:"email"`
	URL          string   `json:"url"`
	Type         string   `json:"type"`
	Bio          string   `json:"bio"`
	Company      string   `json:"company"`
	Website      string   `json:"website"`
	Location     string   `json:"location"`
	Followers    int      `json:"followers"`
	Following    int      `json:"following"`
	TopLanguages []string `json:"topLanguages"`
}

func newUserJSON(user api.User) userJSON {
	topLanguages := user.TopLanguages
	if topLanguages == nil {
		topLanguages = []string{}
	}
	return userJSON{
		ID:           user.ID,
		Login:        user.Login,
		Name:         user.Name,
		Email:        user.Email,
		URL:          user.HTMLURL,
		Type:         user.Type,
		Bio:          user.Bio,
		Company:      user.Company,
		Website:      user.Website,
		Location:     user.Location,
		Followers:    user.Followers,
		Following:    user.Following,
		TopLanguages: topLanguages,
	}
}
