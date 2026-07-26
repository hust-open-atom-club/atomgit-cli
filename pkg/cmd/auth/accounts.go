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
					values[index] = accountJSON{Host: account.Host, Login: account.User, Active: account.Key() == active, Name: account.Name, Email: account.Email}
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
	Host   string `json:"host"`
	Login  string `json:"login"`
	Active bool   `json:"active"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

func newCmdAuthSwitch() *cobra.Command {
	return &cobra.Command{
		Use:   "switch <account>",
		Short: "Switch the active account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := config.SwitchAccount(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Switched active account to %s\n", key)
			return nil
		},
	}
}

func newCmdAuthGitSync(f *cmdutil.Factory) *cobra.Command {
	var global bool
	var gitName, gitEmail string
	cmd := &cobra.Command{
		Use:   "git-sync",
		Short: "Synchronize Git identity with the active account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := config.LoadCredentialStore()
			if err != nil {
				return err
			}
			account, err := store.ActiveAccount()
			if err != nil {
				return err
			}
			name := firstNonEmpty(strings.TrimSpace(gitName), strings.TrimSpace(account.GitName), strings.TrimSpace(account.Name))
			email := firstNonEmpty(strings.TrimSpace(gitEmail), strings.TrimSpace(account.GitEmail), strings.TrimSpace(account.Email))
			if email == "" {
				return fmt.Errorf("active account %s has no Git email; use --git-email", account.Key())
			}
			if name == "" {
				return fmt.Errorf("active account %s has no Git name; use --git-name", account.Key())
			}

			if err := updateGitIdentity(f, global, name, email); err != nil {
				return err
			}
			scope := "repository"
			if global {
				scope = "global"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated %s Git identity for %s to %s <%s>\n", scope, account.Key(), name, email)
			return nil
		},
	}
	cmd.Flags().BoolVar(&global, "global", false, "Update global Git identity instead of the current repository")
	cmd.Flags().StringVar(&gitName, "git-name", "", "Override Git user.name for this synchronization")
	cmd.Flags().StringVar(&gitEmail, "git-email", "", "Override Git user.email for this synchronization")
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

func updateGitIdentity(f *cmdutil.Factory, global bool, name, email string) error {
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
		return fmt.Errorf("read Git user.name: %w", nameErr)
	}
	if !isMissingGitValue(emailErr) {
		return fmt.Errorf("read Git user.email: %w", emailErr)
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
		return fmt.Errorf("update Git user.name: %w", err)
	}
	if _, err := run("config", scope, "user.email", email); err != nil {
		writeErr := fmt.Errorf("update Git user.email: %w", err)
		if restoreErr := restore(); restoreErr != nil {
			return errors.Join(writeErr, fmt.Errorf("rollback Git identity: %w", restoreErr))
		}
		return writeErr
	}
	return nil
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
