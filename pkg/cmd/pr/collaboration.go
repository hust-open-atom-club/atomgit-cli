package pr

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"github.com/spf13/cobra"
)

type prCreateMetadataOptions struct {
	Assignees []string
	Reviewers []string
	Testers   []string
	Labels    []string
	Milestone string
}

type prEditMetadataOptions struct {
	AddAssignees    []string
	RemoveAssignees []string
	AddReviewers    []string
	RemoveReviewers []string
	AddTesters      []string
	RemoveTesters   []string
	AddLabels       []string
	RemoveLabels    []string
	Milestone       string
}

type resolvedPRCreateMetadata struct {
	Assignees      []string
	Reviewers      []string
	Testers        []string
	Labels         []string
	Milestone      int
	MilestoneIsSet bool
}

type resolvedPREditMetadata struct {
	CurrentAssignees []string
	AddAssignees     []string
	RemoveAssignees  []string
	AddReviewers     []string
	RemoveReviewers  []string
	AddTesters       []string
	RemoveTesters    []string
	AddLabels        []string
	RemoveLabels     []string
	Milestone        int
	MilestoneIsSet   bool
}

func (opts prEditMetadataOptions) requested(cmd *cobra.Command) bool {
	return len(opts.AddAssignees)+len(opts.RemoveAssignees)+len(opts.AddReviewers)+len(opts.RemoveReviewers)+
		len(opts.AddTesters)+len(opts.RemoveTesters)+len(opts.AddLabels)+len(opts.RemoveLabels) > 0 ||
		cmd.Flags().Changed("milestone")
}

func resolvePRCreateMetadata(client *api.Client, owner, repo string, opts prCreateMetadataOptions) (resolvedPRCreateMetadata, error) {
	assignees, err := resolvePRUsers(client, opts.Assignees, "assignee")
	if err != nil {
		return resolvedPRCreateMetadata{}, err
	}
	reviewers, err := resolvePRUsers(client, opts.Reviewers, "reviewer")
	if err != nil {
		return resolvedPRCreateMetadata{}, err
	}
	testers, err := resolvePRUsers(client, opts.Testers, "tester")
	if err != nil {
		return resolvedPRCreateMetadata{}, err
	}
	labels, err := resolvePRLabels(client, owner, repo, opts.Labels)
	if err != nil {
		return resolvedPRCreateMetadata{}, err
	}

	result := resolvedPRCreateMetadata{Assignees: assignees, Reviewers: reviewers, Testers: testers, Labels: labels}
	if strings.TrimSpace(opts.Milestone) != "" {
		result.Milestone, err = resolvePRMilestone(client, owner, repo, opts.Milestone)
		if err != nil {
			return resolvedPRCreateMetadata{}, err
		}
		result.MilestoneIsSet = true
	}
	return result, nil
}

func (metadata resolvedPRCreateMetadata) addToCreateBody(body map[string]interface{}) {
	if len(metadata.Assignees) > 0 {
		body["assignees"] = strings.Join(metadata.Assignees, ",")
	}
	if len(metadata.Testers) > 0 {
		body["testers"] = strings.Join(metadata.Testers, ",")
	}
	if len(metadata.Labels) > 0 {
		body["labels"] = strings.Join(metadata.Labels, ",")
	}
	if metadata.MilestoneIsSet {
		body["milestone_number"] = metadata.Milestone
	}
}

