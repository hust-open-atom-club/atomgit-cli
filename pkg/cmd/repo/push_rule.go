package repo

import (
	"fmt"
	"io"
	"net/url"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type pushRuleEditOptions struct {
	RejectNotSignedByGPG bool
	CommitMessageRegex   string
	MaxFileSize          int
	SkipRuleForOwner     bool
	DenyForcePush        bool
	Yes                  bool
	JSON                 bool
}

type pushRuleJSON struct {
	Repository           string `json:"repository"`
	RejectNotSignedByGPG bool   `json:"reject_not_signed_by_gpg"`
	CommitMessageRegex   string `json:"commit_message_regex"`
	MaxFileSize          int    `json:"max_file_size"`
	SkipRuleForOwner     bool   `json:"skip_rule_for_owner"`
	DenyForcePush        bool   `json:"deny_force_push"`
}

type pushRuleEditJSON struct {
	Repository    string                              `json:"repository"`
	ChangedFields api.UpdateRepositoryPushRuleRequest `json:"changed_fields"`
}

func newCmdRepoPushRule(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push-rule",
		Short: "Manage repository push rules",
		Long: `View and edit repository-wide push rules.

These rules cover signed commits, commit message validation, maximum file
size, owner exemptions, and force pushes. Branch and tag protection rules are
managed separately.`,
	}
	cmd.AddCommand(newCmdRepoPushRuleView(f))
	cmd.AddCommand(newCmdRepoPushRuleEdit(f))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func newCmdRepoPushRuleView(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "view [<owner>/<repo>]",
		Short:   "View repository push rules",
		Example: "  ag repo push-rule view owner/repo\n  ag repo push-rule view --json",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := pushRuleAPIClient(f, repository)
			if err != nil {
				return err
			}
			rule, err := getRepositoryPushRule(client, repository)
			if err != nil {
				if api.IsHTTPStatus(err, 404) {
					return fmt.Errorf("push rules were not found for %s: %w", repository, err)
				}
				return fmt.Errorf("failed to view push rules for %s: %w", repository, err)
			}

			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), newPushRuleJSON(repository, rule))
			}
			printPushRule(cmd.OutOrStdout(), repository, rule)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output push rules as JSON")
	return cmd
}

func newCmdRepoPushRuleEdit(f *cmdutil.Factory) *cobra.Command {
	opts := &pushRuleEditOptions{}
	cmd := &cobra.Command{
		Use:   "edit [<owner>/<repo>]",
		Short: "Edit repository push rules",
		Long: `Edit repository-wide push rules.

Only flags explicitly provided are sent to AtomGit; omitted settings remain
unchanged. Explicit false, empty-string, and zero values are preserved. All
updates require confirmation unless --yes is supplied.`,
		Example: `  ag repo push-rule edit owner/repo --deny-force-push --yes
  ag repo push-rule edit --commit-message-regex '^(feat|fix): '
  ag repo push-rule edit owner/repo --reject-not-signed-by-gpg=false --max-file-size 0`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			request, err := buildPushRuleEditRequest(cmd, opts)
			if err != nil {
				return err
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := pushRuleAPIClient(f, repository)
			if err != nil {
				return err
			}

			if !opts.Yes {
				confirmed, err := confirmPushRuleEdit(cmd.InOrStdin(), cmd.ErrOrStderr(), repository, request)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.ErrOrStderr(), "Push-rule update cancelled.")
					return nil
				}
			}

			if err := updateRepositoryPushRule(client, repository, request); err != nil {
				if api.IsHTTPStatus(err, 404) {
					return fmt.Errorf("push rules were not found for %s: %w", repository, err)
				}
				return fmt.Errorf("failed to edit push rules for %s: %w", repository, err)
			}

			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), pushRuleEditJSON{
					Repository:    repository.String(),
					ChangedFields: request,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated push rules for %s\n", repository)
			printPushRuleChanges(cmd.OutOrStdout(), request)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.RejectNotSignedByGPG, "reject-not-signed-by-gpg", false, "Require commits to have verified GPG signatures")
	cmd.Flags().StringVar(&opts.CommitMessageRegex, "commit-message-regex", "", "Regular expression required for commit messages (empty disables it)")
	cmd.Flags().IntVar(&opts.MaxFileSize, "max-file-size", 0, "Maximum committed file size in MB (0 disables the limit)")
	cmd.Flags().BoolVar(&opts.SkipRuleForOwner, "skip-rule-for-owner", false, "Exempt repository administrators from applicable push rules")
	cmd.Flags().BoolVar(&opts.DenyForcePush, "deny-force-push", false, "Deny force pushes, including from administrators")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip update confirmation")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output the updated fields as JSON")
	return cmd
}

