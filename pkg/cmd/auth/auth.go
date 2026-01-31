package auth

import (
	"fmt"

	"github.com/shinwell/ag-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdAuth(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth <command>",
		Short: "Authenticate with AtomGit",
		Long:  `Manage authentication state for AtomGit.`,
	}

	cmd.AddCommand(newCmdAuthStatus(f))
	cmd.AddCommand(newCmdAuthToken(f))

	return cmd
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
