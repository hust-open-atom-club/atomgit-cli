package repo

import (
	"fmt"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type settingsEditOptions struct {
	Yes, JSON bool
	Request   api.UpdateRepositorySettingsRequest
}

type settingsViewJSON struct {
	Repository string                 `json:"repository"`
	Settings   api.RepositorySettings `json:"settings"`
}

type settingsEditJSON struct {
	Repository      string                              `json:"repository"`
	SubmittedFields api.UpdateRepositorySettingsRequest `json:"submitted_fields"`
	Settings        api.RepositorySettings              `json:"settings"`
	Verification    string                              `json:"verification"`
}

func newCmdRepoSettings(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{Use: "settings", Short: "View and edit general repository settings",
		Long: "Manage Actions switches, cross-repository PR triggers, fork restrictions, and branch/tag naming and creation rules.\nMetadata, PR policy, push rules, and branch/tag protection are managed by their existing commands.",
		Args: cobra.NoArgs, SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(newCmdRepoSettingsView(f), newCmdRepoSettingsEdit(f))
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 0, MaxCount: 0},
		Output:      "text", Effects: "none",
	})
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func newCmdRepoSettingsView(f *cmdutil.Factory) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{Use: "view [<owner>/<repo>]", Short: "View general repository settings",
		Long:    "Read repository settings. Missing or null response fields are shown as null, not inferred as false. Requires authentication.",
		Example: "  ag-cli repo settings view owner/repo\n  ag-cli repo settings view --json", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			settings, err := api.GetRepositorySettings(client.WithContext(cmd.Context()), repository.Owner, repository.Name)
			if err != nil {
				return fmt.Errorf("failed to view settings for %s: %w", repository, err)
			}
			if asJSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), settingsViewJSON{repository.String(), settings})
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Repository: %s\nSettings (null means unavailable):\n", repository); err != nil {
				return err
			}
			return cmdutil.WriteJSON(cmd.OutOrStdout(), settings)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output repository and settings as JSON")
	annotateRepositorySettings(cmd, "read", "Fixed JSON envelope: repository and settings; missing response values remain null.")
	return cmd
}

func newCmdRepoSettingsEdit(f *cmdutil.Factory) *cobra.Command {
	opts := &settingsEditOptions{}
	var collect []func()
	var flags []string
	cmd := &cobra.Command{Use: "edit [<owner>/<repo>]", Short: "Edit general repository settings",
		Long:    "Send only explicitly specified settings, preserving false and empty strings. At least one setting is required.\nRequires confirmation unless --yes is supplied. PUT requests are never automatically retried or redirected.\nOutput distinguishes submitted fields from response settings. Null response fields are unavailable; no independent readback is performed.\nRegex dialect and whitelist ID string format are validated by AtomGit, not rewritten by the CLI.",
		Example: "  ag-cli repo settings edit owner/repo --action-pipeline-enabled=false\n  ag-cli repo settings edit --disable-fork --block-cross-repo-pr-trigger --yes\n  ag-cli repo settings edit owner/repo --branch-name-regex '' --json --yes",
		Args:    cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			for _, bind := range collect {
				bind()
			}
			if err := opts.Request.Validate(); err != nil {
				return err
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			if !opts.Yes {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Repository: %s\nSubmitted settings:\n", repository); err != nil {
					return err
				}
				if err := cmdutil.WriteJSON(cmd.ErrOrStderr(), opts.Request); err != nil {
					return err
				}
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), "Apply these repository settings? [y/N] ")
				if err != nil {
					return err
				}
				if !confirmed {
					_, err := fmt.Fprintln(cmd.ErrOrStderr(), "Repository settings update cancelled.")
					return err
				}
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			settings, err := api.UpdateRepositorySettings(client.WithContext(cmd.Context()), repository.Owner, repository.Name, opts.Request)
			if err != nil {
				return fmt.Errorf("failed to edit settings for %s: %w", repository, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), settingsEditJSON{repository.String(), opts.Request, settings, "response_only"})
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Repository: %s\nSettings update acknowledged by API.\nSubmitted fields (not independently verified):\n", repository); err != nil {
				return err
			}
			if err := cmdutil.WriteJSON(cmd.OutOrStdout(), opts.Request); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Response settings (null means unavailable):"); err != nil {
				return err
			}
			return cmdutil.WriteJSON(cmd.OutOrStdout(), settings)
		},
	}
	bindBool := func(name, description string, target **bool) {
		var value bool
		cmd.Flags().BoolVar(&value, name, false, description)
		flags = append(flags, name)
		collect = append(collect, func() {
			if cmd.Flags().Changed(name) {
				*target = &value
			}
		})
	}
	bindString := func(name, description string, target **string) {
		var value string
		cmd.Flags().StringVar(&value, name, "", description)
		flags = append(flags, name)
		collect = append(collect, func() {
			if cmd.Flags().Changed(name) {
				*target = &value
			}
		})
	}
	r := &opts.Request
	bindBool("disable-fork", "Forbid repository forks", &r.DisableFork)
	bindBool("forbidden-developer-create-branch", "Forbid developers from creating branches", &r.ForbiddenDeveloperCreateBranch)
	bindBool("forbidden-developer-create-tag", "Forbid developers from creating tags", &r.ForbiddenDeveloperCreateTag)
	bindBool("forbidden-committer-create-branch", "Forbid committers from creating branches", &r.ForbiddenCommitterCreateBranch)
	bindString("forbidden-developer-create-branch-user-ids", "Branch-creation whitelist user IDs in the server's string format (empty clears)", &r.ForbiddenDeveloperCreateBranchUserIDs)
	bindString("branch-name-regex", "Branch name pattern in the server's regex dialect (empty clears)", &r.BranchNameRegex)
	bindString("tag-name-regex", "Tag name pattern in the server's regex dialect (empty clears)", &r.TagNameRegex)
	bindBool("generate-pre-merge-ref", "Generate pre-merge references", &r.GeneratePreMergeRef)
	bindBool("rebase-disable-trigger-webhook", "Suppress webhook events on rebase", &r.RebaseDisableTriggerWebhook)
	bindBool("open-gpg-verified", "Enable public GPG key verification", &r.OpenGPGVerified)
	bindBool("include-lfs-objects", "Include LFS objects in archive downloads", &r.IncludeLFSObjects)
	bindBool("action-pipeline-enabled", "Enable Actions pipelines", &r.ActionPipelineEnabled)
	bindBool("block-cross-repo-pr-trigger", "Block cross-repository PRs from triggering Actions", &r.BlockCrossRepoPRTrigger)
	bindBool("show-branch-creator", "Show branch creator information", &r.ShowBranchCreator)
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip update confirmation")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output submitted fields and response settings as JSON")
	cmd.MarkFlagsOneRequired(flags...)
	annotateRepositorySettings(cmd, "write", "Only explicit fields are submitted. False and empty strings are preserved. Confirmation is required unless --yes is supplied; PUT is never retried or redirected. JSON contains repository, submitted_fields, settings, and verification=response_only; the response is not an independent readback.")
	return cmd
}

func annotateRepositorySettings(cmd *cobra.Command, effects, note string) {
	commandschema.Annotate(cmd, commandschema.Metadata{Positionals: &commandschema.Positionals{MinCount: 0, MaxCount: 1, Description: "Optional owner/repo; explicit repository takes precedence over Git context."}, Output: "text,json", Effects: effects, Notes: []string{"Requires authentication. General repository settings are separate from metadata, PR policy and push rules.", note}})
}
