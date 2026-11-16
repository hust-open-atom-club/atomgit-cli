package repo

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type archiveOptions struct{ Yes, PasswordStdin, WithPassword bool }

func newCmdRepoArchive(f *cmdutil.Factory, archived bool) *cobra.Command {
	action, state, status := "archive", "archived", api.OrganizationRepositoryArchived
	if !archived {
		action, state, status = "unarchive", "active", api.OrganizationRepositoryActive
	}
	opts := &archiveOptions{}
	cmd := &cobra.Command{
		Use:          action + " [<org>/<repo>]",
		Short:        map[bool]string{true: "Archive an organization repository", false: "Unarchive an organization repository"}[archived],
		SilenceUsage: true,
		Long:         "Set the archive state of an organization-owned repository. Personal repositories are not supported by this endpoint. Archive sends integer status=2; unarchive sends status=0. The repository and target state are shown before confirmation; --yes skips only confirmation. Password is optional: use --with-password for hidden terminal input, or --yes --password-stdin for standard input. No plaintext password flag is provided. Writes are never automatically retried or redirected. Success requires HTTP 200 and business code 0; an uncertain result must be checked on AtomGit before retrying.",
		Example: `  ag-cli repo ` + action + ` my-org/demo
  ag-cli repo ` + action + ` my-org/demo --yes
  ag-cli repo ` + action + ` my-org/demo --with-password
  printf '%s\n' "$PASSWORD" | ag-cli repo ` + action + ` my-org/demo --yes --password-stdin`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			if opts.PasswordStdin && !opts.Yes {
				return fmt.Errorf("--password-stdin requires --yes because standard input cannot also answer confirmation")
			}
			if opts.WithPassword && opts.PasswordStdin {
				return fmt.Errorf("--with-password and --password-stdin are mutually exclusive")
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Repository: %s\nTarget archive state: %s (status=%d)\n", repository, state, status); err != nil {
				return err
			}
			if !opts.Yes {
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), fmt.Sprintf("%s organization repository %s? [y/N]: ", action, repository))
				if err != nil {
					return err
				}
				if !confirmed {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Repository archive update canceled.")
					return err
				}
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			var source api.Repository
			if err := client.Get(api.RepositoryPath(repository.Owner, repository.Name), &source); err != nil {
				return fmt.Errorf("inspect organization repository %s: %w", repository, err)
			}
			if err := validateArchiveRepository(source, repository); err != nil {
				return err
			}
			var password *string
			if opts.WithPassword || opts.PasswordStdin {
				value, err := readRepositoryPassword(cmd, opts.PasswordStdin, "organization repository archive")
				if err != nil {
					return err
				}
				password = &value
			}
			if err := api.SetOrganizationRepositoryArchiveStatus(client, repository.Owner, repository.Name, status, password); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "Archive state could not be confirmed; check AtomGit before retrying.")
				return fmt.Errorf("failed to %s organization repository %s: %w", action, repository, err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Organization repository %s: %s (API acknowledged status=%d).\n", repository, state, status)
			return err
		},
	}
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip the confirmation prompt")
	cmd.Flags().BoolVar(&opts.WithPassword, "with-password", false, "Read an optional account password from a hidden terminal prompt")
	cmd.Flags().BoolVar(&opts.PasswordStdin, "password-stdin", false, "Read an optional account password from standard input (requires --yes)")
	cmd.MarkFlagsMutuallyExclusive("with-password", "password-stdin")
	cmdutil.AddRepositoryContextHelp(cmd)
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 0, MaxCount: 1, Description: "Optional org/repo, otherwise inferred from Git. Only organization-owned repositories are supported."},
		Output:      "text", Effects: "write",
		Notes: []string{"Requires authentication and confirmation unless --yes is supplied. Archive sends status=2; unarchive sends status=0. Password is optional; --with-password uses hidden terminal input, --password-stdin requires --yes. Personal or unverifiable repository ownership is rejected. Writes are not retried or redirected. Success requires HTTP 200 and code=0, not an independent archive-state readback."},
	})
	return cmd
}

func validateArchiveRepository(source api.Repository, target cmdutil.Repository) error {
	path := source.Path
	if path == "" {
		path = source.Name
	}
	if source.ID <= 0 || !strings.EqualFold(path, target.Name) {
		return fmt.Errorf("cannot verify repository identity for %s", target)
	}
	kind, err := classifyRepositoryOwner(source, target.Owner)
	if err != nil {
		return fmt.Errorf("cannot verify organization ownership for %s: %w", target, err)
	}
	if kind != repositoryOwnerOrganization {
		return fmt.Errorf("%s is not an organization-owned repository; archive/unarchive only support organization repositories", target)
	}
	if strings.TrimSpace(source.Namespace.Path) == "" && !strings.EqualFold(strings.TrimSpace(source.Owner.Login), target.Owner) {
		return fmt.Errorf("repository owner does not match organization %q", target.Owner)
	}
	return nil
}
