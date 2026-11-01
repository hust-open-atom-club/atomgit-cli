package org

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const (
	defaultJobHistoryLimit = 30
	maxJobHistoryPerPage   = 100
	jobHistoryTimeLayout   = "2006-01-02 15:04:05"
)

type jobHistoryListOptions struct {
	StartTimeBegin string
	StartTimeEnd   string
	Limit          int
	JSON           bool
}

type jobHistoryOutput struct {
	// TotalCount is the number of entries returned after applying --limit, not
	// the total number of entries reported by the API.
	TotalCount int                  `json:"total_count"`
	JobHistory []actions.JobHistory `json:"job_history"`
}

func newCmdOrgRunnerJobHistory(f *cmdutil.Factory) *cobra.Command {
	opts := &jobHistoryListOptions{Limit: defaultJobHistoryLimit}
	cmd := &cobra.Command{
		Use:   "runner-job-history <org>",
		Short: "List organization runner job execution history",
		Long:  "List the read-only execution history of organization-level AtomGit Actions runner resources. This command does not cancel or retry jobs. Time filters use the API's YYYY-MM-DD HH:mm:ss format and are sent without timezone conversion; the API documentation does not define a timezone. The duration field is returned as the API's raw integer because its unit is not documented.",
		Example: `  ag-cli org runner-job-history my-organization
  ag-cli org runner-job-history my-organization --limit 100
  ag-cli org runner-job-history my-organization --start-time-begin "2026-10-01 00:00:00" --start-time-end "2026-10-31 23:59:59" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			organization, err := parseRunnerGroupOrganization(args[0])
			if err != nil {
				return err
			}
			if err := validateJobHistoryOptions(opts); err != nil {
				return err
			}
			client, err := organizationActionsClient(f)
			if err != nil {
				return err
			}
			items, err := collectJobHistoryPages(opts.Limit, func(page, perPage int) (actions.JobHistoryListResponse, error) {
				return client.ListOrganizationRunnerJobHistory(organization, actions.ListJobHistoryOptions{
					Page:           page,
					PerPage:        perPage,
					StartTimeBegin: opts.StartTimeBegin,
					StartTimeEnd:   opts.StartTimeEnd,
				})
			})
			if err != nil {
				return fmt.Errorf("failed to list runner job history for organization %q: %w", organization, err)
			}
			if opts.JSON {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), jobHistoryOutput{TotalCount: len(items), JobHistory: items})
			}
			if len(items) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No runner job history found for organization %q.\n", organization)
				return nil
			}
			return writeJobHistory(cmd, items)
		},
	}
	cmd.Flags().StringVar(&opts.StartTimeBegin, "start-time-begin", "", "Filter by job start time, from (YYYY-MM-DD HH:mm:ss)")
	cmd.Flags().StringVar(&opts.StartTimeEnd, "start-time-end", "", "Filter by job start time, to (YYYY-MM-DD HH:mm:ss)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultJobHistoryLimit, "Maximum number of job history entries to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output runner job history as JSON")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals: &commandschema.Positionals{MinCount: 1, MaxCount: 1, Description: "Organization path. Required; repository context is not used."},
		Output:      "text", Effects: "none",
		Notes: []string{
			"Read-only; does not cancel or retry jobs. --json returns total_count for the entries returned after --limit and a job_history array using the documented snake_case fields.",
			"Time filters use the API's YYYY-MM-DD HH:mm:ss format and are sent without timezone conversion; the API documentation does not define a timezone.",
			"The API duration field is emitted as a raw integer because its unit is not documented. Missing start or complete times are displayed as '-' in text output.",
		},
	})
	return cmd
}

func validateJobHistoryOptions(opts *jobHistoryListOptions) error {
	if opts.Limit <= 0 {
		return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
	}
	beginValue, begin, beginSet, err := normalizeJobHistoryTime("--start-time-begin", opts.StartTimeBegin)
	if err != nil {
		return err
	}
	endValue, end, endSet, err := normalizeJobHistoryTime("--start-time-end", opts.StartTimeEnd)
	if err != nil {
		return err
	}
	if beginSet && endSet && begin.After(end) {
		return fmt.Errorf("--start-time-begin must not be after --start-time-end")
	}
	opts.StartTimeBegin = beginValue
	opts.StartTimeEnd = endValue
	return nil
}

func normalizeJobHistoryTime(flag, value string) (string, time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", time.Time{}, false, nil
	}
	parsed, err := time.Parse(jobHistoryTimeLayout, value)
	if err != nil || parsed.Format(jobHistoryTimeLayout) != value {
		return "", time.Time{}, false, fmt.Errorf("invalid %s: %q (must be YYYY-MM-DD HH:mm:ss)", flag, value)
	}
	return value, parsed, true, nil
}

func collectJobHistoryPages(limit int, fetch func(page, perPage int) (actions.JobHistoryListResponse, error)) ([]actions.JobHistory, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid limit: %d (must be positive)", limit)
	}
	// Page numbers are relative to per_page. Keep this constant across requests
	// so each page continues at the same offset; truncate to limit only after
	// receiving the page.
	perPage := min(maxJobHistoryPerPage, limit)
	items := make([]actions.JobHistory, 0, perPage)
	seenItems := make(map[string]struct{})
	seenPages := make(map[string]struct{})
	expectedTotal := -1
	expectedPageSize, expectedPageCount := 0, 0
	for page := 1; ; page++ {
		response, err := fetch(page, perPage)
		if err != nil {
			return nil, err
		}
		if response.Total < 0 || response.PageSize < 0 || response.PageCount < 0 {
			return nil, fmt.Errorf("inconsistent pagination: API returned negative metadata (total %d, page_size %d, page_count %d)", response.Total, response.PageSize, response.PageCount)
		}
		if response.PageNum != page {
			return nil, fmt.Errorf("inconsistent pagination: requested page %d but API returned page_num %d", page, response.PageNum)
		}
		if response.PageSize == 0 {
			return nil, fmt.Errorf("inconsistent pagination: API returned page_size 0")
		}
		if response.Total == 0 && response.PageCount != 0 {
			return nil, fmt.Errorf("inconsistent pagination: API reports total 0 but page_count %d", response.PageCount)
		}
		if expectedTotal < 0 {
			expectedTotal = response.Total
			expectedPageSize, expectedPageCount = response.PageSize, response.PageCount
		} else if response.Total != expectedTotal {
			return nil, fmt.Errorf("inconsistent pagination: API reported total %d after %d", response.Total, expectedTotal)
		}
		// Validate the page boundaries before consuming content or returning at
		// the limit. A changed page size can otherwise silently skip entries.
		if response.PageSize != expectedPageSize {
			return nil, fmt.Errorf("inconsistent pagination: API reported page_size %d after %d", response.PageSize, expectedPageSize)
		}
		if response.PageCount != expectedPageCount {
			return nil, fmt.Errorf("inconsistent pagination: API reported page_count %d after %d", response.PageCount, expectedPageCount)
		}
		if page == 1 && response.Total > 0 && response.PageSize > 0 && response.PageCount > 0 {
			wantPageCount := (response.Total + response.PageSize - 1) / response.PageSize
			if wantPageCount != response.PageCount {
				return nil, fmt.Errorf("inconsistent pagination: API reports %d items with page_size %d but page_count %d", response.Total, response.PageSize, response.PageCount)
			}
		}
		if response.PageSize > 0 && len(response.Content) > response.PageSize {
			return nil, fmt.Errorf("inconsistent pagination: page %d returned %d items with page_size %d", page, len(response.Content), response.PageSize)
		}
		if response.Total == 0 && len(response.Content) > 0 {
			return nil, fmt.Errorf("inconsistent pagination: page %d returned %d items but total is 0", page, len(response.Content))
		}
		if len(response.Content) == 0 {
			if expectedTotal > len(items) {
				return nil, fmt.Errorf("incomplete pagination: API reports %d items but only %d were returned", expectedTotal, len(items))
			}
			break
		}
		fingerprint, err := json.Marshal(response.Content)
		if err != nil {
			return nil, fmt.Errorf("fingerprint page: %w", err)
		}
		if _, exists := seenPages[string(fingerprint)]; exists {
			return nil, fmt.Errorf("pagination made no progress at page %d", page)
		}
		seenPages[string(fingerprint)] = struct{}{}
		newItems := 0
		for _, item := range response.Content {
			key := jobHistoryIdentity(item)
			if key == "" {
				return nil, fmt.Errorf("fingerprint item on page %d", page)
			}
			if _, exists := seenItems[key]; exists {
				continue
			}
			seenItems[key] = struct{}{}
			newItems++
			items = append(items, item)
			if expectedTotal >= 0 && len(items) > expectedTotal {
				return nil, fmt.Errorf("inconsistent pagination: API reports %d items but %d were returned", expectedTotal, len(items))
			}
			if len(items) == limit {
				return items, nil
			}
		}
		if newItems == 0 {
			return nil, fmt.Errorf("pagination made no progress at page %d", page)
		}
		if expectedTotal >= 0 && len(items) == expectedTotal {
			break
		}
		if response.PageCount > 0 && page >= response.PageCount {
			if expectedTotal > len(items) {
				return nil, fmt.Errorf("incomplete pagination: API reports %d items but only %d were returned", expectedTotal, len(items))
			}
			break
		}
		if response.PageSize > 0 {
			if len(response.Content) < response.PageSize && expectedTotal > len(items) {
				return nil, fmt.Errorf("incomplete pagination: API reports %d items but only %d were returned", expectedTotal, len(items))
			}
		} else if len(response.Content) < perPage && expectedTotal > len(items) {
			return nil, fmt.Errorf("incomplete pagination: API reports %d items but only %d were returned", expectedTotal, len(items))
		}
	}
	return items, nil
}

func jobHistoryIdentity(item actions.JobHistory) string {
	key := strings.Join([]string{
		item.JobRunID,
		item.JobID,
		item.JobExternalID,
		item.PipelineRunID,
		item.PipelineExternalID,
	}, "\x00")
	if strings.Trim(key, "\x00") != "" {
		return "job:" + key
	}
	fingerprint, err := json.Marshal(item)
	if err != nil {
		return ""
	}
	return "json:" + string(fingerprint)
}

func writeJobHistory(cmd *cobra.Command, items []actions.JobHistory) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "JOB RUN\tJOB ID\tJOB NAME\tSTATE\tRUNS ON\tPIPELINE\tPIPELINE RUN\tORG\tREPO\tSTART\tCOMPLETE\tDURATION")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\n",
			displayValue(item.JobRunID),
			displayValue(item.JobID),
			displayValue(item.JobName),
			displayValue(item.State),
			displayValue(item.RunsOn),
			displayValue(item.PipelineName),
			displayValue(item.PipelineRunID),
			displayValue(item.OrgNamespaceID),
			displayValue(item.RepoNamespaceID),
			displayValue(item.StartTime),
			displayValue(item.CompleteTime),
			item.Duration,
		)
	}
	return w.Flush()
}
