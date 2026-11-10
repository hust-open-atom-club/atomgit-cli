package repo

import (
	"fmt"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func newCmdRepoCLA(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{Use: "cla", Short: "Manage repository CLA configuration",
		Long: "List available contributor license agreements, select an agreement, or disable CLA.\nThe available list does not describe the repository's current configuration."}
	cmd.AddCommand(newCmdRepoCLAList(f), newCmdRepoCLAWrite(f, false), newCmdRepoCLAWrite(f, true))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func newCmdRepoCLAList(f *cmdutil.Factory) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{Use: "list [<owner>/<repo>]", Short: "List available CLAs",
		Long:    "List available CLAs, not the repository's current CLA configuration.",
		Example: "  ag-cli repo cla list owner/repo\n  ag-cli repo cla list --json", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			agreements, err := api.ListRepositoryCLAs(client.WithContext(cmd.Context()), repository.Owner, repository.Name)
			if err != nil {
				return fmt.Errorf("failed to list available CLAs for %s: %w", repository, err)
			}
			if asJSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), agreements)
			}
			for _, agreement := range agreements {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", cmdutil.EscapeTSVField(agreement.ID), cmdutil.EscapeTSVField(agreement.Name)); err != nil {
					return err
				}
			}
			return nil
		}}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output available CLAs as JSON")
	annotateCLA(cmd, "read", "Lists available agreements, not current configuration.")
	return cmd
}

func newCmdRepoCLAWrite(f *cmdutil.Factory, disable bool) *cobra.Command {
	var id string
	var enforce, yes bool
	action, short := "set", "Select and enable a repository CLA"
	if disable {
		action, short = "disable", "Disable repository CLA"
	}
	cmd := &cobra.Command{Use: action + " [<owner>/<repo>]", Short: short,
		Long: short + ". Requires confirmation unless --yes is supplied.\nWrites are never automatically retried or redirected. Check AtomGit before retrying a failed update.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !disable && strings.TrimSpace(id) == "" {
				return fmt.Errorf("--cla-id must not be empty")
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			var check *bool
			if cmd.Flags().Changed("enforce-check") {
				check = &enforce
			}
			if !yes {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Repository: %s\nCLA action: %s\n", repository, action); err != nil {
					return err
				}
				if !disable {
					if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "CLA ID: %s\n", id); err != nil {
						return err
					}
					if check != nil {
						if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Enforce check: %t\n", *check); err != nil {
							return err
						}
					}
				}
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), "Apply this CLA configuration? [y/N] ")
				if err != nil {
					return err
				}
				if !confirmed {
					_, err := fmt.Fprintln(cmd.ErrOrStderr(), "CLA update cancelled.")
					return err
				}
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			client = client.WithContext(cmd.Context())
			if disable {
				err = api.DisableRepositoryCLA(client, repository.Owner, repository.Name)
			} else {
				err = api.SetRepositoryCLA(client, repository.Owner, repository.Name, id, check)
			}
			if err != nil {
				return fmt.Errorf("failed to %s CLA for %s: %w", action, repository, err)
			}
			if disable {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Disabled CLA for %s\n", repository)
			} else {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Set CLA %s for %s\n", id, repository)
			}
			return err
		}}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip update confirmation")
	if !disable {
		cmd.Flags().StringVar(&id, "cla-id", "", "Available CLA ID (required)")
		_ = cmd.MarkFlagRequired("cla-id")
		cmd.Flags().BoolVar(&enforce, "enforce-check", false, "Prevent merging PRs without a completed CLA check (only sent when specified)")
		cmd.Example = "  ag-cli repo cla set owner/repo --cla-id agreement-id --enforce-check=false --yes"
	} else {
		cmd.Example = "  ag-cli repo cla disable owner/repo --yes"
	}
	annotateCLA(cmd, "write", "Requires confirmation unless --yes is supplied. No automatic write retries or redirects.")
	return cmd
}

func annotateCLA(cmd *cobra.Command, effects, note string) {
	output := "text"
	if cmd.Flags().Lookup("json") != nil {
		output = "text,json"
	}
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 0, MaxCount: 1, Description: "Optional owner/repo; inferred from Git context when omitted."},
		Output:      output, Effects: effects, Notes: []string{note},
	})
}
