package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/oauth"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewCmdAuth(f *cmdutil.Factory) *cobra.Command {
	return newCmdAuthWithDeps(f, loginDeps{browserLogin: oauth.Login, validateToken: oauth.FetchUser})
}

// newCmdAuthWithDeps builds the auth command tree with injectable login
// dependencies so tests can exercise the full Execute lifecycle (including
// PreRunE wiring) without network access.
func newCmdAuthWithDeps(f *cmdutil.Factory, deps loginDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth <command>",
		Short: "Authenticate with AtomGit",
		Long:  `Manage authentication state for AtomGit.`,
	}

	cmd.AddCommand(newCmdAuthLoginWithDeps(f, deps))
	cmd.AddCommand(newCmdAuthLogout())
	cmd.AddCommand(newCmdAuthRefresh())
	cmd.AddCommand(newCmdAuthList())
	cmd.AddCommand(newCmdAuthSwitch(f))
	cmd.AddCommand(newCmdAuthStatus(f))
	cmd.AddCommand(newCmdAuthToken(f))
	for _, child := range cmd.Commands() {
		if child.Name() == "login" {
			// login migrates legacy credentials itself, only after the new
			// credentials have been validated, so a failed login never
			// rewrites the credential store.
			continue
		}
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
	return newCmdAuthLoginWithDeps(f, loginDeps{browserLogin: oauth.Login, validateToken: oauth.FetchUser})
}

// loginDeps injects the network-facing halves of login so tests can stub them.
type loginDeps struct {
	browserLogin  func(context.Context) (*oauth.LoginResult, error)
	validateToken func(ctx context.Context, token string) (*oauth.UserInfo, error)
}

func newCmdAuthLoginWithFunc(f *cmdutil.Factory, login func(context.Context) (*oauth.LoginResult, error)) *cobra.Command {
	return newCmdAuthLoginWithDeps(f, loginDeps{browserLogin: login, validateToken: oauth.FetchUser})
}

func newCmdAuthLoginWithDeps(f *cmdutil.Factory, deps loginDeps) *cobra.Command {
	var force bool
	var withToken bool
	var gitName, gitEmail string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with AtomGit OAuth (opens browser, saves token.json)",
		Args:  cobra.ExactArgs(0),
		Long: `Opens a browser to authorize ag against atomgit.com, then writes
access_token and user to the XDG config path (see README). With --with-token,
skips the browser and reads an existing access token (PAT or OAuth token)
from standard input instead — useful in sandboxes, containers, and CI where
no browser is available:

    echo "$TOKEN" | ag auth login --with-token
    ag auth login --with-token < token.txt

Piped or redirected input is read to EOF without any prompt; in an
interactive terminal a single hidden prompt is shown. The token is validated
against the AtomGit user API before it is saved. If already logged in, skips
unless --force is set. The first saved account becomes active; later logins
do not change the active account.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Runtime failures (empty stdin, validation errors, network
			// errors) must be single-line errors, not a usage dump.
			cmd.SilenceUsage = true

			out := cmd.OutOrStdout()
			if !force {
				if _, err := f.Config.GetToken(); err == nil {
					user, _ := f.Config.GetUser()
					skipped := "browser login"
					if withToken {
						skipped = "token login"
					}
					fmt.Fprintf(out, "✓ Already logged in as %s — skipping %s.\n", user, skipped)
					fmt.Fprintln(out, "  Use `ag auth refresh` if this account has a refresh token, `ag auth logout` to sign out, or `ag auth login --force` to authenticate again.")
					return nil
				}
			}

			ctx := cmd.Context()
			if _, ok := ctx.Deadline(); !ok {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 15*time.Minute)
				defer cancel()
			}

			var cred *config.StoredCredentials
			if withToken {
				token, err := readTokenFromStdin(cmd)
				if err != nil {
					return err
				}
				user, err := deps.validateToken(ctx, token)
				if err != nil {
					return fmt.Errorf("token validation failed (the token may be invalid or expired): %w", err)
				}
				if user.Login == "" {
					return fmt.Errorf("token validation failed: user API returned empty login")
				}
				cred = &config.StoredCredentials{
					AccessToken: token,
					User:        user.Login,
					Name:        user.Name,
					Email:       user.Email,
					GitName:     strings.TrimSpace(gitName),
					GitEmail:    strings.TrimSpace(gitEmail),
					CreatedAt:   time.Now().Unix(),
				}
			} else {
				result, err := deps.browserLogin(ctx)
				if err != nil {
					return err
				}
				cred = &config.StoredCredentials{
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
			}
			// Migrate legacy credentials only now that the new credentials
			// have been validated, so a failed login never rewrites the
			// credential store.
			if err := migrateLegacyCredentials(cmd, args); err != nil {
				return err
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
			fmt.Fprintf(out, "✓ Logged in to atomgit.com as %s\n", cred.User)
			fmt.Fprintf(out, "  Token saved to %s\n", path)
			if store.Active != cred.Key() {
				fmt.Fprintf(out, "  Active account remains %s\n", store.Active)
				fmt.Fprintf(out, "  Run `ag auth switch %s` to use this account\n", cred.Key())
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Always authenticate again even if already logged in")
	cmd.Flags().BoolVar(&withToken, "with-token", false, "Read an access token from standard input instead of browser OAuth")
	cmd.Flags().StringVar(&gitName, "git-name", "", "Override the Git user.name stored for this account")
	cmd.Flags().StringVar(&gitEmail, "git-email", "", "Override the Git user.email stored for this account")
	return cmd
}

// readTokenFromStdin reads an access token from the command's stdin. Piped or
// redirected input is consumed to EOF without any prompt; an interactive
// terminal gets a single hidden-input prompt so the token is not echoed.
func readTokenFromStdin(cmd *cobra.Command) (string, error) {
	in := cmd.InOrStdin()
	if file, ok := in.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		fmt.Fprint(cmd.ErrOrStderr(), "Paste your token: ")
		bytes, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(cmd.ErrOrStderr())
		if err != nil {
			return "", fmt.Errorf("read token: %w", err)
		}
		if token := strings.TrimSpace(string(bytes)); token != "" {
			return token, nil
		}
		return "", fmt.Errorf("no token provided")
	}
	bytes, err := io.ReadAll(in)
	if err != nil {
		return "", fmt.Errorf("read token from stdin: %w", err)
	}
	if token := strings.TrimSpace(string(bytes)); token != "" {
		return token, nil
	}
	return "", fmt.Errorf("no token provided on stdin")
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
					return fmt.Errorf("not authenticated: run `ag auth login`")
				}
				return err
			}
			if cred.RefreshToken == "" {
				return fmt.Errorf("no refresh_token stored for %s; this account was likely logged in via `ag auth login --with-token`, which cannot be refreshed — sign in again with a new token (`echo \"$TOKEN\" | ag auth login --with-token --force`) or run `ag auth login --force` for browser OAuth", cred.User)
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
				// GetToken already returns "not authenticated: run `ag auth login`"
				// for a missing token; returning it unchanged avoids a duplicated
				// "not authenticated: not authenticated:" prefix.
				return err
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
