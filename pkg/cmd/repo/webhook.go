package repo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

var webhookEventFields = map[string]string{
	"push":           "push_events",
	"tag-push":       "tag_push_events",
	"issues":         "issues_events",
	"note":           "note_events",
	"merge-requests": "merge_requests_events",
}

type webhookSecretOptions struct {
	EnvName string
	File    string
	Stdin   bool
}

func newCmdRepoWebhook(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage repository webhooks",
		Long: `List, inspect, create, edit, delete, and test repository webhooks.

Webhook secrets are accepted only from an environment variable, a file, or
standard input. They are never included in command output. The API exposes the
active state as read-only metadata, so this command does not send an
undocumented active field.`,
	}
	cmd.AddCommand(newCmdRepoWebhookList(f))
	cmd.AddCommand(newCmdRepoWebhookView(f))
	cmd.AddCommand(newCmdRepoWebhookCreate(f))
	cmd.AddCommand(newCmdRepoWebhookEdit(f))
	cmd.AddCommand(newCmdRepoWebhookDelete(f))
	cmd.AddCommand(newCmdRepoWebhookTest(f))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func newCmdRepoWebhookList(f *cmdutil.Factory) *cobra.Command {
	var limit int
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "list [<owner>/<repo>]",
		Short:   "List repository webhooks",
		Example: "  ag repo webhook list owner/repo --limit 50",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", limit)
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := webhookAPIClient(f)
			if err != nil {
				return err
			}
			items, err := listWebhooks(client, repository, limit)
			if err != nil {
				return fmt.Errorf("failed to list webhooks: %w", err)
			}
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), webhooksJSON(items))
			}
			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No webhooks found.")
				return nil
			}
			for _, item := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "#%d %s [%s] events: %s\n", item.ID, item.URL, webhookActiveState(item), strings.Join(webhookEvents(item), ", "))
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of webhooks to list")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output webhooks as JSON")
	return cmd
}

func newCmdRepoWebhookView(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "view [<owner>/<repo>] <id>",
		Short:   "View a repository webhook",
		Example: "  ag repo webhook view owner/repo 42",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			id, err := parseWebhookID(remaining[0])
			if err != nil {
				return err
			}
			client, err := webhookAPIClient(f)
			if err != nil {
				return err
			}
			item, err := getWebhook(client, repository, id)
			if err != nil {
				return err
			}
			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), newWebhookJSON(item))
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Webhook: #%d\n", item.ID)
			fmt.Fprintf(out, "URL: %s\n", item.URL)
			fmt.Fprintf(out, "State: %s\n", webhookActiveState(item))
			fmt.Fprintf(out, "Events: %s\n", strings.Join(webhookEvents(item), ", "))
			if item.Result != "" || item.ResultCode != 0 {
				fmt.Fprintf(out, "Last result: %d %s\n", item.ResultCode, item.Result)
			}
			if item.CreatedAt != "" {
				fmt.Fprintf(out, "Created: %s\n", item.CreatedAt)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output webhook as JSON")
	return cmd
}

func newCmdRepoWebhookCreate(f *cmdutil.Factory) *cobra.Command {
	var targetURL, events, encryption string
	var secret webhookSecretOptions
	cmd := &cobra.Command{
		Use:     "create [<owner>/<repo>]",
		Short:   "Create a repository webhook",
		Example: "  ag repo webhook create owner/repo --url https://example.com/hook --events push,issues --secret-env WEBHOOK_SECRET",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL, err := validateWebhookURL(targetURL)
			if err != nil {
				return err
			}
			eventValues, err := parseWebhookEvents(events, false)
			if err != nil {
				return err
			}
			var secretValue string
			var secretProvided bool
			if !secret.Stdin {
				secretValue, secretProvided, err = readWebhookSecret(cmd.InOrStdin(), secret)
				if err != nil {
					return err
				}
				if webhookSecretFlagsChanged(cmd) && !secretProvided {
					return fmt.Errorf("webhook secret source must not be empty")
				}
			}
			encryptionValue, encryptionProvided, err := parseWebhookEncryption(encryption)
			if err != nil {
				return err
			}
			if encryptionProvided && !webhookSecretFlagsChanged(cmd) {
				return fmt.Errorf("--encryption requires a secret source")
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}
			if secret.Stdin {
				secretValue, secretProvided, err = readWebhookSecret(cmd.InOrStdin(), secret)
				if err != nil {
					return err
				}
				if !secretProvided {
					return fmt.Errorf("webhook secret source must not be empty")
				}
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}
			body := map[string]interface{}{"url": targetURL}
			applyWebhookEvents(body, eventValues)
			if secretProvided {
				body["password"] = secretValue
				if encryptionProvided {
					body["encryption_type"] = encryptionValue
				} else {
					body["encryption_type"] = 0
				}
			}
			var created api.Webhook
			if err := mutateWebhook(client, http.MethodPost, webhookCollectionPath(repository), body, &created); err != nil {
				return fmt.Errorf("failed to create webhook: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created webhook #%d for %s\n", created.ID, created.URL)
			return nil
		},
	}
	cmd.Flags().StringVar(&targetURL, "url", "", "Webhook target HTTP(S) URL")
	cmd.Flags().StringVar(&events, "events", "", "Comma-separated events: push, tag-push, issues, note, merge-requests")
	cmd.Flags().StringVar(&encryption, "encryption", "", "Secret mode: password or signature")
	addWebhookSecretFlags(cmd, &secret)
	_ = cmd.MarkFlagRequired("url")
	_ = cmd.MarkFlagRequired("events")
	return cmd
}

