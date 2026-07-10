package auth

import (
	"context"
	"errors"
	"fmt"
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
	cmd.AddCommand(newCmdAuthStatus(f))
	cmd.AddCommand(newCmdAuthToken(f))

	return cmd
}

func newCmdAuthLogout() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored token and user (delete credential files)",
		RunE: func(cmd *cobra.Command, args []string) error {
			removed, err := config.ClearCredentials()
			if err != nil {
				return err
			}
			if len(removed) == 0 {
				fmt.Println("✗ No credential files found (already logged out)")
				return nil
			}
			fmt.Println("✓ Logged out. Removed:")
			for _, p := range removed {
				fmt.Printf("  %s\n", p)
			}
			return nil
		},
	}
}

func newCmdAuthLogin(f *cmdutil.Factory) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with AtomGit OAuth (opens browser, saves token.json)",
		Long: `Opens a browser to authorize ag against atomgit.com, then writes
access_token and user to the XDG config path (see README).
If already logged in, skips the browser unless --force is set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				if _, err := f.Config.GetToken(); err == nil {
					user, _ := f.Config.GetUser()
					fmt.Printf("✓ Already logged in as %s — skipping browser login.\n", user)
					fmt.Println("  Use `ag auth refresh` to refresh the access token, `ag auth logout` to sign out, or `ag auth login --force` to use the browser again.")
					return nil
				}
			}

			ctx := cmd.Context()
			if _, ok := ctx.Deadline(); !ok {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 15*time.Minute)
				defer cancel()
			}

			result, err := oauth.Login(ctx)
			if err != nil {
				return err
			}
			cred := &config.StoredCredentials{
				AccessToken:  result.AccessToken,
				User:         result.Login,
				RefreshToken: result.RefreshToken,
				ExpiresIn:    result.ExpiresIn,
				TokenType:    result.TokenType,
				CreatedAt:    time.Now().Unix(),
			}
			if err := config.SaveCredentials(cred); err != nil {
				return err
			}
			path, err := config.PrimaryTokenPath()
			if err != nil {
				return err
			}
			fmt.Printf("✓ Logged in to atomgit.com as %s\n", result.Login)
			fmt.Printf("  Token saved to %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Always run browser login even if already logged in")
	return cmd
}

func newCmdAuthRefresh() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the access token using the stored refresh_token",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			cred, err := config.LoadStoredCredentials()
			if err != nil {
				if errors.Is(err, config.ErrTokenNotFound) {
					return fmt.Errorf("not logged in; run `ag auth login`")
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

			if err := config.SaveCredentials(cred); err != nil {
				return err
			}
			path, err := config.PrimaryTokenPath()
			if err != nil {
				return err
			}
			fmt.Printf("✓ Token refreshed for %s\n", cred.User)
			fmt.Printf("  Saved to %s\n", path)
			return nil
		},
	}
}

func newCmdAuthStatus(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "View authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				fmt.Println("✗ Not authenticated")
				fmt.Printf("  Token file error: %s\n", err)
				return nil
			}

			user, err := f.Config.GetUser()
			if err != nil {
				fmt.Println("✗ Token found but user not configured")
				return nil
			}

			// Mask token for display
			maskedToken := token
			if len(token) > 8 {
				maskedToken = token[:4] + "****" + token[len(token)-4:]
			}

			fmt.Printf("✓ Logged in to atomgit.com as %s\n", user)
			fmt.Printf("  Token: %s\n", maskedToken)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := f.Config.GetToken()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			fmt.Println(token)
			return nil
		},
	}

	return cmd
}
