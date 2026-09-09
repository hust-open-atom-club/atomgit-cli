package repo

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type remoteMirrorJSON struct {
	Type                   string  `json:"type"`
	Repository             string  `json:"repository"`
	ID                     *int64  `json:"id,omitempty"`
	ProjectID              *int64  `json:"projectId,omitempty"`
	Status                 *string `json:"status,omitempty"`
	Destination            *string `json:"destination,omitempty"`
	LastUpdateAt           *string `json:"lastUpdateAt,omitempty"`
	LastSuccessfulUpdateAt *string `json:"lastSuccessfulUpdateAt,omitempty"`
	FailureCount           *int    `json:"failureCount,omitempty"`
	Enabled                *bool   `json:"enabled,omitempty"`
	Private                *bool   `json:"private,omitempty"`
	LastError              *string `json:"lastError,omitempty"`
	Message                *string `json:"message,omitempty"`
	Force                  *bool   `json:"force,omitempty"`
	CreatedAt              *string `json:"createdAt,omitempty"`
	UpdatedAt              *string `json:"updatedAt,omitempty"`
}

func newCmdRepoMirror(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mirror",
		Short: "Inspect repository remote mirrors",
		Long:  "Inspect configured push mirrors and repository mirror state. These commands are read-only and do not synchronize local Git remotes.",
	}
	cmd.AddCommand(newCmdRepoMirrorList(f))
	cmd.AddCommand(newCmdRepoMirrorView(f))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func newCmdRepoMirrorList(f *cmdutil.Factory) *cobra.Command {
	var opts struct {
		Limit int
		JSON  bool
	}
	cmd := &cobra.Command{
		Use:   "list [<owner>/<repo>]",
		Short: "List configured push remote mirrors",
		Long:  "List configured push remote mirrors. Destinations and returned messages are sanitized before text or JSON output.",
		Example: `  ag repo mirror list owner/repo
  ag repo mirror list owner/repo --limit 100 --json
  ag repo mirror list`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Limit <= 0 {
				return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
			}
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := remoteMirrorAPIClient(f, repository)
			if err != nil {
				return err
			}
			mirrors, err := api.ListRepositoryPushRemoteMirrors(client, repository.Owner, repository.Name, opts.Limit)
			if err != nil {
				return fmt.Errorf("failed to list push remote mirrors for %s: %w", repository, err)
			}

			if opts.JSON {
				output := make([]remoteMirrorJSON, len(mirrors))
				for index, mirror := range mirrors {
					output[index] = newRemoteMirrorJSON("push", repository, mirror)
				}
				return cmdutil.WriteJSON(cmd.OutOrStdout(), output)
			}
			if len(mirrors) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No push remote mirrors configured for %s.\n", repository)
				return nil
			}
			for index, mirror := range mirrors {
				if index > 0 {
					fmt.Fprintln(cmd.OutOrStdout())
				}
				printRemoteMirror(cmd.OutOrStdout(), "Push remote mirror", repository, mirror)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of push remote mirrors to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output push remote mirrors as JSON")
	return cmd
}

func newCmdRepoMirrorView(f *cmdutil.Factory) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "view [<owner>/<repo>]",
		Short:   "View repository remote mirror state",
		Long:    "View repository remote mirror state. Destinations and returned errors are sanitized before text or JSON output.",
		Example: "  ag repo mirror view owner/repo\n  ag repo mirror view --json",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			client, err := remoteMirrorAPIClient(f, repository)
			if err != nil {
				return err
			}
			mirror, err := api.GetRepositoryRemoteMirror(client, repository.Owner, repository.Name)
			if err != nil {
				if api.IsHTTPStatus(err, http.StatusNotFound) {
					return fmt.Errorf("repository remote mirror was not found for %s: %w", repository, err)
				}
				return fmt.Errorf("failed to view repository remote mirror for %s: %w", repository, err)
			}

			if jsonOutput {
				return cmdutil.WriteJSON(cmd.OutOrStdout(), newRemoteMirrorJSON("repository", repository, mirror))
			}
			printRemoteMirror(cmd.OutOrStdout(), "Repository remote mirror", repository, mirror)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output repository remote mirror state as JSON")
	return cmd
}

func remoteMirrorAPIClient(f *cmdutil.Factory, repository cmdutil.Repository) (*api.Client, error) {
	token, err := f.Config.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect remote mirrors for %s: %w", repository, cmdutil.AuthenticationError(err))
	}
	client, err := f.NewAPIClient(token)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect remote mirrors for %s: %w", repository, err)
	}
	return client, nil
}

func newRemoteMirrorJSON(mirrorType string, repository cmdutil.Repository, mirror api.RemoteMirror) remoteMirrorJSON {
	return remoteMirrorJSON{
		Type:                   mirrorType,
		Repository:             repository.String(),
		ID:                     mirror.ID,
		ProjectID:              mirror.ProjectID,
		Status:                 mirror.UpdateStatus,
		Destination:            mirror.URL,
		LastUpdateAt:           mirror.LastUpdateAt,
		LastSuccessfulUpdateAt: mirror.LastSuccessfulUpdateAt,
		FailureCount:           mirror.NumberOfFailures,
		Enabled:                mirror.MirroringEnabled,
		Private:                mirror.IsPrivate,
		LastError:              mirror.LastError,
		Message:                mirror.Message,
		Force:                  mirror.Force,
		CreatedAt:              mirror.CreatedAt,
		UpdatedAt:              mirror.UpdatedAt,
	}
}

func printRemoteMirror(out io.Writer, heading string, repository cmdutil.Repository, mirror api.RemoteMirror) {
	fmt.Fprintln(out, heading)
	fmt.Fprintf(out, "  Repository: %s\n", repository)
	printRemoteMirrorInt64(out, "ID", mirror.ID)
	printRemoteMirrorInt64(out, "Project ID", mirror.ProjectID)
	printRemoteMirrorString(out, "Destination", mirror.URL)
	printRemoteMirrorString(out, "Status", mirror.UpdateStatus)
	printRemoteMirrorBool(out, "Enabled", mirror.MirroringEnabled)
	printRemoteMirrorBool(out, "Private", mirror.IsPrivate)
	printRemoteMirrorBool(out, "Force", mirror.Force)
	printRemoteMirrorInt(out, "Failures", mirror.NumberOfFailures)
	printRemoteMirrorString(out, "Last update", mirror.LastUpdateAt)
	printRemoteMirrorString(out, "Last successful update", mirror.LastSuccessfulUpdateAt)
	printRemoteMirrorString(out, "Created", mirror.CreatedAt)
	printRemoteMirrorString(out, "Updated", mirror.UpdatedAt)
	printRemoteMirrorString(out, "Last error", mirror.LastError)
	printRemoteMirrorString(out, "Message", mirror.Message)
}

func printRemoteMirrorString(out io.Writer, label string, value *string) {
	if value != nil && strings.TrimSpace(*value) != "" {
		fmt.Fprintf(out, "  %s: %s\n", label, *value)
	}
}

func printRemoteMirrorInt64(out io.Writer, label string, value *int64) {
	if value != nil {
		fmt.Fprintf(out, "  %s: %d\n", label, *value)
	}
}

func printRemoteMirrorInt(out io.Writer, label string, value *int) {
	if value != nil {
		fmt.Fprintf(out, "  %s: %d\n", label, *value)
	}
}

func printRemoteMirrorBool(out io.Writer, label string, value *bool) {
	if value != nil {
		fmt.Fprintf(out, "  %s: %t\n", label, *value)
	}
}