func newCmdRepoWebhookEdit(f *cmdutil.Factory) *cobra.Command {
	var targetURL, events, encryption string
	var secret webhookSecretOptions
	cmd := &cobra.Command{
		Use:     "edit [<owner>/<repo>] <id>",
		Short:   "Edit a repository webhook",
		Example: "  ag repo webhook edit owner/repo 42 --events push,merge-requests",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("url") && !cmd.Flags().Changed("events") && !cmd.Flags().Changed("encryption") && !webhookSecretFlagsChanged(cmd) {
				return fmt.Errorf("at least one webhook setting must be provided")
			}
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			id, err := parseWebhookID(remaining[0])
			if err != nil {
				return err
			}
			body := make(map[string]interface{})
			if cmd.Flags().Changed("url") {
				targetURL, err = validateWebhookURL(targetURL)
				if err != nil {
					return err
				}
				body["url"] = targetURL
			}
			if cmd.Flags().Changed("events") {
				eventValues, err := parseWebhookEvents(events, true)
				if err != nil {
					return err
				}
				applyWebhookEvents(body, eventValues)
			}
			var secretValue string
			var secretProvided bool
			if !secret.Stdin {
				secretValue, secretProvided, err = readWebhookSecret(cmd.InOrStdin(), secret)
				if err != nil {
					return err
				}
				if webhookSecretFlagsChanged(cmd) && !secretProvided {
					return fmt.Errorf("webhook secret source must not be empty")
				}
			}
			encryptionValue := 0
			if cmd.Flags().Changed("encryption") {
				encryptionValue, _, err = parseWebhookEncryption(encryption)
				if err != nil {
					return err
				}
			}
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}
			if secret.Stdin {
				secretValue, secretProvided, err = readWebhookSecret(cmd.InOrStdin(), secret)
				if err != nil {
					return err
				}
				if !secretProvided {
					return fmt.Errorf("webhook secret source must not be empty")
				}
			}
			if secretProvided {
				body["password"] = secretValue
			}
			if cmd.Flags().Changed("encryption") {
				body["encryption_type"] = encryptionValue
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}
			current, err := getWebhook(client, repository, id)
			if err != nil {
				return err
			}
			if _, ok := body["url"]; !ok {
				currentURL, err := validateWebhookURL(current.URL)
				if err != nil {
					return fmt.Errorf("webhook #%d has no valid API-required URL: %w", id, err)
				}
				body["url"] = currentURL
			}
			var updated api.Webhook
			if err := mutateWebhook(client, http.MethodPatch, webhookPath(repository, id), body, &updated); err != nil {
				return fmt.Errorf("failed to edit webhook #%d: %w", id, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated webhook #%d\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&targetURL, "url", "", "New webhook target HTTP(S) URL")
	cmd.Flags().StringVar(&events, "events", "", "Replace events; use none to disable all events")
	cmd.Flags().StringVar(&encryption, "encryption", "", "Secret mode: password or signature")
	addWebhookSecretFlags(cmd, &secret)
	return cmd
}

func newCmdRepoWebhookDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete [<owner>/<repo>] <id>",
		Short:   "Delete a repository webhook",
		Example: "  ag repo webhook delete owner/repo 42 --yes",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			id, err := parseWebhookID(remaining[0])
			if err != nil {
				return err
			}
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}
			if !yes {
				confirmed, err := confirmWebhookAction(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Permanently delete webhook #%d from %s", id, repository.String()))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Deletion cancelled.")
					return nil
				}
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}
			if err := client.Delete(webhookPath(repository, id)); err != nil {
				return fmt.Errorf("failed to delete webhook #%d: %w", id, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted webhook #%d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func newCmdRepoWebhookTest(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "test [<owner>/<repo>] <id>",
		Short:   "Send a test payload to a repository webhook",
		Example: "  ag repo webhook test owner/repo 42 --yes",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			id, err := parseWebhookID(remaining[0])
			if err != nil {
				return err
			}
			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}
			if !yes {
				confirmed, err := confirmWebhookAction(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Send a real test payload through webhook #%d in %s", id, repository.String()))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Webhook test cancelled.")
					return nil
				}
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}
			if err := client.Post(webhookPath(repository, id)+"/tests", nil, nil); err != nil {
				return fmt.Errorf("failed to test webhook #%d: %w", id, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Sent test payload through webhook #%d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func webhookAPIClient(f *cmdutil.Factory) (*api.Client, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, cmdutil.AuthenticationError(err)
	}
	return f.NewAPIClient(token)
}

func webhookCollectionPath(repository cmdutil.Repository) string {
	return fmt.Sprintf("/repos/%s/%s/hooks", repository.Owner, repository.Name)
}

func webhookPath(repository cmdutil.Repository, id int64) string {
	return webhookCollectionPath(repository) + "/" + strconv.FormatInt(id, 10)
}

func getWebhook(client *api.Client, repository cmdutil.Repository, id int64) (api.Webhook, error) {
	resp, err := client.DoRequestRaw(http.MethodGet, webhookPath(repository, id))
	if err != nil {
		return api.Webhook{}, fmt.Errorf("failed to get webhook #%d: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return api.Webhook{}, fmt.Errorf("failed to get webhook #%d: API request failed: %s (response body omitted to protect webhook secrets)", id, resp.Status)
	}
	var item api.Webhook
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return api.Webhook{}, fmt.Errorf("failed to get webhook #%d: decode response: %w", id, err)
	}
	return item, nil
}

func listWebhooks(client *api.Client, repository cmdutil.Repository, limit int) ([]api.Webhook, error) {
	const maxPerPage = 100
	items := make([]api.Webhook, 0, limit)
	for page := 1; len(items) < limit; page++ {
		path := fmt.Sprintf("%s?page=%d&per_page=%d", webhookCollectionPath(repository), page, maxPerPage)
		resp, err := client.DoRequestRaw(http.MethodGet, path)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("API request failed: %s (response body omitted to protect webhook secrets)", resp.Status)
		}
		var pageItems []api.Webhook
		decodeErr := json.NewDecoder(resp.Body).Decode(&pageItems)
		resp.Body.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("decode webhook list response: %w", decodeErr)
		}
		items = append(items, pageItems...)
		if len(pageItems) < maxPerPage {
			break
		}
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func mutateWebhook(client *api.Client, method, path string, body map[string]interface{}, result *api.Webhook) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode webhook request: %w", err)
	}
	resp, err := client.DoRequestRawWithBody(method, path, encoded, "application/json", "application/json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API request failed: %s (response body omitted to protect webhook secrets)", resp.Status)
	}
	if result == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil && err != io.EOF {
		return fmt.Errorf("decode webhook response: %w", err)
	}
	return nil
}