func applyPRCreateMetadata(client *api.Client, owner, repo, number string, metadata resolvedPRCreateMetadata) error {
	if len(metadata.Reviewers) == 0 {
		return nil
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%s/reviewers", owner, repo, number)
	return client.Post(path, map[string]interface{}{"reviewers": strings.Join(metadata.Reviewers, ","), "add": true}, nil)
}

func resolvePREditMetadata(client *api.Client, owner, repo, number string, opts prEditMetadataOptions, cmd *cobra.Command) (resolvedPREditMetadata, error) {
	if !opts.requested(cmd) {
		return resolvedPREditMetadata{}, nil
	}

	pairs := []struct {
		add, remove []string
		role        string
	}{
		{opts.AddAssignees, opts.RemoveAssignees, "assignee"},
		{opts.AddReviewers, opts.RemoveReviewers, "reviewer"},
		{opts.AddTesters, opts.RemoveTesters, "tester"},
	}
	resolved := make([][2][]string, len(pairs))
	for i, pair := range pairs {
		var err error
		resolved[i][0], err = resolvePRUsers(client, pair.add, pair.role)
		if err != nil {
			return resolvedPREditMetadata{}, err
		}
		resolved[i][1], err = resolvePRUsers(client, pair.remove, pair.role)
		if err != nil {
			return resolvedPREditMetadata{}, err
		}
		if overlap := firstOverlap(resolved[i][0], resolved[i][1]); overlap != "" {
			return resolvedPREditMetadata{}, fmt.Errorf("%s %q cannot be both added and removed", pair.role, overlap)
		}
	}

	addLabels, err := resolvePRLabels(client, owner, repo, opts.AddLabels)
	if err != nil {
		return resolvedPREditMetadata{}, err
	}
	removeLabels, err := resolvePRLabels(client, owner, repo, opts.RemoveLabels)
	if err != nil {
		return resolvedPREditMetadata{}, err
	}
	if overlap := firstOverlap(addLabels, removeLabels); overlap != "" {
		return resolvedPREditMetadata{}, fmt.Errorf("label %q cannot be both added and removed", overlap)
	}

	var current api.PullRequest
	prPath := fmt.Sprintf("/repos/%s/%s/pulls/%s", owner, repo, number)
	if err := client.Get(prPath, &current); err != nil {
		return resolvedPREditMetadata{}, fmt.Errorf("resolve current PR collaboration metadata: %w", err)
	}
	currentLabels := current.Labels
	if len(opts.AddLabels)+len(opts.RemoveLabels) > 0 {
		labelsPath := fmt.Sprintf("/repos/%s/%s/pulls/%s/labels", owner, repo, number)
		if err := client.Get(labelsPath, &currentLabels); err != nil {
			return resolvedPREditMetadata{}, fmt.Errorf("resolve current PR labels: %w", err)
		}
	}

	result := resolvedPREditMetadata{
		CurrentAssignees: userLogins(current.Assignees),
		AddAssignees:     missingValues(resolved[0][0], userLogins(current.Assignees)),
		RemoveAssignees:  existingValues(resolved[0][1], userLogins(current.Assignees)),
		AddReviewers:     missingValues(resolved[1][0], userLogins(current.ApprovalReviewers)),
		RemoveReviewers:  existingValues(resolved[1][1], userLogins(current.ApprovalReviewers)),
		AddTesters:       missingValues(resolved[2][0], userLogins(current.Testers)),
		RemoveTesters:    existingValues(resolved[2][1], userLogins(current.Testers)),
		AddLabels:        missingValues(addLabels, labelNames(currentLabels)),
		RemoveLabels:     existingValues(removeLabels, labelNames(currentLabels)),
	}
	if cmd.Flags().Changed("milestone") {
		result.MilestoneIsSet = true
		if !strings.EqualFold(strings.TrimSpace(opts.Milestone), "none") {
			result.Milestone, err = resolvePRMilestone(client, owner, repo, opts.Milestone)
			if err != nil {
				return resolvedPREditMetadata{}, err
			}
		}
		if (current.Milestone == nil && result.Milestone == 0) || (current.Milestone != nil && current.Milestone.Number == result.Milestone) {
			result.MilestoneIsSet = false
		}
	}
	return result, nil
}

func applyPREditMetadata(client *api.Client, owner, repo, number string, metadata resolvedPREditMetadata) error {
	base := fmt.Sprintf("/repos/%s/%s/pulls/%s", owner, repo, number)
	if metadata.MilestoneIsSet {
		if err := client.Patch(base, map[string]int{"milestone_number": metadata.Milestone}, nil); err != nil {
			return fmt.Errorf("set milestone: %w", err)
		}
	}
	if err := applyAssigneeChanges(client, base, metadata); err != nil {
		return err
	}
	if err := applyRoleChanges(client, base+"/reviewers", "reviewers", metadata.AddReviewers, metadata.RemoveReviewers); err != nil {
		return fmt.Errorf("update approval reviewers: %w", err)
	}
	if err := applyRoleChanges(client, base+"/testers", "testers", metadata.AddTesters, metadata.RemoveTesters); err != nil {
		return fmt.Errorf("update testers: %w", err)
	}
	for _, label := range metadata.RemoveLabels {
		if err := client.Delete(base + "/labels/" + url.PathEscape(label)); err != nil {
			return fmt.Errorf("remove label %q: %w", label, err)
		}
	}
	if len(metadata.AddLabels) > 0 {
		if err := client.Post(base+"/labels", metadata.AddLabels, nil); err != nil {
			return fmt.Errorf("add labels: %w", err)
		}
	}
	return nil
}

func applyAssigneeChanges(client *api.Client, base string, metadata resolvedPREditMetadata) error {
	path := base + "/assignees"
	if len(metadata.RemoveAssignees) > 0 {
		query := url.Values{"assignees": {strings.Join(metadata.RemoveAssignees, ",")}}
		if err := client.Delete(path + "?" + query.Encode()); err != nil {
			return fmt.Errorf("remove assignees: %w", err)
		}
	}
	if len(metadata.AddAssignees) > 0 {
		if err := client.Post(path, map[string]string{"assignees": strings.Join(metadata.AddAssignees, ",")}, nil); err != nil {
			return fmt.Errorf("add assignees: %w", err)
		}
	}
	return nil
}

func applyRoleChanges(client *api.Client, path, field string, add, remove []string) error {
	if len(remove) > 0 {
		if err := client.DeleteWithBody(path, map[string]string{field: strings.Join(remove, ",")}); err != nil {
			return err
		}
	}
	if len(add) > 0 {
		if err := client.Post(path, map[string]interface{}{field: strings.Join(add, ","), "add": true}, nil); err != nil {
			return err
		}
	}
	return nil
}

func resolvePRUsers(client *api.Client, values []string, role string) ([]string, error) {
	values, err := normalizeUnique(values, role)
	if err != nil {
		return nil, err
	}
	resolved := make([]string, 0, len(values))
	for _, value := range values {
		var user api.User
		if err := client.Get("/users/"+url.PathEscape(value), &user); err != nil {
			return nil, fmt.Errorf("resolve %s %q: %w", role, value, err)
		}
		login := strings.TrimSpace(user.Login)
		if login == "" {
			return nil, fmt.Errorf("resolve %s %q: API response did not include a login", role, value)
		}
		resolved = appendUnique(resolved, login)
	}
	return resolved, nil
}

func resolvePRLabels(client *api.Client, owner, repo string, values []string) ([]string, error) {
	values, err := normalizeUnique(values, "label")
	if err != nil || len(values) == 0 {
		return values, err
	}
	labels, err := api.GetPaginated[api.Label](client, 1000, func(page, perPage int) string {
		return fmt.Sprintf("/repos/%s/%s/labels?page=%d&per_page=%d", owner, repo, page, perPage)
	})
	if err != nil {
		return nil, fmt.Errorf("resolve labels: %w", err)
	}
	return resolveNamedValues(values, labelNames(labels), "label")
}

func resolvePRMilestone(client *api.Client, owner, repo, value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("milestone cannot be empty")
	}
	milestones, err := api.GetPaginated[api.Milestone](client, 1000, func(page, perPage int) string {
		return fmt.Sprintf("/repos/%s/%s/milestones?state=all&page=%d&per_page=%d", owner, repo, page, perPage)
	})
	if err != nil {
		return 0, fmt.Errorf("resolve milestone %q: %w", value, err)
	}
	if number, parseErr := strconv.Atoi(value); parseErr == nil && number > 0 {
		for _, milestone := range milestones {
			if milestone.Number == number {
				return number, nil
			}
		}
		return 0, fmt.Errorf("milestone number %d does not exist", number)
	}
	matches := make([]api.Milestone, 0, 1)
	for _, milestone := range milestones {
		if strings.EqualFold(strings.TrimSpace(milestone.Title), value) {
			matches = append(matches, milestone)
		}
	}
	if len(matches) == 0 {
		return 0, fmt.Errorf("milestone %q does not exist", value)
	}
	if len(matches) > 1 {
		return 0, fmt.Errorf("milestone title %q is ambiguous; use its number", value)
	}
	return matches[0].Number, nil
}

