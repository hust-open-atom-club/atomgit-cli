package repo

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type moduleEditOptions struct {
	Wiki, Issue, Security, MergeRequest, Fork, Analysis, Discussion bool
	Yes, JSON                                                       bool
}

type moduleEditJSON struct {
	Repository      string                     `json:"repository"`
	SubmittedFields api.RepositoryModuleUpdate `json:"submitted_fields"`
	Verification    string                     `json:"verification"`
}

func newCmdRepoModule(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{Use: "module", Short: "Manage repository module switches", Long: "Edit repository module switches. Omitted switches remain unchanged; this command does not query current settings."}
	cmd.AddCommand(newCmdRepoModuleEdit(f))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func newCmdRepoModuleEdit(f *cmdutil.Factory) *cobra.Command {
	opts := &moduleEditOptions{}
	cmd := &cobra.Command{
		Use: "edit [<owner>/<repo>]", Short: "Edit repository module switches",
		Long:    "Enable or disable wiki, issue, security, merge request, fork, analysis, and discussion modules. Only explicitly supplied switches are sent; omitted fields keep their existing values. Use --issue=false to disable a module. Changes require confirmation unless --yes is supplied. Writes are never automatically retried or redirected. Success requires HTTP 200, integer code=1, and msg=success. Output describes submitted changes, not independently verified current settings; no GET module/setting request is made.",
		Example: "  ag-cli repo module edit owner/repo --wiki=true --issue=false\n  ag-cli repo module edit --discussion=true --yes --json",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MaximumNArgs(1)(cmd, args); err != nil {
				return err
			}
			return buildModuleUpdate(cmd, opts).Validate()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			request := buildModuleUpdate(cmd, opts)
			if err := request.Validate(); err != nil {
				return err
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			if !opts.Yes {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Repository: %s\nSubmitted module changes:\n", repository); err != nil {
					return err
				}
				if err := cmdutil.WriteJSON(cmd.ErrOrStderr(), request); err != nil {
					return err
				}
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), "Apply these repository module changes? [y/N] ")
				if err != nil {
					return err
				}
				if !confirmed {
					_, err := fmt.Fprintln(cmd.ErrOrStderr(), "Repository module update canceled.")
					return err
				}
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			if err := api.UpdateRepositoryModules(client.WithContext(cmd.Context()), repository.Owner, repository.Name, request); err != nil {
				return fmt.Errorf("failed to edit module settings for %s: %w", repository, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), moduleEditJSON{repository.String(), request, "api_acknowledged"})
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Updated module settings for %s (API acknowledged).\nSubmitted fields (not independently verified):\n", repository); err != nil {
				return err
			}
			return cmdutil.WriteJSON(cmd.OutOrStdout(), request)
		},
	}
	cmd.Flags().BoolVar(&opts.Wiki, "wiki", false, "Enable wiki (use =false to disable)")
	cmd.Flags().BoolVar(&opts.Issue, "issue", false, "Enable issues (use =false to disable)")
	cmd.Flags().BoolVar(&opts.Security, "security", false, "Enable security (use =false to disable)")
	cmd.Flags().BoolVar(&opts.MergeRequest, "merge-request", false, "Enable merge requests (use =false to disable)")
	cmd.Flags().BoolVar(&opts.Fork, "fork", false, "Enable forks (use =false to disable)")
	cmd.Flags().BoolVar(&opts.Analysis, "analysis", false, "Enable analysis (use =false to disable)")
	cmd.Flags().BoolVar(&opts.Discussion, "discussion", false, "Enable discussions (use =false to disable)")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip update confirmation")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output the repository and submitted fields as JSON")
	cmd.MarkFlagsOneRequired("wiki", "issue", "security", "merge-request", "fork", "analysis", "discussion")
	commandschema.Annotate(cmd, commandschema.Metadata{Positionals: &commandschema.Positionals{MinCount: 0, MaxCount: 1, Description: "Optional owner/repo; explicit repository takes precedence over Git context."}, Output: "text,json", Effects: "write", Notes: []string{"At least one module flag must be supplied; false is sent explicitly and omitted fields remain unchanged.", "Requires confirmation unless --yes is supplied. No automatic PUT retries or redirects.", "Acknowledgement requires HTTP 200, integer code=1 and msg=success; output reports submitted fields without current-settings readback."}})
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func buildModuleUpdate(cmd *cobra.Command, opts *moduleEditOptions) api.RepositoryModuleUpdate {
	r := api.RepositoryModuleUpdate{}
	for _, field := range []struct {
		flag   string
		value  *bool
		target **bool
	}{
		{"wiki", &opts.Wiki, &r.Wiki}, {"issue", &opts.Issue, &r.Issue}, {"security", &opts.Security, &r.Security},
		{"merge-request", &opts.MergeRequest, &r.MergeRequest}, {"fork", &opts.Fork, &r.Fork}, {"analysis", &opts.Analysis, &r.Analysis}, {"discussion", &opts.Discussion, &r.Discussion},
	} {
		if cmd.Flags().Changed(field.flag) {
			*field.target = field.value
		}
	}
	return r
}