func parseWebhookID(value string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid webhook id %q (expected a positive integer)", value)
	}
	return id, nil
}

func validateWebhookURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("invalid webhook URL %q (expected an HTTP(S) URL without credentials)", value)
	}
	return parsed.String(), nil
}

func parseWebhookEvents(value string, allowNone bool) (map[string]bool, error) {
	value = strings.TrimSpace(value)
	if allowNone && strings.EqualFold(value, "none") {
		result := make(map[string]bool, len(webhookEventFields))
		for name := range webhookEventFields {
			result[name] = false
		}
		return result, nil
	}
	if value == "" {
		return nil, fmt.Errorf("at least one webhook event is required")
	}
	result := make(map[string]bool, len(webhookEventFields))
	for _, part := range strings.Split(value, ",") {
		name := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(part)), "_", "-")
		if _, ok := webhookEventFields[name]; !ok {
			return nil, fmt.Errorf("unsupported webhook event %q (expected push, tag-push, issues, note, or merge-requests)", strings.TrimSpace(part))
		}
		result[name] = true
	}
	return result, nil
}

func applyWebhookEvents(body map[string]interface{}, selected map[string]bool) {
	for name, field := range webhookEventFields {
		body[field] = selected[name]
	}
}

func parseWebhookEncryption(value string) (int, bool, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return 0, false, nil
	}
	switch value {
	case "password":
		return 0, true, nil
	case "signature":
		return 1, true, nil
	default:
		return 0, false, fmt.Errorf("invalid webhook encryption %q (expected password or signature)", value)
	}
}

