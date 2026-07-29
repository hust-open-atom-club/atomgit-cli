package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/oauth"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdAuth(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth <command>",
		Short: "Authenticate with AtomGit",
		Long:  `Manage authentication state for AtomGit.`,
	}

	cmd.AddCommand(newCmdAuthLogin(f))
	cmd.AddCommand(newCmdAuthLogout())
	cmd.AddCommand(newCmdAuthRefresh())
	cmd.AddCommand(newCmdAuthList())
	cmd.AddCommand(newCmdAuthSwitch(f))
	cmd.AddCommand(newCmdAuthStatus(f))
	cmd.AddCommand(newCmdAuthToken(f))
	for _, child := range cmd.Commands() {
		child.PreRunE = migrateLegacyCredentials
	}

	return cmd
}

func migrateLegacyCredentials(*cobra.Command, []string) error {
	_, err := config.MigrateCredentialStore()
	if errors.Is(err, config.ErrTokenNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("migrate credential store: %w", err)
	}
	return nil
}

func newCmdAuthLogout() *cobra.Command {
	var account string
	var all bool
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove the active or a selected stored account",
		Long: `Remove the active account or one selected with --account.
An active account can only be removed when it is the last saved account;
otherwise switch to another account first. Use --all to remove every account.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if all {
				removed, err := config.ClearCredentials()
				if err != nil {
					return err
				}
				if len(removed) == 0 {
					fmt.Fprintln(out, "✗ No credential files found (already logged out)")
					return nil
				}
				fmt.Fprintln(out, "✓ Logged out all accounts")
				return nil
			}
			store, err := config.LoadCredentialStore()
			if errors.Is(err, config.ErrTokenNotFound) {
				fmt.Fprintln(out, "✗ No credential files found (already logged out)")
				return nil
			}
			if err != nil {
				return err
			}
			selector := account
			if selector == "" {
				selector = store.Active
			}
			removed, empty, err := config.RemoveAccount(selector)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "✓ Logged out %s\n", removed)
			if empty {
				path, _ := config.PrimaryTokenPath()
				fmt.Fprintf(out, "  No saved accounts remain; removed %s\n", path)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&account, "account", "", "Account username to remove")
	cmd.Flags().BoolVar(&all, "all", false, "Remove all saved accounts")
	cmd.MarkFlagsMutuallyExclusive("account", "all")
	return cmd
}

func newCmdAuthLogin(f *cmdutil.Factory) *cobra.Command {
	return newCmdAuthLoginWithFunc(f, oauth.Login)
}

func newCmdAuthLoginWithFunc(f *cmdutil.Factory, login func(context.Context) (*oauth.LoginResult, error)) *cobra.Command {
	var force bool
	var gitName, gitEmail string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with AtomGit OAuth (opens browser, saves token.json)",
		Args:  cobra.NoArgs,
		Long: `Opens a browser to authorize ag against atomgit.com, then writes
access_token and user to the XDG config path (see README).
If already logged in, skips the browser unless --force is set. The first saved
account becomes active; later logins do not change the active account.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if !force {
				if _, err := f.Config.GetToken(); err == nil {
					user, _ := f.Config.GetUser()
					fmt.Fprintf(out, "✓ Already logged in as %s — skipping browser login.\n", user)
					fmt.Fprintln(out, "  Use `ag auth refresh` to refresh the access token, `ag auth logout` to sign out, or `ag auth login --force` to use the browser again.")
					return nil
				}
			}

			ctx := cmd.Context()
			if _, ok := ctx.Deadline(); !ok {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 15*time.Minute)
				defer cancel()
			}

			result, err := login(ctx)
			if err != nil {
				return err
			}
			cred := &config.StoredCredentials{
				AccessToken:  result.AccessToken,
				User:         result.Login,
				Name:         result.Name,
				Email:        result.Email,
				GitName:      strings.TrimSpace(gitName),
				GitEmail:     strings.TrimSpace(gitEmail),
				RefreshToken: result.RefreshToken,
				ExpiresIn:    result.ExpiresIn,
				TokenType:    result.TokenType,
				CreatedAt:    time.Now().Unix(),
			}
			if !cmd.Flags().Changed("git-name") || !cmd.Flags().Changed("git-email") {
				store, err := config.LoadCredentialStore()
				if err == nil {
					if existing, resolveErr := store.ResolveAccount(cred.Key()); resolveErr == nil {
						if !cmd.Flags().Changed("git-name") {
							cred.GitName = existing.GitName
						}
						if !cmd.Flags().Changed("git-email") {
							cred.GitEmail = existing.GitEmail
						}
					}
				} else if !errors.Is(err, config.ErrTokenNotFound) {
					return err
				}
			}
			// Logging in authenticates an account; selecting it is a separate,
			// explicit operation once another active account already exists.
			if err := config.SaveAccount(cred, false); err != nil {
				return err
			}
			store, err := config.LoadCredentialStore()
			if err != nil {
				return err
			}
			path, err := config.PrimaryTokenPath()
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "✓ Logged in to atomgit.com as %s\n", result.Login)
			fmt.Fprintf(out, "  Token saved to %s\n", path)
			if store.Active != cred.Key() {
				fmt.Fprintf(out, "  Active account remains %s\n", store.Active)
				fmt.Fprintf(out, "  Run `ag auth switch %s` to use this account\n", cred.Key())
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Always run browser login even if already logged in")
	cmd.Flags().StringVar(&gitName, "git-name", "", "Override the Git user.name stored for this account")
	cmd.Flags().StringVar(&gitEmail, "git-email", "", "Override the Git user.email stored for this account")
	return cmd
}

func newCmdAuthRefresh() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the access token using the stored refresh_token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			cred, err := config.LoadStoredCredentials()
			if err != nil {
				if errors.Is(err, config.ErrTokenNotFound) {
					return config.ErrNotAuthenticated
				}
				return err
			}
			if cred.RefreshToken == "" {
				return fmt.Errorf("no refresh_token in credential file; run `ag auth login` to sign in again (OAuth must return a refresh token)")
			}

			tok, err := oauth.RefreshAccessToken(ctx, cred.RefreshToken)
			if err != nil {
				return err
			}
			cred.AccessToken = tok.AccessToken
			if tok.RefreshToken != "" {
				cred.RefreshToken = tok.RefreshToken
			}
			cred.ExpiresIn = tok.ExpiresIn
			cred.TokenType = tok.TokenType
			cred.CreatedAt = time.Now().Unix()

			if err := config.SaveAccount(cred, false); err != nil {
				return err
			}
			path, err := config.PrimaryTokenPath()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "✓ Token refreshed for %s\n", cred.User)
			fmt.Fprintf(out, "  Saved to %s\n", path)
			return nil
		},
	}
}

func newCmdAuthStatus(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "View authentication status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				out := cmd.OutOrStdout()
				fmt.Fprintln(out, "✗ Not authenticated")
				fmt.Fprintf(out, "  Token file error: %s\n", err)
				return nil
			}

			user, err := f.Config.GetUser()
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "✗ Token found but user not configured")
				return nil
			}

			// Mask token for display
			maskedToken := token
			if len(token) > 8 {
				maskedToken = token[:4] + "****" + token[len(token)-4:]
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "✓ Logged in to atomgit.com as %s\n", user)
			fmt.Fprintf(out, "  Token: %s\n", maskedToken)
			return nil
		},
	}

	return cmd
}

func newCmdAuthToken(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Print the authentication token",
		Long:  `Display the authentication token used for AtomGit API requests.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), token)
			return nil
		},
	}

	return cmd
}
