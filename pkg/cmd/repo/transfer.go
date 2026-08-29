package repo

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	maxTransferPasswordBytes  = 64 << 10
	transferNamespacePageSize = 100
	maxTransferNamespacePages = 100
)

type transferOptions struct {
	Destination   string
	Yes           bool
	PasswordStdin bool
}

type repositoryOwnerKind int

const (
	repositoryOwnerUnknown repositoryOwnerKind = iota
	repositoryOwnerUser
	repositoryOwnerOrganization
)

func newCmdRepoTransfer(f *cmdutil.Factory) *cobra.Command {
	opts := &transferOptions{}
	cmd := &cobra.Command{
		Use:   "transfer [<owner>/<repo>] --to <organization>",
		Short: "Transfer a repository to an organization",
		Long: `Transfer a repository to an AtomGit organization.

Repository transfer can change access, URLs, and automation. The command shows
the source and resolved destination organization and requires confirmation
unless --yes is supplied. The currently documented AtomGit transfer APIs only
describe organization destinations, so personal-user destinations are rejected.
After the transfer, the authoritative repository name and URL are read back from
AtomGit. Local Git remotes are never modified.

AtomGit requires the account password when the source repository is owned by
an organization. It is read without echo from an interactive terminal, or from
standard input when --password-stdin is combined with --yes.`,
		Example: `  ag repo transfer owner/repo --to target-organization
  ag repo transfer owner/repo --to target-organization --yes
  printf '%s\n' "$PASSWORD" | ag repo transfer source-organization/repo --to target-organization --yes --password-stdin`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, _, err := cmdutil.ResolveRepositoryFromArgs(f, args, 0)
			if err != nil {
				return err
			}
			destination, err := validateTransferDestination(opts.Destination)
			if err != nil {
				return err
			}
			if opts.PasswordStdin && !opts.Yes {
				return fmt.Errorf("--password-stdin requires --yes because standard input cannot also answer the confirmation prompt")
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}
			return runRepositoryTransfer(cmd, client, repository, destination, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Destination, "to", "", "Destination organization namespace (required)")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip the confirmation prompt")
	cmd.Flags().BoolVar(&opts.PasswordStdin, "password-stdin", false, "Read the organization-transfer password from standard input (requires --yes)")
	_ = cmd.MarkFlagRequired("to")
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func validateTransferDestination(value string) (string, error) {
	destination := strings.TrimSpace(value)
	if destination == "" {
		return "", fmt.Errorf("destination namespace must not be empty")
	}
	if destination != value || cmdutil.InvalidRepositoryPart(destination) || strings.IndexFunc(destination, unicode.IsSpace) >= 0 {
		return "", fmt.Errorf("invalid destination namespace %q", value)
	}
	return destination, nil
}

func runRepositoryTransfer(cmd *cobra.Command, client *api.Client, repository cmdutil.Repository, destination string, opts *transferOptions) error {
	var source api.Repository
	sourcePath := repositoryAPIPath(repository.Owner, repository.Name)
	if err := client.Get(sourcePath, &source); err != nil {
		return fmt.Errorf("failed to inspect source repository %s: %w", repository, err)
	}

	resolvedDestination, err := resolveTransferDestination(client, destination)
	if err != nil {
		return err
	}
	destination = resolvedDestination
	kind, err := classifyRepositoryOwner(source, repository.Owner)
	if err != nil {
		return fmt.Errorf("cannot determine documented transfer path for %s to organization %s: %w", repository, destination, err)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Source: %s\n", repository)
	fmt.Fprintf(out, "Destination organization: %s\n", destination)
	if !opts.Yes {
		confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), fmt.Sprintf("Transfer %s to %s? Repository URLs and access may change. [y/N] ", repository, destination))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(out, "Transfer cancelled.")
			return nil
		}
	}

	newOwner := destination
	newName := repository.Name
	switch kind {
	case repositoryOwnerUser:
		if opts.PasswordStdin {
			return fmt.Errorf("--password-stdin is only valid for an organization-owned source repository")
		}
		response, err := api.TransferRepository(client, repository.Owner, repository.Name, destination)
		if err != nil {
			return handleTransferRequestError(out, client, repository, destination, repository.Name, fmt.Sprintf("failed to transfer repository %s to %s", repository, destination), err)
		}
		if value := strings.TrimSpace(response.NewOwner); value != "" {
			if !strings.EqualFold(value, destination) {
				return fmt.Errorf("ambiguous transfer result for %s: API reported destination owner %q, expected %q", repository, value, destination)
			}
			newOwner = value
		}
		if value := strings.TrimSpace(response.NewName); value != "" {
			if cmdutil.InvalidRepositoryPart(value) {
				return fmt.Errorf("ambiguous transfer result for %s: API reported invalid repository name %q", repository, value)
			}
			newName = value
		}
	case repositoryOwnerOrganization:
		password, err := readTransferPassword(cmd, opts.PasswordStdin)
		if err != nil {
			return err
		}
		response, err := api.TransferOrganizationRepository(client, repository.Owner, repository.Name, destination, password)
		password = ""
		if err != nil {
			return handleTransferRequestError(out, client, repository, destination, repository.Name, fmt.Sprintf("failed to transfer organization repository %s to %s", repository, destination), err)
		}
		if response.Code != 1 {
			return fmt.Errorf("ambiguous transfer result for %s: organization transfer returned code %d", repository, response.Code)
		}
	default:
		return fmt.Errorf("cannot determine transfer endpoint for %s", repository)
	}

	return verifyAndReportTransferredRepository(out, client, repository, newOwner, newName)
}

