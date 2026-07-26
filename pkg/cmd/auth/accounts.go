package auth

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdAuthList() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved AtomGit accounts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			accounts, active, err := config.ListAccounts()
			if errors.Is(err, config.ErrTokenNotFound) {
				if jsonOutput {
					return cmdutil.WriteJSON(cmd.OutOrStdout(), []accountJSON{})
				}
				fmt.Fprintln(cmd.OutOrStdout(), "No saved accounts.")
				return nil
			}
			if err != nil {
				return err
			}
			if jsonOutput {
				values := make([]accountJSON, len(accounts))
				for index, account := range accounts {
					values[index] = accountJSON{Login: account.User, Active: account.Key() == active, Name: account.Name, Email: account.Email}
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), values)
			}
			for _, account := range accounts {
				marker := " "
				if account.Key() == active {
					marker = "*"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", marker, account.Key())
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output accounts as JSON")
	return cmd
}

type accountJSON struct {
	Login  string `json:"login"`
	Active bool   `json:"active"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

func newCmdAuthSwitch(f *cmdutil.Factory) *cobra.Command {
	var noGit, global bool
	var gitName, gitEmail string
	cmd := &cobra.Command{
		Use:   "switch <account>",
		Short: "Switch the active account and synchronize Git identity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := config.LoadCredentialStore()
			if err != nil {
				return err
			}
			account, err := store.ResolveAccount(args[0])
			if err != nil {
				return err
			}
			name := firstNonEmpty(strings.TrimSpace(gitName), strings.TrimSpace(account.GitName), strings.TrimSpace(account.Name))
			email := firstNonEmpty(strings.TrimSpace(gitEmail), strings.TrimSpace(account.GitEmail), strings.TrimSpace(account.Email))
			if !noGit && email == "" {
				return fmt.Errorf("account %s has no Git email; use --git-email or --no-git", account.Key())
			}
			if !noGit && name == "" {
				return fmt.Errorf("account %s has no Git name; use --git-name or --no-git", account.Key())
			}

			var rollback func() error
			if !noGit {
				rollback, err = updateGitIdentity(f, global, name, email)
				if err != nil {
					return err
				}
			}
			key, err := config.SwitchAccount(account.Key())
			if err != nil {
				if rollback != nil {
					if rollbackErr := rollback(); rollbackErr != nil {
						return fmt.Errorf("switch account: %v; additionally failed to restore Git identity: %w", err, rollbackErr)
					}
				}
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Switched active account to %s\n", key)
			if noGit {
				return nil
			}
			scope := "repository"
			if global {
				scope = "global"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated %s Git identity to %s <%s>\n", scope, name, email)
			return nil
		},
	}
	cmd.Flags().BoolVar(&noGit, "no-git", false, "Do not update Git identity")
	cmd.Flags().BoolVar(&global, "global", false, "Update global Git identity instead of the current repository")
	cmd.Flags().StringVar(&gitName, "git-name", "", "Override Git user.name for this switch")
	cmd.Flags().StringVar(&gitEmail, "git-email", "", "Override Git user.email for this switch")
	cmd.MarkFlagsMutuallyExclusive("no-git", "global")
	return cmd
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func updateGitIdentity(f *cmdutil.Factory, global bool, name, email string) (func() error, error) {
	run := func(args ...string) (string, error) {
		if f != nil && f.GitConfig != nil {
			return f.GitConfig(args...)
		}
		output, err := exec.Command("git", args...).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
		}
		return strings.TrimSpace(string(output)), nil
	}
	scope := "--local"
	if global {
		scope = "--global"
	}
	oldName, nameErr := run("config", scope, "--get", "user.name")
	oldEmail, emailErr := run("config", scope, "--get", "user.email")
	if !isMissingGitValue(nameErr) {
		return nil, fmt.Errorf("read Git user.name: %w", nameErr)
	}
	if !isMissingGitValue(emailErr) {
		return nil, fmt.Errorf("read Git user.email: %w", emailErr)
	}
	restore := func() error {
		var restoreNameErr, restoreEmailErr error
		if err := restoreGitValue(run, scope, "user.name", oldName, nameErr == nil); err != nil {
			restoreNameErr = fmt.Errorf("restore Git user.name: %w", err)
		}
		if err := restoreGitValue(run, scope, "user.email", oldEmail, emailErr == nil); err != nil {
			restoreEmailErr = fmt.Errorf("restore Git user.email: %w", err)
		}
		return errors.Join(restoreNameErr, restoreEmailErr)
	}
	if _, err := run("config", scope, "user.name", name); err != nil {
		return nil, fmt.Errorf("update Git user.name: %w", err)
	}
	if _, err := run("config", scope, "user.email", email); err != nil {
		writeErr := fmt.Errorf("update Git user.email: %w", err)
		if restoreErr := restore(); restoreErr != nil {
			return nil, errors.Join(writeErr, fmt.Errorf("rollback Git identity: %w", restoreErr))
		}
		return nil, writeErr
	}
	return restore, nil
}

func isMissingGitValue(err error) bool {
	if err == nil {
		return true
	}
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == 1
}

func restoreGitValue(run func(...string) (string, error), scope, key, value string, existed bool) error {
	if existed {
		_, err := run("config", scope, key, value)
		return err
	}
	_, err := run("config", scope, "--unset-all", key)
	if isMissingGitValue(err) {
		return nil
	}
	return err
}