func buildPushRuleEditRequest(cmd *cobra.Command, opts *pushRuleEditOptions) (api.UpdateRepositoryPushRuleRequest, error) {
	request := api.UpdateRepositoryPushRuleRequest{}
	if cmd.Flags().Changed("reject-not-signed-by-gpg") {
		request.RejectNotSignedByGPG = &opts.RejectNotSignedByGPG
	}
	if cmd.Flags().Changed("commit-message-regex") {
		request.CommitMessageRegex = &opts.CommitMessageRegex
	}
	if cmd.Flags().Changed("max-file-size") {
		if opts.MaxFileSize < 0 {
			return request, fmt.Errorf("maximum file size cannot be negative")
		}
		request.MaxFileSize = &opts.MaxFileSize
	}
	if cmd.Flags().Changed("skip-rule-for-owner") {
		request.SkipRuleForOwner = &opts.SkipRuleForOwner
	}
	if cmd.Flags().Changed("deny-force-push") {
		request.DenyForcePush = &opts.DenyForcePush
	}
	if !pushRuleRequestChanged(request) {
		return request, fmt.Errorf("at least one push-rule setting must be provided")
	}
	return request, nil
}

func pushRuleRequestChanged(request api.UpdateRepositoryPushRuleRequest) bool {
	return request.RejectNotSignedByGPG != nil ||
		request.CommitMessageRegex != nil ||
		request.MaxFileSize != nil ||
		request.SkipRuleForOwner != nil ||
		request.DenyForcePush != nil
}

func pushRuleAPIClient(f *cmdutil.Factory, repository cmdutil.Repository) (*api.Client, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to access push rules for %s: %w", repository, cmdutil.AuthenticationError(err))
	}
	client, err := f.NewAPIClient(token)
	if err != nil {
		return nil, fmt.Errorf("failed to access push rules for %s: %w", repository, err)
	}
	return client, nil
}

func repositoryPushRulePath(repository cmdutil.Repository) string {
	return fmt.Sprintf("/repos/%s/%s/push_config", url.PathEscape(repository.Owner), url.PathEscape(repository.Name))
}

func getRepositoryPushRule(client *api.Client, repository cmdutil.Repository) (api.RepositoryPushRule, error) {
	var rule api.RepositoryPushRule
	if err := client.Get(repositoryPushRulePath(repository), &rule); err != nil {
		return api.RepositoryPushRule{}, err
	}
	return rule, nil
}

func updateRepositoryPushRule(client *api.Client, repository cmdutil.Repository, request api.UpdateRepositoryPushRuleRequest) error {
	return client.Put(repositoryPushRulePath(repository), request, nil)
}

func newPushRuleJSON(repository cmdutil.Repository, rule api.RepositoryPushRule) pushRuleJSON {
	return pushRuleJSON{
		Repository:           repository.String(),
		RejectNotSignedByGPG: rule.RejectNotSignedByGPG.Bool(),
		CommitMessageRegex:   rule.CommitMessageRegex,
		MaxFileSize:          rule.MaxFileSize,
		SkipRuleForOwner:     rule.SkipRuleForOwner.Bool(),
		DenyForcePush:        rule.DenyForcePush.Bool(),
	}
}

func printPushRule(out io.Writer, repository cmdutil.Repository, rule api.RepositoryPushRule) {
	fmt.Fprintf(out, "Repository: %s\n", repository)
	fmt.Fprintf(out, "Reject unsigned commits: %t\n", rule.RejectNotSignedByGPG.Bool())
	fmt.Fprintf(out, "Commit message regex: %q\n", rule.CommitMessageRegex)
	fmt.Fprintf(out, "Maximum file size (MB): %d\n", rule.MaxFileSize)
	fmt.Fprintf(out, "Skip rules for repository owner: %t\n", rule.SkipRuleForOwner.Bool())
	fmt.Fprintf(out, "Deny force pushes: %t\n", rule.DenyForcePush.Bool())
}

func printPushRuleChanges(out io.Writer, request api.UpdateRepositoryPushRuleRequest) {
	fmt.Fprintln(out, "Changed fields:")
	if request.RejectNotSignedByGPG != nil {
		fmt.Fprintf(out, "  reject_not_signed_by_gpg: %t\n", *request.RejectNotSignedByGPG)
	}
	if request.CommitMessageRegex != nil {
		fmt.Fprintf(out, "  commit_message_regex: %q\n", *request.CommitMessageRegex)
	}
	if request.MaxFileSize != nil {
		fmt.Fprintf(out, "  max_file_size: %d\n", *request.MaxFileSize)
	}
	if request.SkipRuleForOwner != nil {
		fmt.Fprintf(out, "  skip_rule_for_owner: %t\n", *request.SkipRuleForOwner)
	}
	if request.DenyForcePush != nil {
		fmt.Fprintf(out, "  deny_force_push: %t\n", *request.DenyForcePush)
	}
}

func confirmPushRuleEdit(in io.Reader, out io.Writer, repository cmdutil.Repository, request api.UpdateRepositoryPushRuleRequest) (bool, error) {
	fmt.Fprintf(out, "Repository: %s\n", repository)
	printPushRuleChanges(out, request)
	return cmdutil.Confirm(in, out, "Apply these push-rule changes? [y/N] ")
}