func resolveTransferDestination(client *api.Client, destination string) (string, error) {
	for page := 1; page <= maxTransferNamespacePages; page++ {
		query := url.Values{}
		query.Set("mode", "all")
		query.Set("page", strconv.Itoa(page))
		query.Set("perPage", strconv.Itoa(transferNamespacePageSize))
		var namespaces []api.Namespace
		if err := client.Get("/user/namespaces?"+query.Encode(), &namespaces); err != nil {
			return "", fmt.Errorf("failed to resolve destination organization %q: %w", destination, err)
		}
		for _, namespace := range namespaces {
			path := strings.TrimSpace(namespace.Path)
			if !strings.EqualFold(path, destination) {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(namespace.Type)) {
			case "group", "organization", "org":
				return path, nil
			case "user", "personal":
				return "", fmt.Errorf("destination %q is a personal-user namespace; the documented AtomGit transfer APIs only support organization destinations", destination)
			default:
				return "", fmt.Errorf("destination %q has unsupported namespace type %q; the documented AtomGit transfer APIs only support organization destinations", destination, namespace.Type)
			}
		}
		if len(namespaces) < transferNamespacePageSize {
			return "", fmt.Errorf("destination organization %q was not found among namespaces visible to the authenticated user", destination)
		}
	}
	return "", fmt.Errorf("destination organization %q could not be resolved within %d namespace pages", destination, maxTransferNamespacePages)
}

func handleTransferRequestError(out io.Writer, client *api.Client, repository cmdutil.Repository, destination, name, httpErrorContext string, transferErr error) error {
	var httpErr *api.HTTPError
	if errors.As(transferErr, &httpErr) {
		return fmt.Errorf("%s: %w", httpErrorContext, transferErr)
	}

	transferred, readBackErr := readBackTransferredRepository(client, destination, name)
	if readBackErr != nil {
		return fmt.Errorf("transfer request for %s may have completed, but its final state is unknown: the transfer result could not be read (%v), and destination read-back for %s/%s failed: %w", repository, transferErr, destination, name, readBackErr)
	}
	fullName, repositoryURL, verifyErr := verifiedTransferredIdentity(transferred, destination, name)
	if verifyErr != nil {
		return fmt.Errorf("transfer request for %s may have completed, but its final state is unknown: the transfer result could not be read (%v), and destination read-back was ambiguous: %w", repository, transferErr, verifyErr)
	}
	reportTransferredRepository(out, fullName, repositoryURL, true)
	return nil
}

func verifyAndReportTransferredRepository(out io.Writer, client *api.Client, repository cmdutil.Repository, newOwner, newName string) error {
	transferred, err := readBackTransferredRepository(client, newOwner, newName)
	if err != nil {
		return fmt.Errorf("transfer request for %s succeeded, but the final repository %s/%s could not be verified: %w", repository, newOwner, newName, err)
	}
	fullName, repositoryURL, err := verifiedTransferredIdentity(transferred, newOwner, newName)
	if err != nil {
		return fmt.Errorf("transfer request for %s succeeded, but the final state is ambiguous: %w", repository, err)
	}

	reportTransferredRepository(out, fullName, repositoryURL, false)
	return nil
}

func reportTransferredRepository(out io.Writer, fullName, repositoryURL string, recovered bool) {
	fmt.Fprintf(out, "✓ Transferred repository to %s\n", fullName)
	fmt.Fprintf(out, "  URL: %s\n", repositoryURL)
	if recovered {
		fmt.Fprintln(out, "  Confirmed by destination read-back after the transfer result could not be read.")
	}
	fmt.Fprintln(out, "  Local Git remotes were not changed.")
}

