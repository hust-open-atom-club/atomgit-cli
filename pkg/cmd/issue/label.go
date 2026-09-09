package issue

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type issueLabelOperation string

const (
	issueLabelAdd    issueLabelOperation = "add"
	issueLabelRemove issueLabelOperation = "remove"
)

func newCmdIssueLabel(f *cmdutil.Factory) *cobra.Command {
	var addLabels, removeLabels string

	cmd := &cobra.Command{
		Use:   "label [<owner>/<repo>] <number> [<labels>]",
		Short: "Add or remove labels on an issue",
		Long: `Add or remove labels on an issue.

Labels are comma-separated. Positional labels are treated as labels to add for
backward compatibility. Use --add or --remove to make the operation explicit.`,
		Example: `  ag issue label owner/repo 42 "bug, help wanted"
  ag issue label owner/repo 42 --add "bug, help wanted"
  ag issue label owner/repo 42 --remove "priority/high"`,
		Args: cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, number, labels, operation, err := resolveIssueLabelInput(f, cmd, args, addLabels, removeLabels)
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			labelsPath := fmt.Sprintf("/repos/%s/%s/issues/%s/labels", repository.Owner, repository.Name, number)
			switch operation {
			case issueLabelAdd:
				if err := client.Post(labelsPath, labels, nil); err != nil {
					return fmt.Errorf("failed to add labels to issue: %w", err)
				}
			case issueLabelRemove:
				if err := removeIssueLabels(client, labelsPath, labels); err != nil {
					return fmt.Errorf("failed to remove labels from issue: %w", err)
				}
			default:
				return fmt.Errorf("unsupported issue label operation: %s", operation)
			}

			verb := "Added"
			preposition := "to"
			if operation == issueLabelRemove {
				verb = "Removed"
				preposition = "from"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s labels %s issue #%s: %s\n", verb, preposition, number, strings.Join(labels, ", "))
			return nil
		},
	}

	cmd.Flags().StringVar(&addLabels, "add", "", "Comma-separated labels to add")
	cmd.Flags().StringVar(&removeLabels, "remove", "", "Comma-separated labels to remove")
	return cmd
}

func resolveIssueLabelInput(f *cmdutil.Factory, cmd *cobra.Command, args []string, addValue, removeValue string) (cmdutil.Repository, string, []string, issueLabelOperation, error) {
	addChanged := cmd.Flags().Changed("add")
	removeChanged := cmd.Flags().Changed("remove")
	if addChanged && removeChanged {
		return cmdutil.Repository{}, "", nil, "", fmt.Errorf("--add and --remove cannot be used together")
	}

	operation := issueLabelAdd
	labelValue := ""
	trailingArgs := 1
	if addChanged || removeChanged {
		if len(args) > 2 {
			return cmdutil.Repository{}, "", nil, "", fmt.Errorf("positional labels cannot be used with --add or --remove")
		}
		if addChanged {
			labelValue = addValue
		} else {
			operation = issueLabelRemove
			labelValue = removeValue
		}
	} else {
		trailingArgs = 2
		if len(args) == 1 || (len(args) == 2 && strings.Contains(args[0], "/")) {
			return cmdutil.Repository{}, "", nil, "", fmt.Errorf("labels are required; pass positional labels or use --add/--remove")
		}
		labelValue = args[len(args)-1]
	}

	repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, trailingArgs)
	if err != nil {
		return cmdutil.Repository{}, "", nil, "", err
	}
	labels, err := parseIssueLabels(labelValue)
	if err != nil {
		return cmdutil.Repository{}, "", nil, "", err
	}
	number := strings.TrimSpace(remaining[0])
	parsedNumber, err := strconv.Atoi(number)
	if err != nil || parsedNumber <= 0 {
		return cmdutil.Repository{}, "", nil, "", fmt.Errorf("invalid issue number %q (must be a positive integer)", number)
	}
	return repository, number, labels, operation, nil
}

func parseIssueLabels(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	labels := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		label := strings.TrimSpace(part)
		if label == "" {
			return nil, fmt.Errorf("label cannot be empty")
		}
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}
	return labels, nil
}

func removeIssueLabels(client *api.Client, labelsPath string, labels []string) error {
	removed := make([]string, 0, len(labels))
	for _, label := range labels {
		path := labelsPath + "/" + url.PathEscape(label)
		if err := client.Delete(path); err != nil {
			if len(removed) == 0 {
				return fmt.Errorf("failed to remove label %q: %w", label, err)
			}

			if rollbackErr := client.Post(labelsPath, removed, nil); rollbackErr != nil {
				return fmt.Errorf(
					"failed to remove label %q after removing %s; failed to restore previously removed labels: %v: %w",
					label, strings.Join(removed, ", "), rollbackErr, err,
				)
			}
			return fmt.Errorf(
				"failed to remove label %q; restored previously removed labels: %s: %w",
				label, strings.Join(removed, ", "), err,
			)
		}
		removed = append(removed, label)
	}
	return nil
}
