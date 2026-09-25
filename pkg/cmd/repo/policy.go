package repo

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type policyEditOptions struct {
	section     string
	yes, json   bool
	permission  api.UpdatePermissionPolicyRequest
	codeReview  api.UpdateCodeReviewPolicyRequest
	pullRequest api.UpdatePullRequestPolicyRequest
}
type policyFlag struct {
	name, section string
	assign        func()
}
type policyViewJSON struct {
	Repository string         `json:"repository"`
	Settings   map[string]any `json:"settings"`
}
type policyEditJSON struct {
	Repository    string `json:"repository"`
	Section       string `json:"section"`
	ChangedFields any    `json:"changed_fields"`
}

func newCmdRepoPolicy(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{Use: "policy", Short: "View and edit repository policy settings",
		Long: "Manage permission mode, code-review defaults, and pull-request settings.\nPush rules and branch/tag protection are managed separately."}
	cmd.AddCommand(newCmdRepoPolicyView(f), newCmdRepoPolicyEdit(f))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}
func validPolicySection(section string) bool {
	return section == "permission" || section == "code-review" || section == "pull-request"
}
func policyAPIClient(f *cmdutil.Factory) (*api.Client, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, cmdutil.AuthenticationError(err)
	}
	return f.NewAPIClient(token)
}
func newCmdRepoPolicyView(f *cmdutil.Factory) *cobra.Command {
	var section string
	var jsonOutput bool
	cmd := &cobra.Command{Use: "view [<owner>/<repo>]", Short: "View repository policy settings",
		Long:    "View all three sections, or select one with --section.\nCode-review defaults are read from pull_request_settings, not GET /reviewer.\nJSON uses repository and settings keys; unavailable optional fields are null.",
		Example: "  ag repo policy view owner/repo\n  ag repo policy view --section permission --json",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if section != "" && !validPolicySection(section) {
				return fmt.Errorf("invalid policy section %q", section)
			}
			if cmd.Flags().Changed("section") && section == "" {
				return fmt.Errorf("--section cannot be empty")
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := policyAPIClient(f)
			if err != nil {
				return err
			}
			result := policyViewJSON{Repository: repository.String(), Settings: map[string]any{}}
			if section == "" || section == "permission" {
				value, err := api.GetPermissionPolicy(client, repository.Owner, repository.Name)
				if err != nil {
					return fmt.Errorf("failed to view permission policy for %s: %w", repository, err)
				}
				result.Settings["permission"] = value
			}
			if section != "permission" {
				value, err := api.GetPullRequestPolicy(client, repository.Owner, repository.Name)
				if err != nil {
					return fmt.Errorf("failed to view pull-request policy for %s: %w", repository, err)
				}
				if section == "" || section == "pull-request" {
					result.Settings["pull-request"] = value
				}
				if section == "" || section == "code-review" {
					review, err := api.CodeReviewPolicyFrom(value)
					if err != nil {
						return fmt.Errorf("failed to view code-review policy for %s: %w", repository, err)
					}
					result.Settings["code-review"] = review
				}
			}
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), result)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Repository: %s\n", repository)
			for _, name := range []string{"permission", "code-review", "pull-request"} {
				if value, ok := result.Settings[name]; ok {
					fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", name)
					if err := printPolicyFields(cmd.OutOrStdout(), value); err != nil {
						return err
					}
				}
			}
			return nil
		}}
	cmd.Flags().StringVar(&section, "section", "", "Section: permission, code-review, or pull-request (default: all)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output policy settings as JSON")
	return cmd
}
func newCmdRepoPolicyEdit(f *cmdutil.Factory) *cobra.Command {
	opts := &policyEditOptions{}
	var bindings []policyFlag
	cmd := &cobra.Command{Use: "edit [<owner>/<repo>]", Short: "Edit one repository policy section",
		Long:    "Send only explicitly supplied settings. False, zero, and empty strings are preserved.\nAll changes require confirmation unless --yes is supplied.\nCode-review assignees/testers are usernames; pull-request approver/tester IDs are IDs.\nPermission mode must be 1 (inherited) or 2 (independent).\nApproval-required-reviewers must be 0..5; other counts must be nonnegative.\nMerge-method: merge, rebase_merge, ff. Merged-commit-author: merged_by, created_by.",
		Example: "  ag repo policy edit owner/repo --section permission --mode 2 --yes\n  ag repo policy edit --section code-review --assignees alice,bob --testers-number 0\n  ag repo policy edit owner/repo --section pull-request --can-force-merge=false --yes",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !validPolicySection(opts.section) {
				return fmt.Errorf("--section must be permission, code-review, or pull-request")
			}
			changed := 0
			for _, binding := range bindings {
				if !cmd.Flags().Changed(binding.name) {
					continue
				}
				if binding.section != opts.section {
					return fmt.Errorf("--%s belongs to section %s, not %s", binding.name, binding.section, opts.section)
				}
				if err := validatePolicyFlag(cmd, binding.name); err != nil {
					return err
				}
				binding.assign()
				changed++
			}
			if changed == 0 {
				return fmt.Errorf("at least one setting for section %s must be provided", opts.section)
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			var request any
			switch opts.section {
			case "permission":
				request = opts.permission
			case "code-review":
				request = opts.codeReview
			case "pull-request":
				request = opts.pullRequest
			}
			if !opts.yes {
				fmt.Fprintf(cmd.ErrOrStderr(), "Repository: %s\nSection: %s\nChanged fields:\n", repository, opts.section)
				if err := printPolicyFields(cmd.ErrOrStderr(), request); err != nil {
					return err
				}
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), "Apply these policy changes? [y/N] ")
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.ErrOrStderr(), "Policy update cancelled.")
					return nil
				}
			}
			client, err := policyAPIClient(f)
			if err != nil {
				return err
			}
			switch opts.section {
			case "permission":
				err = api.UpdatePermissionPolicy(client, repository.Owner, repository.Name, opts.permission)
			case "code-review":
				err = api.UpdateCodeReviewPolicy(client, repository.Owner, repository.Name, opts.codeReview)
			case "pull-request":
				err = api.UpdatePullRequestPolicy(client, repository.Owner, repository.Name, opts.pullRequest)
			}
			if err != nil {
				return fmt.Errorf("failed to edit %s policy for %s: %w", opts.section, repository, err)
			}
			if opts.json {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), policyEditJSON{repository.String(), opts.section, request})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated %s policy for %s\nChanged fields:\n", opts.section, repository)
			return printPolicyFields(cmd.OutOrStdout(), request)
		}}
	cmd.Flags().StringVar(&opts.section, "section", "", "Section: permission, code-review, or pull-request (required)")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip update confirmation")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output the submitted fields as JSON")
	_ = cmd.MarkFlagRequired("section")
	addPolicyInt(cmd, &bindings, "permission", "mode", &opts.permission.Mode, "Permission mode: 1 (inherited), 2 (independent)")
	addPolicyString(cmd, &bindings, "code-review", "assignees", &opts.codeReview.Assignees, "Comma-separated approver usernames (empty clears)")
	addPolicyString(cmd, &bindings, "code-review", "testers", &opts.codeReview.Testers, "Comma-separated tester usernames (empty clears)")
	addPolicyInt(cmd, &bindings, "code-review", "assignees-number", &opts.codeReview.AssigneesNumber, "Minimum approver count (0 disables)")
	addPolicyInt(cmd, &bindings, "code-review", "testers-number", &opts.codeReview.TestersNumber, "Minimum tester count (0 disables)")
	addPolicyBool(cmd, &bindings, "pull-request", "approval-required-reviewers-enable", &opts.pullRequest.ApprovalRequiredReviewersEnable, "Enable the minimum reviewer gate (pull-request section)")
	addPolicyInt(cmd, &bindings, "pull-request", "approval-required-reviewers", &opts.pullRequest.ApprovalRequiredReviewers, "Minimum reviewer count: 0 disables, otherwise 1..5 (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "only-allow-merge-if-all-discussions-are-resolved", &opts.pullRequest.OnlyAllowMergeIfAllDiscussionsAreResolved, "Require all review discussions to be resolved (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "only-allow-merge-if-pipeline-succeeds", &opts.pullRequest.OnlyAllowMergeIfPipelineSucceeds, "Require a successful pipeline before merging (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "disable-merge-by-self", &opts.pullRequest.DisableMergeBySelf, "Prevent authors from merging their own pull requests (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "can-force-merge", &opts.pullRequest.CanForceMerge, "Allow administrators to force merge (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "add-notes-after-merged", &opts.pullRequest.AddNotesAfterMerged, "Allow review and comments after merging (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "mark-auto-merged-mr-as-closed", &opts.pullRequest.MarkAutoMergedMRAsClosed, "Mark automatically merged pull requests as closed (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "can-reopen", &opts.pullRequest.CanReopen, "Allow reopening closed pull requests (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "delete-source-branch-when-merged", &opts.pullRequest.DeleteSourceBranchWhenMerged, "Delete the source branch by default after merging (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "disable-squash-merge", &opts.pullRequest.DisableSquashMerge, "Disable squash merging (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "auto-squash-merge", &opts.pullRequest.AutoSquashMerge, "Enable squash by default for new pull requests (pull-request section)")
	addPolicyString(cmd, &bindings, "pull-request", "merge-method", &opts.pullRequest.MergeMethod, "Merge method: merge, rebase_merge, or ff (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "squash-merge-with-no-merge-commit", &opts.pullRequest.SquashMergeWithNoMergeCommit, "Do not create a merge commit for squash merges (pull-request section)")
	addPolicyString(cmd, &bindings, "pull-request", "merged-commit-author", &opts.pullRequest.MergedCommitAuthor, "Merge commit author: merged_by or created_by (pull-request section)")
	addPolicyInt(cmd, &bindings, "pull-request", "approval-required-approvers", &opts.pullRequest.ApprovalRequiredApprovers, "Minimum approver count (nonnegative) (pull-request section)")
	addPolicyString(cmd, &bindings, "pull-request", "approval-approver-ids", &opts.pullRequest.ApprovalApproverIDs, "Comma-separated approver user IDs (empty clears) (pull-request section)")
	addPolicyString(cmd, &bindings, "pull-request", "approval-tester-ids", &opts.pullRequest.ApprovalTesterIDs, "Comma-separated tester user IDs (empty clears) (pull-request section)")
	addPolicyInt(cmd, &bindings, "pull-request", "approval-required-testers", &opts.pullRequest.ApprovalRequiredTesters, "Minimum tester count (nonnegative) (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "is-check-cla", &opts.pullRequest.IsCheckCLA, "Require CLA validation (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "is-allow-lite-merge-request", &opts.pullRequest.IsAllowLiteMergeRequest, "Enable lightweight pull requests (pull-request section)")
	addPolicyString(cmd, &bindings, "pull-request", "lite-merge-request-prefix-title", &opts.pullRequest.LiteMergeRequestPrefixTitle, "Lightweight pull request title prefix (empty clears) (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "close-issue-when-mr-merged", &opts.pullRequest.CloseIssueWhenMRMerged, "Select closing linked issues by default (pull-request section)")
	addPolicyBool(cmd, &bindings, "pull-request", "forbidden-pr-related-issue-closed", &opts.pullRequest.ForbiddenPRRelatedIssueClosed, "Disable the option to close linked issues after merging (pull-request section)")
	return cmd
}
func addPolicyBool(cmd *cobra.Command, bindings *[]policyFlag, section, name string, target **bool, help string) {
	value := cmd.Flags().Bool(name, false, help)
	*bindings = append(*bindings, policyFlag{name, section, func() { *target = value }})
}
func addPolicyInt(cmd *cobra.Command, bindings *[]policyFlag, section, name string, target **int, help string) {
	value := cmd.Flags().Int(name, 0, help)
	*bindings = append(*bindings, policyFlag{name, section, func() { *target = value }})
}
func addPolicyString(cmd *cobra.Command, bindings *[]policyFlag, section, name string, target **string, help string) {
	value := cmd.Flags().String(name, "", help)
	*bindings = append(*bindings, policyFlag{name, section, func() { *target = value }})
}
func validatePolicyFlag(cmd *cobra.Command, name string) error {
	flag := cmd.Flags().Lookup(name)
	if flag.Value.Type() == "int" {
		value, err := cmd.Flags().GetInt(name)
		if err != nil {
			return err
		}
		if name == "mode" {
			if value != 1 && value != 2 {
				return fmt.Errorf("--mode must be 1 or 2")
			}
		} else if value < 0 || (name == "approval-required-reviewers" && value > 5) {
			return fmt.Errorf("invalid --%s count", name)
		}
	}
	value := flag.Value.String()
	switch name {
	case "merge-method":
		if value != "merge" && value != "rebase_merge" && value != "ff" {
			return fmt.Errorf("--merge-method must be merge, rebase_merge, or ff")
		}
	case "merged-commit-author":
		if value != "merged_by" && value != "created_by" {
			return fmt.Errorf("--merged-commit-author must be merged_by or created_by")
		}
	case "assignees", "testers", "approval-approver-ids", "approval-tester-ids":
		if value != "" {
			for part := range strings.SplitSeq(value, ",") {
				if strings.TrimSpace(part) == "" || strings.ContainsAny(part, " \t\r\n") {
					return fmt.Errorf("--%s must be a comma-separated list without whitespace or empty entries", name)
				}
			}
		}
	}
	return nil
}
func printPolicyFields(out io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(out, "  %s: %s\n", key, fields[key])
	}
	return nil
}