func classifyRepositoryOwner(repository api.Repository, expectedOwner string) (repositoryOwnerKind, error) {
	namespace := strings.TrimSpace(repository.Namespace.Path)
	owner := strings.TrimSpace(repository.Owner.Login)
	ownerType := strings.ToLower(strings.TrimSpace(repository.Owner.Type))

	if namespace != "" {
		if !strings.EqualFold(namespace, expectedOwner) {
			return repositoryOwnerUnknown, fmt.Errorf("API reported source namespace %q, expected %q", namespace, expectedOwner)
		}
	}
	switch ownerType {
	case "organization", "org", "group":
		return repositoryOwnerOrganization, nil
	case "user", "personal":
		if strings.EqualFold(owner, expectedOwner) {
			return repositoryOwnerUser, nil
		}
		if namespace != "" {
			// Organization repository responses identify the acting user in
			// owner while namespace carries the repository owner.
			return repositoryOwnerOrganization, nil
		}
		return repositoryOwnerUnknown, fmt.Errorf("API reported source owner %q, expected %q", owner, expectedOwner)
	}
	if owner != "" {
		if strings.EqualFold(owner, expectedOwner) {
			return repositoryOwnerUser, nil
		}
		if namespace != "" {
			return repositoryOwnerOrganization, nil
		}
		return repositoryOwnerUnknown, fmt.Errorf("API reported source owner %q, expected %q", owner, expectedOwner)
	}
	if namespace != "" {
		return repositoryOwnerOrganization, nil
	}
	return repositoryOwnerUnknown, fmt.Errorf("API response omitted source owner type and namespace")
}

func readTransferPassword(cmd *cobra.Command, fromStdin bool) (string, error) {
	if fromStdin {
		contents, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxTransferPasswordBytes+1))
		if err != nil {
			return "", fmt.Errorf("read organization-transfer password from stdin: %w", err)
		}
		if len(contents) > maxTransferPasswordBytes {
			return "", fmt.Errorf("organization-transfer password from stdin exceeds %d bytes", maxTransferPasswordBytes)
		}
		password := strings.TrimSuffix(strings.TrimSuffix(string(contents), "\n"), "\r")
		if password == "" {
			return "", fmt.Errorf("organization-transfer password must not be empty")
		}
		return password, nil
	}

	file, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return "", fmt.Errorf("organization-owned repository transfer requires an interactive password; use --yes --password-stdin for non-interactive use")
	}
	if _, err := fmt.Fprint(cmd.ErrOrStderr(), "AtomGit password: "); err != nil {
		return "", fmt.Errorf("write password prompt: %w", err)
	}
	contents, err := term.ReadPassword(int(file.Fd()))
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", fmt.Errorf("read organization-transfer password: %w", err)
	}
	if len(contents) == 0 {
		return "", fmt.Errorf("organization-transfer password must not be empty")
	}
	return string(contents), nil
}

func readBackTransferredRepository(client *api.Client, owner, repo string) (api.Repository, error) {
	var repository api.Repository
	if err := client.Get(repositoryAPIPath(owner, repo), &repository); err != nil {
		return api.Repository{}, err
	}
	return repository, nil
}

func verifiedTransferredIdentity(repository api.Repository, expectedOwner, expectedName string) (string, string, error) {
	owner := strings.TrimSpace(repository.Namespace.Path)
	if owner == "" {
		owner = strings.TrimSpace(repository.Owner.Login)
	}
	name := strings.TrimSpace(repository.Name)
	fullName := strings.Trim(strings.TrimSpace(repository.FullName), "/")
	if fullName != "" {
		parts := strings.Split(fullName, "/")
		if len(parts) == 2 {
			if owner == "" {
				owner = parts[0]
			}
			if name == "" {
				name = parts[1]
			}
		}
	}
	if owner == "" || name == "" {
		return "", "", fmt.Errorf("API response omitted the repository owner or name")
	}
	if !strings.EqualFold(owner, expectedOwner) || name != expectedName {
		return "", "", fmt.Errorf("API returned %s/%s, expected %s/%s", owner, name, expectedOwner, expectedName)
	}
	fullName = owner + "/" + name
	repositoryURL := strings.TrimSpace(repository.HTMLURL)
	if repositoryURL == "" {
		repositoryURL = strings.TrimSpace(repository.AlternateHTMLURL)
	}
	if repositoryURL == "" {
		return "", "", fmt.Errorf("API response for %s omitted the repository URL", fullName)
	}
	return fullName, repositoryURL, nil
}

func repositoryAPIPath(owner, repo string) string {
	return "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo)
}
