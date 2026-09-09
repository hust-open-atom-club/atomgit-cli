package runner

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type listOptions struct {
	JSON  bool
	Limit int
}

func newCmdList(f *cmdutil.Factory, shared bool) *cobra.Command {
	opts := &listOptions{}
	name := "list"
	short := "List host runners configured for a repository"
	source := "repository"
	if shared {
		name = "shared"
		short = "List host runners shared with a repository"
		source = "shared"
	}

	cmd := &cobra.Command{
		Use:     name + " [<owner>/<repo>]",
		Short:   short,
		Long:    short + ". This command is read-only and does not change runner configuration.",
		Example: "  ag runner " + name + " owner/repo\n  ag runner " + name + " owner/repo --limit 25 --json",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			if opts.Limit < 0 {
				return fmt.Errorf("invalid limit %d: must be zero or a positive integer", opts.Limit)
			}
			token, err := requireToken(f)
			if err != nil {
				return err
			}
			client, err := newActionsClient(f, token)
			if err != nil {
				return err
			}

			runners, err := listAllRunners(client, repository.Owner, repository.Name, shared, opts.Limit)
			if err != nil {
				return fmt.Errorf("failed to list %s runners for %s/%s: %w", source, repository.Owner, repository.Name, err)
			}
			result := actions.RunnerListResponse{TotalCount: len(runners), Runners: runners}
			if opts.JSON {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal json output: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}
			if len(runners) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No %s runners found.\n", source)
				return nil
			}
			return writeRunnerTable(cmd, source, runners)
		},
	}
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output runners as JSON")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 0, "Maximum number of runners to list (0 means all)")
	return cmd
}

func listAllRunners(client *actions.Client, owner, repo string, shared bool, limit int) ([]actions.Runner, error) {
	if limit < 0 {
		return nil, fmt.Errorf("invalid limit %d: must be zero or a positive integer", limit)
	}

	runners := make([]actions.Runner, 0, maxRunnersPerPage)
	seenIDs := make(map[string]struct{})
	seenPages := make(map[string]struct{})
	expectedTotal := 0
	for page := 1; ; page++ {
		var (
			response actions.RunnerListResponse
			err      error
		)
		options := actions.ListRunnersOptions{Page: page, PerPage: maxRunnersPerPage}
		if shared {
			response, err = client.ListSharedRunners(owner, repo, options)
		} else {
			response, err = client.ListRunners(owner, repo, options)
		}
		if err != nil {
			return nil, err
		}
		if response.TotalCount > 0 {
			if expectedTotal == 0 {
				expectedTotal = response.TotalCount
			} else if response.TotalCount != expectedTotal {
				return nil, fmt.Errorf("inconsistent pagination: API reported total_count %d after %d", response.TotalCount, expectedTotal)
			}
		} else if expectedTotal > 0 {
			return nil, fmt.Errorf("inconsistent pagination: API reported total_count 0 after %d", expectedTotal)
		}

		if len(response.Runners) == 0 {
			if expectedTotal > len(runners) {
				return nil, fmt.Errorf("incomplete pagination: API reports %d runners but only %d were returned", expectedTotal, len(runners))
			}
			break
		}
		pageKey, err := json.Marshal(response.Runners)
		if err != nil {
			return nil, fmt.Errorf("fingerprint runner page: %w", err)
		}
		if _, exists := seenPages[string(pageKey)]; exists {
			return nil, fmt.Errorf("pagination made no progress at page %d", page)
		}
		seenPages[string(pageKey)] = struct{}{}

		newItems := 0
		for _, runner := range response.Runners {
			key := runnerKey(runner)
			if _, exists := seenIDs[key]; exists {
				continue
			}
			seenIDs[key] = struct{}{}
			newItems++
			runners = append(runners, runner)
			if limit > 0 && len(runners) >= limit {
				return runners[:limit], nil
			}
		}
		if newItems == 0 {
			return nil, fmt.Errorf("pagination made no progress at page %d", page)
		}
		if expectedTotal > 0 && len(runners) >= expectedTotal {
			break
		}
		if len(response.Runners) < maxRunnersPerPage {
			if expectedTotal > len(runners) {
				return nil, fmt.Errorf("incomplete pagination: API reports %d runners but only %d were returned", expectedTotal, len(runners))
			}
			break
		}
	}
	return runners, nil
}

func runnerKey(runner actions.Runner) string {
	if id := strings.TrimSpace(string(runner.ID)); id != "" {
		return "id:" + id
	}
	return "fallback:" + strings.Join([]string{
		runner.Name,
		runner.OS,
		runner.Platform,
		runner.Scope,
	}, "\x00")
}

func writeRunnerTable(cmd *cobra.Command, source string, runners []actions.Runner) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "SOURCE\tSCOPE\tID\tNAME\tSTATUS\tBUSY\tONLINE\tPLATFORM\tOS\tLABELS")
	for _, runner := range runners {
		scope := strings.TrimSpace(runner.Scope)
		if scope == "" {
			scope = "-"
		}
		id := string(runner.ID)
		if id == "" {
			id = "-"
		}
		name := strings.TrimSpace(runner.Name)
		if name == "" {
			name = "-"
		}
		status := strings.TrimSpace(runner.Status)
		if status == "" {
			status = "-"
		}
		platform := strings.TrimSpace(runner.Platform)
		if platform == "" {
			platform = "-"
		}
		osName := strings.TrimSpace(runner.OS)
		if osName == "" {
			osName = "-"
		}
		labels := runnerLabels(runner.Labels)
		if labels == "" {
			labels = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			source, scope, id, name, status, optionalBool(runner.Busy), optionalBool(runner.Online), platform, osName, labels)
	}
	return w.Flush()
}

func optionalBool(value *bool) string {
	if value == nil {
		return "-"
	}
	if *value {
		return "true"
	}
	return "false"
}

func runnerLabels(labels []actions.RunnerLabel) string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		if name := strings.TrimSpace(label.Name); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ",")
}
