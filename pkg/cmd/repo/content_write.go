package repo

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/commandschema"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const maxRepositoryContentBytes = 10 << 20

type contentWriteOptions struct {
	file string
	sha  string
	yes  bool
}

func newCmdRepoContentCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &contentWriteOptions{}
	cmd := &cobra.Command{
		Use:   "create [<owner>/<repo>] <path> --branch <branch> --message <message> (--file <path> | --file -)",
		Short: "Create a repository file",
		Long: `Create one file through the AtomGit contents API.

The file bytes are read from --file or stdin and sent as Base64. Bytes are not
trimmed or re-encoded as text. An empty file is valid. This command does not
read the destination first and does not update an existing file.

--branch is required so the write cannot fall back to the repository default
branch. The request is not retried.`,
		Example: `  ag-cli repo content create owner/repo docs/guide.md --branch main --message "Add guide" --file guide.md
  ag-cli repo content create README.md --branch main --message "Add README" --file -`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, path, branch, message, err := resolveContentWrite(f, cmd, args)
			if err != nil {
				return err
			}
			content, err := readContentBytes(opts.file, cmd.InOrStdin())
			if err != nil {
				return err
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			change, err := api.CreateRepositoryContent(client, repository.Owner, repository.Name, path, branch, message, content)
			if err != nil {
				return err
			}
			printContentChange(cmd.OutOrStdout(), "Created", path, branch, change)
			return nil
		},
	}
	addContentMutationFlags(cmd, opts, true, false)
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals:   &commandschema.Positionals{MinCount: 1, MaxCount: 2, Description: "Repository-relative file path, optionally preceded by owner/repo."},
		RequiredFlags: []string{"branch", "message", "file"},
		Output:        "text", Effects: "write",
		Notes: []string{
			"Creates one file with POST and does not convert a conflict into an update.",
			"Reads raw bytes, including empty and binary files, and sends Base64 JSON. The request is not retried.",
		},
	})
	return cmd
}

func newCmdRepoContentUpdate(f *cmdutil.Factory) *cobra.Command {
	opts := &contentWriteOptions{}
	cmd := &cobra.Command{
		Use:   "update [<owner>/<repo>] <path> --branch <branch> --message <message> --sha <blob-sha> (--file <path> | --file -)",
		Short: "Update a repository file",
		Long: `Replace one file through the AtomGit contents API.

--sha is the current file Blob SHA, not a commit SHA. A conflicting SHA fails
without reading a newer SHA or retrying the write. The file bytes are read
from --file or stdin and are not trimmed or re-encoded as text.`,
		Example: `  ag-cli repo content update owner/repo docs/guide.md --branch main --message "Update guide" --sha abcdef1234567890abcdef1234567890abcdef12 --file guide.md
  ag-cli repo content update README.md --branch main --message "Update README" --sha abcdef1234567890abcdef1234567890abcdef12 --file -`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, path, branch, message, err := resolveContentWrite(f, cmd, args)
			if err != nil {
				return err
			}
			if err := validateContentBlobSHA(opts.sha); err != nil {
				return err
			}
			content, err := readContentBytes(opts.file, cmd.InOrStdin())
			if err != nil {
				return err
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			change, err := api.UpdateRepositoryContent(client, repository.Owner, repository.Name, path, branch, message, opts.sha, content)
			if err != nil {
				return err
			}
			printContentChange(cmd.OutOrStdout(), "Updated", path, branch, change)
			return nil
		},
	}
	addContentMutationFlags(cmd, opts, true, true)
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals:   &commandschema.Positionals{MinCount: 1, MaxCount: 2, Description: "Repository-relative file path, optionally preceded by owner/repo."},
		RequiredFlags: []string{"branch", "message", "sha", "file"},
		Output:        "text", Effects: "write",
		Notes: []string{
			"Replaces one file with PUT. --sha is the current file Blob SHA; conflicts are not refreshed or retried.",
			"Reads raw bytes, including empty and binary files, and sends Base64 JSON.",
		},
	})
	return cmd
}