func normalizeUnique(values []string, kind string) ([]string, error) {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%s cannot be empty", kind)
		}
		result = appendUnique(result, value)
	}
	return result, nil
}

func resolveNamedValues(requested, available []string, kind string) ([]string, error) {
	result := make([]string, 0, len(requested))
	for _, value := range requested {
		matches := make([]string, 0, 1)
		for _, candidate := range available {
			if strings.EqualFold(value, candidate) {
				matches = append(matches, candidate)
			}
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("%s %q does not exist", kind, value)
		}
		if len(matches) > 1 {
			sort.Strings(matches)
			return nil, fmt.Errorf("%s %q is ambiguous: %s", kind, value, strings.Join(matches, ", "))
		}
		result = appendUnique(result, matches[0])
	}
	return result, nil
}

func userLogins(users []api.User) []string {
	values := make([]string, 0, len(users))
	for _, user := range users {
		if login := strings.TrimSpace(user.Login); login != "" {
			values = appendUnique(values, login)
		}
	}
	return values
}

func labelNames(labels []api.Label) []string {
	values := make([]string, 0, len(labels))
	for _, label := range labels {
		if name := strings.TrimSpace(label.Name); name != "" {
			values = appendUnique(values, name)
		}
	}
	return values
}

func appendUnique(values []string, additions ...string) []string {
	for _, addition := range additions {
		if !containsFold(values, addition) {
			values = append(values, addition)
		}
	}
	return values
}

func containsFold(values []string, value string) bool {
	for _, candidate := range values {
		if strings.EqualFold(candidate, value) {
			return true
		}
	}
	return false
}

func firstOverlap(left, right []string) string {
	for _, value := range left {
		if containsFold(right, value) {
			return value
		}
	}
	return ""
}

func missingValues(requested, current []string) []string {
	result := make([]string, 0, len(requested))
	for _, value := range requested {
		if !containsFold(current, value) {
			result = append(result, value)
		}
	}
	return result
}

func existingValues(requested, current []string) []string {
	result := make([]string, 0, len(requested))
	for _, value := range requested {
		if containsFold(current, value) {
			result = append(result, value)
		}
	}
	return result
}

func removeValues(values, removals []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !containsFold(removals, value) {
			result = append(result, value)
		}
	}
	return result
}
