package issue

import (
	"fmt"
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
				return fmt.Errorf("not authenticated: %w", err)
			}
			client, err := newAPIClient(f, token)
			if err != nil {
				return err
			}

			issuePath := fmt.Sprintf("/repos/%s/%s/issues/%s", repository.Owner, repository.Name, number)
			var current api.Issue
			if err := client.Get(issuePath, &current); err != nil {
				return fmt.Errorf("failed to get issue labels: %w", err)
			}

			currentLabels := issueLabelNames(current.Labels)
			updatedLabels, changedLabels, err := updateIssueLabels(currentLabels, labels, operation)
			if err != nil {
				return err
			}
			if len(changedLabels) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Issue #%s already has labels: %s\n", number, strings.Join(labels, ", "))
				return nil
			}

			labelsPath := issuePath + "/labels"
			if err := client.Put(labelsPath, updatedLabels, nil); err != nil {
				return fmt.Errorf("failed to %s labels on issue: %w", operation, err)
			}

			verb := "Added"
			preposition := "to"
			if operation == issueLabelRemove {
				verb = "Removed"
				preposition = "from"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s labels %s issue #%s: %s\n", verb, preposition, number, strings.Join(changedLabels, ", "))
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

func issueLabelNames(labels []api.Label) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		if name := strings.TrimSpace(label.Name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func updateIssueLabels(current, requested []string, operation issueLabelOperation) ([]string, []string, error) {
	present := make(map[string]struct{}, len(current))
	for _, label := range current {
		present[label] = struct{}{}
	}

	switch operation {
	case issueLabelAdd:
		updated := append([]string(nil), current...)
		changed := make([]string, 0, len(requested))
		for _, label := range requested {
			if _, exists := present[label]; exists {
				continue
			}
			present[label] = struct{}{}
			updated = append(updated, label)
			changed = append(changed, label)
		}
		return updated, changed, nil

	case issueLabelRemove:
		missing := make([]string, 0)
		remove := make(map[string]struct{}, len(requested))
		for _, label := range requested {
			if _, exists := present[label]; !exists {
				missing = append(missing, label)
				continue
			}
			remove[label] = struct{}{}
		}
		if len(missing) > 0 {
			return nil, nil, fmt.Errorf("labels not found on issue: %s", strings.Join(missing, ", "))
		}

		updated := make([]string, 0, len(current)-len(remove))
		for _, label := range current {
			if _, removeLabel := remove[label]; !removeLabel {
				updated = append(updated, label)
			}
		}
		return updated, append([]string(nil), requested...), nil

	default:
		return nil, nil, fmt.Errorf("unsupported issue label operation: %s", operation)
	}
}
