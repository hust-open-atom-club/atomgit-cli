package repo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

var collaboratorPermissionRank = map[string]int{
	"pull":  1,
	"push":  2,
	"admin": 3,
}

func newCmdRepoCollaborator(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collaborator",
		Short: "Manage repository collaborators",
		Long: `List, inspect, add, edit, and remove direct repository collaborators.

The built-in AtomGit permissions are pull, push, and admin. Organization-
inherited permissions are displayed but cannot be changed by these commands.`,
	}
	cmd.AddCommand(newCmdRepoCollaboratorList(f))
	cmd.AddCommand(newCmdRepoCollaboratorView(f))
	cmd.AddCommand(newCmdRepoCollaboratorAdd(f))
	cmd.AddCommand(newCmdRepoCollaboratorEdit(f))
	cmd.AddCommand(newCmdRepoCollaboratorRemove(f))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func newCmdRepoCollaboratorList(f *cmdutil.Factory) *cobra.Command {
	var limit int
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "list [<owner>/<repo>]",
		Short:   "List repository collaborators",
		Example: "  ag repo collaborator list owner/repo --limit 50",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", limit)
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := collaboratorAPIClient(f)
			if err != nil {
				return err
			}
			items, err := api.GetPaginated[api.Collaborator](client, limit, func(page, perPage int) string {
				return fmt.Sprintf("%s?page=%d&per_page=%d", collaboratorCollectionPath(repository), page, perPage)
			})
			if err != nil {
				return fmt.Errorf("failed to list collaborators: %w", err)
			}
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), collaboratorsJSON(items, repository))
			}
			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No collaborators found.")
				return nil
			}
			for _, item := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s [%s] %s", collaboratorUsername(item), collaboratorPermission(item), collaboratorAccess(item, repository))
				if item.RoleName != "" {
					fmt.Fprintf(cmd.OutOrStdout(), " (%s)", item.RoleName)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of collaborators to list")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output collaborators as JSON")
	return cmd
}

func newCmdRepoCollaboratorView(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "view [<owner>/<repo>] <username>",
		Short:   "View a repository collaborator's effective permission",
		Example: "  ag repo collaborator view owner/repo octocat",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			username, err := validateCollaboratorUsername(remaining[0])
			if err != nil {
				return err
			}
			client, err := collaboratorAPIClient(f)
			if err != nil {
				return err
			}
			item, found, err := getCollaborator(client, repository, username)
			if err != nil {
				return fmt.Errorf("failed to view collaborator %q: %w", username, err)
			}
			if !found {
				return fmt.Errorf("collaborator %q was not found in %s", username, repository.String())
			}
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), newCollaboratorJSON(item, repository))
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Username: %s\n", collaboratorUsername(item))
			fmt.Fprintf(out, "Permission: %s\n", collaboratorPermission(item))
			fmt.Fprintf(out, "Access: %s\n", collaboratorAccess(item, repository))
			if item.RoleName != "" {
				fmt.Fprintf(out, "Role: %s\n", item.RoleName)
			}
			if item.SourceName != "" {
				fmt.Fprintf(out, "Source: %s\n", item.SourceName)
			}
			if item.WebURL != "" {
				fmt.Fprintf(out, "URL: %s\n", item.WebURL)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output collaborator as JSON")
	return cmd
}

