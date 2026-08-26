package user

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"text/tabwriter"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const (
	defaultNamespaceMode      = "intrant"
	defaultNamespaceListLimit = 30
)

type namespaceOptions struct {
	Mode  string
	Limit int
	JSON  bool
}

type namespaceJSON struct {
	ID   int64  `json:"id"`
	Path string `json:"path"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

func newCmdUserNamespaces(f *cmdutil.Factory) *cobra.Command {
	opts := &namespaceOptions{
		Mode:  defaultNamespaceMode,
		Limit: defaultNamespaceListLimit,
	}
	cmd := &cobra.Command{
		Use:   "namespaces",
		Short: "List namespaces for the authenticated user",
		Long:  "List user and group namespaces visible to the authenticated user. The default mode is intrant.",
		Example: `  ag user namespaces
  ag user namespaces --mode project --limit 100
  ag user namespaces --mode all --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUserNamespaces(cmd.OutOrStdout(), f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Mode, "mode", defaultNamespaceMode, "Namespace source: intrant, project, or all")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", defaultNamespaceListLimit, "Maximum number of namespaces to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output namespaces as JSON")
	return cmd
}

func runUserNamespaces(out io.Writer, f *cmdutil.Factory, opts *namespaceOptions) error {
	if !validNamespaceMode(opts.Mode) {
		return fmt.Errorf("invalid mode %q: must be intrant, project, or all", opts.Mode)
	}
	if opts.Limit <= 0 {
		return fmt.Errorf("invalid limit: %d (must be positive)", opts.Limit)
	}

	token, err := f.Config.GetToken()
	if err != nil {
		return cmdutil.AuthenticationError(err)
	}
	client, err := f.NewAPIClient(token)
	if err != nil {
		return err
	}

	namespaces, err := api.GetPaginated[api.Namespace](client, opts.Limit, func(page, perPage int) string {
		query := url.Values{}
		query.Set("mode", opts.Mode)
		query.Set("page", strconv.Itoa(page))
		query.Set("perPage", strconv.Itoa(perPage))
		return "/user/namespaces?" + query.Encode()
	})
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	if opts.JSON {
		return cmdutil.WriteJSON(out, namespacesJSON(namespaces))
	}
	if len(namespaces) == 0 {
		fmt.Fprintln(out, "No namespaces found.")
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "PATH\tNAME\tTYPE\tURL")
	for _, namespace := range namespaces {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			cmdutil.EscapeTSVField(namespace.Path),
			cmdutil.EscapeTSVField(namespace.Name),
			cmdutil.EscapeTSVField(namespace.Type),
			cmdutil.EscapeTSVField(namespace.HTMLURL),
		)
	}
	return w.Flush()
}

func validNamespaceMode(mode string) bool {
	switch mode {
	case "intrant", "project", "all":
		return true
	default:
		return false
	}
}

func namespacesJSON(namespaces []api.Namespace) []namespaceJSON {
	result := make([]namespaceJSON, len(namespaces))
	for index, namespace := range namespaces {
		result[index] = namespaceJSON{
			ID:   namespace.ID,
			Path: namespace.Path,
			Name: namespace.Name,
			URL:  namespace.HTMLURL,
			Type: namespace.Type,
		}
	}
	return result
}