func newCmdRepoContentDelete(f *cmdutil.Factory) *cobra.Command {
	opts := &contentWriteOptions{}
	cmd := &cobra.Command{
		Use:   "delete [<owner>/<repo>] <path> --branch <branch> --message <message> --sha <blob-sha>",
		Short: "Delete a repository file",
		Long: `Delete one file through the AtomGit contents API.

--sha is the current file Blob SHA, not a commit SHA. Confirmation happens
before authentication unless --yes is set. Cancelling sends no request. A
conflicting SHA fails without reading a newer SHA or retrying.`,
		Example: `  ag-cli repo content delete owner/repo docs/guide.md --branch main --message "Remove guide" --sha abcdef1234567890abcdef1234567890abcdef12
  ag-cli repo content delete README.md --branch main --message "Remove README" --sha abcdef1234567890abcdef1234567890abcdef12 --yes`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, path, branch, message, err := resolveContentWrite(f, cmd, args)
			if err != nil {
				return err
			}
			if err := validateContentBlobSHA(opts.sha); err != nil {
				return err
			}
			if !opts.yes {
				confirmed, err := cmdutil.Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), fmt.Sprintf("Delete %s from %s on branch %s? [y/N] ", path, repository.String(), branch))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Deletion cancelled.")
					return nil
				}
			}
			client, err := f.AuthenticatedAPIClient()
			if err != nil {
				return err
			}
			change, err := api.DeleteRepositoryContent(client, repository.Owner, repository.Name, path, branch, message, opts.sha)
			if err != nil {
				return err
			}
			printContentChange(cmd.OutOrStdout(), "Deleted", path, branch, change)
			return nil
		},
	}
	addContentMutationFlags(cmd, opts, false, true)
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip confirmation prompt")
	commandschema.Annotate(cmd, commandschema.Metadata{
		Positionals:   &commandschema.Positionals{MinCount: 1, MaxCount: 2, Description: "Repository-relative file path, optionally preceded by owner/repo."},
		RequiredFlags: []string{"branch", "message", "sha"},
		Output:        "text", Effects: "write",
		Notes: []string{
			"Deletes one file with DELETE. --sha is the current file Blob SHA; conflicts are not refreshed or retried.",
			"Confirms before authentication unless --yes is set. Cancellation sends no request.",
		},
	})
	return cmd
}

func addContentMutationFlags(cmd *cobra.Command, opts *contentWriteOptions, withFile, withSHA bool) {
	cmd.Flags().String("branch", "", "Branch to modify (required)")
	cmd.Flags().String("message", "", "Commit message (required)")
	_ = cmd.MarkFlagRequired("branch")
	_ = cmd.MarkFlagRequired("message")
	if withFile {
		cmd.Flags().StringVar(&opts.file, "file", "", "Read raw file bytes from a path, or - for stdin")
		_ = cmd.MarkFlagRequired("file")
	}
	if withSHA {
		cmd.Flags().StringVar(&opts.sha, "sha", "", "Current file Blob SHA")
		_ = cmd.MarkFlagRequired("sha")
	}
}

func resolveContentWrite(f *cmdutil.Factory, cmd *cobra.Command, args []string) (cmdutil.Repository, string, string, string, error) {
	repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
	if err != nil {
		return cmdutil.Repository{}, "", "", "", err
	}
	path := remaining[0]
	if err := validateContentPath(path, false); err != nil {
		return cmdutil.Repository{}, "", "", "", err
	}
	branch, err := requiredContentValue(cmd, "branch")
	if err != nil {
		return cmdutil.Repository{}, "", "", "", err
	}
	message, err := requiredContentValue(cmd, "message")
	if err != nil {
		return cmdutil.Repository{}, "", "", "", err
	}
	return repository, path, branch, message, nil
}

func requiredContentValue(cmd *cobra.Command, name string) (string, error) {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("--%s cannot be empty", name)
	}
	return value, nil
}

func validateContentBlobSHA(sha string) error {
	if len(sha) != 40 || strings.TrimLeft(sha, "0123456789abcdefABCDEF") != "" {
		return errors.New("--sha must be a 40-character file Blob SHA")
	}
	if _, err := hex.DecodeString(sha); err != nil {
		return errors.New("--sha must be a 40-character file Blob SHA")
	}
	return nil
}

func readContentBytes(path string, stdin io.Reader) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("--file cannot be empty")
	}
	var (
		content []byte
		err     error
	)
	if path == "-" {
		content, err = io.ReadAll(io.LimitReader(stdin, maxRepositoryContentBytes+1))
		if err != nil {
			return nil, fmt.Errorf("read content from stdin: %w", err)
		}
	} else {
		file, openErr := os.Open(path)
		if openErr != nil {
			return nil, fmt.Errorf("read content file: %w", openErr)
		}
		defer file.Close()
		content, err = io.ReadAll(io.LimitReader(file, maxRepositoryContentBytes+1))
		if err != nil {
			return nil, fmt.Errorf("read content file: %w", err)
		}
	}
	if len(content) > maxRepositoryContentBytes {
		return nil, fmt.Errorf("content exceeds %d bytes", maxRepositoryContentBytes)
	}
	return content, nil
}

func printContentChange(out io.Writer, action, path, branch string, change api.RepositoryContentChange) {
	fmt.Fprintf(out, "%s %s on %s\n", action, path, branch)
	if change.Commit != nil && change.Commit.SHA != "" {
		fmt.Fprintf(out, "Commit: %s\n", change.Commit.SHA)
	}
	if change.Content != nil && change.Content.SHA != "" {
		fmt.Fprintf(out, "Blob: %s\n", change.Content.SHA)
	}
}