func newCmdRepoCollaboratorAdd(f *cmdutil.Factory) *cobra.Command {
	var permission string
	cmd := &cobra.Command{
		Use:     "add [<owner>/<repo>] <username>",
		Short:   "Add a direct repository collaborator",
		Example: "  ag repo collaborator add owner/repo octocat --permission push",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			permission, err := validateCollaboratorPermission(permission)
			if err != nil {
				return err
			}
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			username, err := validateCollaboratorUsername(remaining[0])
			if err != nil {
				return err
			}
			client, err := collaboratorAPIClient(f)
			if err != nil {
				return err
			}
			current, found, err := getCollaborator(client, repository, username)
			if err != nil {
				return fmt.Errorf("failed to check collaborator %q: %w", username, err)
			}
			if found {
				if directCollaborator(current, repository) {
					return fmt.Errorf("%q is already a direct collaborator; use `ag repo collaborator edit`", username)
				}
				return inheritedCollaboratorError(current, repository)
			}
			var added api.Collaborator
			if err := client.Put(collaboratorPath(repository, username), map[string]string{"permission": permission}, &added); err != nil {
				return fmt.Errorf("failed to add collaborator %q: %w", username, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added collaborator %s with %s permission\n", username, permission)
			return nil
		},
	}
	cmd.Flags().StringVarP(&permission, "permission", "p", "push", "Permission: pull, push, or admin")
	return cmd
}

func newCmdRepoCollaboratorEdit(f *cmdutil.Factory) *cobra.Command {
	var permission string
	var yes bool
	cmd := &cobra.Command{
		Use:     "edit [<owner>/<repo>] <username>",
		Short:   "Update a direct repository collaborator's permission",
		Example: "  ag repo collaborator edit owner/repo octocat --permission pull",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			permission, err := validateCollaboratorPermission(permission)
			if err != nil {
				return err
			}
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			username, err := validateCollaboratorUsername(remaining[0])
			if err != nil {
				return err
			}
			client, err := collaboratorAPIClient(f)
			if err != nil {
				return err
			}
			current, found, err := getCollaborator(client, repository, username)
			if err != nil {
				return fmt.Errorf("failed to check collaborator %q: %w", username, err)
			}
			if !found {
				return fmt.Errorf("collaborator %q was not found in %s", username, repository.String())
			}
			if err := ensureMutableDirectCollaborator(current, repository); err != nil {
				return err
			}
			currentPermission := collaboratorPermission(current)
			if currentPermission == permission {
				return fmt.Errorf("collaborator %q already has %s permission", username, permission)
			}
			if !yes && collaboratorPermissionReduction(currentPermission, permission) {
				confirmed, err := confirmCollaboratorAction(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Reduce %s's permission from %s to %s", username, currentPermission, permission))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Permission update cancelled.")
					return nil
				}
			}
			var updated api.Collaborator
			if err := client.Put(collaboratorPath(repository, username), map[string]string{"permission": permission}, &updated); err != nil {
				return fmt.Errorf("failed to update collaborator %q: %w", username, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated collaborator %s to %s permission\n", username, permission)
			return nil
		},
	}
	cmd.Flags().StringVarP(&permission, "permission", "p", "", "Permission: pull, push, or admin")
	_ = cmd.MarkFlagRequired("permission")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation for permission reductions")
	return cmd
}

func newCmdRepoCollaboratorRemove(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "remove [<owner>/<repo>] <username>",
		Short:   "Remove a direct repository collaborator",
		Example: "  ag repo collaborator remove owner/repo octocat --yes",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			username, err := validateCollaboratorUsername(remaining[0])
			if err != nil {
				return err
			}
			client, err := collaboratorAPIClient(f)
			if err != nil {
				return err
			}
			current, found, err := getCollaborator(client, repository, username)
			if err != nil {
				return fmt.Errorf("failed to check collaborator %q: %w", username, err)
			}
			if !found {
				return fmt.Errorf("collaborator %q was not found in %s", username, repository.String())
			}
			if err := ensureMutableDirectCollaborator(current, repository); err != nil {
				return err
			}
			if !yes {
				confirmed, err := confirmCollaboratorAction(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Remove %s from %s", username, repository.String()))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Removal cancelled.")
					return nil
				}
			}
			if err := client.Delete(collaboratorPath(repository, username)); err != nil {
				return fmt.Errorf("failed to remove collaborator %q: %w", username, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed collaborator %s\n", username)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func collaboratorAPIClient(f *cmdutil.Factory) (*api.Client, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, fmt.Errorf("not authenticated: %w", err)
	}
	return newAPIClient(f, token)
}

func collaboratorCollectionPath(repository cmdutil.Repository) string {
	return fmt.Sprintf("/repos/%s/%s/collaborators", repository.Owner, repository.Name)
}

func collaboratorPath(repository cmdutil.Repository, username string) string {
	return collaboratorCollectionPath(repository) + "/" + url.PathEscape(username)
}

func getCollaborator(client *api.Client, repository cmdutil.Repository, username string) (api.Collaborator, bool, error) {
	path := collaboratorPath(repository, username) + "/permission"
	resp, err := client.DoRequestRaw(http.MethodGet, path)
	if err != nil {
		return api.Collaborator{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return api.Collaborator{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return api.Collaborator{}, false, fmt.Errorf("API error: %s - %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var item api.Collaborator
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return api.Collaborator{}, false, fmt.Errorf("decode collaborator response: %w", err)
	}
	return item, true, nil
}

func validateCollaboratorUsername(value string) (string, error) {
	username := strings.TrimSpace(value)
	if username == "" || strings.ContainsAny(username, "/\\") || strings.ContainsAny(username, "\r\n\t ") {
		return "", fmt.Errorf("invalid collaborator username %q", value)
	}
	return username, nil
}

func validateCollaboratorPermission(value string) (string, error) {
	permission := strings.ToLower(strings.TrimSpace(value))
	if _, ok := collaboratorPermissionRank[permission]; !ok {
		return "", fmt.Errorf("invalid collaborator permission %q (expected pull, push, or admin)", value)
	}
	return permission, nil
}

func collaboratorUsername(item api.Collaborator) string {
	for _, value := range []string{item.Username, item.Login, item.Name} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "unknown"
}

func collaboratorPermission(item api.Collaborator) string {
	if permission := strings.ToLower(strings.TrimSpace(item.Permission)); permission != "" {
		return permission
	}
	switch {
	case bool(item.Permissions.Admin):
		return "admin"
	case bool(item.Permissions.Push):
		return "push"
	case bool(item.Permissions.Pull):
		return "pull"
	default:
		return "unknown"
	}
}

func directCollaborator(item api.Collaborator, repository cmdutil.Repository) bool {
	return strings.EqualFold(strings.TrimSpace(item.Type), "ProjectMember") &&
		strings.EqualFold(strings.TrimSpace(item.SourceName), repository.Name) &&
		(strings.TrimSpace(item.JoinWay) == "" || strings.EqualFold(strings.TrimSpace(item.JoinWay), "normal"))
}

func collaboratorAccess(item api.Collaborator, repository cmdutil.Repository) string {
	if directCollaborator(item, repository) {
		return "direct"
	}
	if source := strings.TrimSpace(item.SourceName); source != "" {
		return "inherited from " + source
	}
	return "inherited or unknown source"
}

func inheritedCollaboratorError(item api.Collaborator, repository cmdutil.Repository) error {
	return fmt.Errorf("collaborator %q has %s access; inherited permissions cannot be modified at repository level", collaboratorUsername(item), collaboratorAccess(item, repository))
}

func ensureMutableDirectCollaborator(item api.Collaborator, repository cmdutil.Repository) error {
	if !directCollaborator(item, repository) {
		return inheritedCollaboratorError(item, repository)
	}
	if strings.EqualFold(strings.TrimSpace(item.RoleName), "owner") {
		return fmt.Errorf("repository owner %q cannot be modified as a collaborator", collaboratorUsername(item))
	}
	return nil
}

func collaboratorPermissionReduction(current, target string) bool {
	currentRank, currentKnown := collaboratorPermissionRank[current]
	targetRank := collaboratorPermissionRank[target]
	return !currentKnown || targetRank < currentRank
}

func confirmCollaboratorAction(in io.Reader, out io.Writer, prompt string) (bool, error) {
	fmt.Fprintf(out, "%s? [y/N] ", prompt)
	var response string
	if _, err := fmt.Fscan(in, &response); err != nil && err != io.EOF {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	return strings.EqualFold(response, "y") || strings.EqualFold(response, "yes"), nil
}

type collaboratorJSON struct {
	Username   string `json:"username"`
	Name       string `json:"name"`
	Permission string `json:"permission"`
	Access     string `json:"access"`
	Direct     bool   `json:"direct"`
	Role       string `json:"role"`
	Source     string `json:"source"`
	Type       string `json:"type"`
	WebURL     string `json:"url"`
}

func collaboratorsJSON(items []api.Collaborator, repository cmdutil.Repository) []collaboratorJSON {
	result := make([]collaboratorJSON, len(items))
	for index, item := range items {
		result[index] = newCollaboratorJSON(item, repository)
	}
	return result
}

func newCollaboratorJSON(item api.Collaborator, repository cmdutil.Repository) collaboratorJSON {
	return collaboratorJSON{
		Username: collaboratorUsername(item), Name: item.Name, Permission: collaboratorPermission(item),
		Access: collaboratorAccess(item, repository), Direct: directCollaborator(item, repository),
		Role: item.RoleName, Source: item.SourceName, Type: item.Type, WebURL: item.WebURL,
	}
}