func addWebhookSecretFlags(cmd *cobra.Command, options *webhookSecretOptions) {
	cmd.Flags().StringVar(&options.EnvName, "secret-env", "", "Read the webhook secret from this environment variable")
	cmd.Flags().StringVar(&options.File, "secret-file", "", "Read the webhook secret from a file")
	cmd.Flags().BoolVar(&options.Stdin, "secret-stdin", false, "Read the webhook secret from standard input")
	cmd.MarkFlagsMutuallyExclusive("secret-env", "secret-file", "secret-stdin")
}

func webhookSecretFlagsChanged(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("secret-env") || cmd.Flags().Changed("secret-file") || cmd.Flags().Changed("secret-stdin")
}

func readWebhookSecret(in io.Reader, options webhookSecretOptions) (string, bool, error) {
	sources := 0
	if strings.TrimSpace(options.EnvName) != "" {
		sources++
	}
	if strings.TrimSpace(options.File) != "" {
		sources++
	}
	if options.Stdin {
		sources++
	}
	if sources == 0 {
		return "", false, nil
	}
	if sources > 1 {
		return "", false, fmt.Errorf("webhook secret sources are mutually exclusive")
	}
	var value string
	if options.EnvName != "" {
		var found bool
		value, found = os.LookupEnv(strings.TrimSpace(options.EnvName))
		if !found {
			return "", false, fmt.Errorf("webhook secret environment variable %q is not set", strings.TrimSpace(options.EnvName))
		}
	} else if options.File != "" {
		contents, err := os.ReadFile(strings.TrimSpace(options.File))
		if err != nil {
			return "", false, fmt.Errorf("read webhook secret file: %w", err)
		}
		value = string(contents)
	} else {
		contents, err := io.ReadAll(io.LimitReader(in, (1<<20)+1))
		if err != nil {
			return "", false, fmt.Errorf("read webhook secret from stdin: %w", err)
		}
		if len(contents) > 1<<20 {
			return "", false, fmt.Errorf("webhook secret from stdin exceeds 1 MiB")
		}
		value = string(contents)
	}
	value = strings.TrimSuffix(strings.TrimSuffix(value, "\n"), "\r")
	if value == "" {
		return "", false, fmt.Errorf("webhook secret must not be empty")
	}
	return value, true, nil
}

func confirmWebhookAction(in io.Reader, out io.Writer, prompt string) (bool, error) {
	fmt.Fprintf(out, "%s? [y/N] ", prompt)
	var response string
	if _, err := fmt.Fscan(in, &response); err != nil && err != io.EOF {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	return strings.EqualFold(response, "y") || strings.EqualFold(response, "yes"), nil
}

func webhookEvents(item api.Webhook) []string {
	events := make([]string, 0, len(webhookEventFields))
	for _, event := range []struct {
		name    string
		enabled api.FlexibleBool
	}{
		{name: "push", enabled: item.PushEvents},
		{name: "tag-push", enabled: item.TagPushEvents},
		{name: "issues", enabled: item.IssuesEvents},
		{name: "note", enabled: item.NoteEvents},
		{name: "merge-requests", enabled: item.MergeRequestsEvents},
	} {
		if bool(event.enabled) {
			events = append(events, event.name)
		}
	}
	if len(events) == 0 {
		return []string{"none"}
	}
	return events
}

func webhookActiveState(item api.Webhook) string {
	if bool(item.Active) {
		return "active"
	}
	return "inactive"
}

type webhookJSON struct {
	ID         int64    `json:"id"`
	URL        string   `json:"url"`
	Active     bool     `json:"active"`
	Events     []string `json:"events"`
	Result     string   `json:"result"`
	ResultCode int      `json:"resultCode"`
	CreatedAt  string   `json:"createdAt"`
}

func webhooksJSON(items []api.Webhook) []webhookJSON {
	result := make([]webhookJSON, len(items))
	for index, item := range items {
		result[index] = newWebhookJSON(item)
	}
	return result
}

func newWebhookJSON(item api.Webhook) webhookJSON {
	return webhookJSON{
		ID: item.ID, URL: item.URL, Active: bool(item.Active), Events: webhookEvents(item),
		Result: item.Result, ResultCode: item.ResultCode, CreatedAt: item.CreatedAt,
	}
}
