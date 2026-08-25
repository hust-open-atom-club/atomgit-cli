package repo

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type readFileOptions struct {
	ref  string
	json bool
}

type readDirOptions struct {
	ref  string
	json bool
}

type contentOptions struct {
	ref  string
	json bool
}

type contentFileJSON struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	SHA      string `json:"sha"`
	Size     int64  `json:"size"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
	Ref      string `json:"ref"`
}

type contentDirEntryJSON struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
	Size int64  `json:"size"`
}

func validateContentPath(path string, allowRoot bool) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}
	if path == "." {
		if !allowRoot {
			return errors.New("'.' is only valid for directory listing; use read-dir for repository root")
		}
		return nil
	}
	if strings.HasPrefix(path, "/") {
		return errors.New("path must not start with '/'")
	}
	if strings.HasSuffix(path, "/") {
		return errors.New("path must not end with '/'")
	}
	if strings.Contains(path, "//") {
		return errors.New("path must not contain repeated separators")
	}
	segments := strings.Split(path, "/")
	for _, s := range segments {
		if s == "." || s == ".." {
			return errors.New("path must not contain '.' or '..' segments")
		}
	}
	return nil
}

func validateContentRef(cmd *cobra.Command, ref string) (string, error) {
	if cmd.Flags().Changed("ref") && strings.TrimSpace(ref) == "" {
		return "", errors.New("--ref cannot be empty")
	}
	return strings.TrimSpace(ref), nil
}

func contentTypeMismatchError(repository cmdutil.Repository, path, ref, actualType, suggestedAction string) error {
	refDescription := "the default branch"
	refGuidance := ""
	if ref != "" {
		refDescription = fmt.Sprintf("ref %q", ref)
		refGuidance = fmt.Sprintf(" and --ref %q", ref)
	}
	return fmt.Errorf(
		"path %q in repository %q at %s is a %s; use 'ag repo content %s' with repository %q, path %q%s instead",
		path,
		repository.String(),
		refDescription,
		actualType,
		suggestedAction,
		repository.String(),
		path,
		refGuidance,
	)
}

func newCmdRepoContent(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "content",
		Short: "Browse repository contents",
		Long:  "List directories and view files without modifying repository contents.",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdRepoContentList(f))
	cmd.AddCommand(newCmdRepoContentView(f))
	cmdutil.AddRepositoryContextHelp(cmd)
	return cmd
}

func newCmdRepoContentView(f *cmdutil.Factory) *cobra.Command {
	opts := &contentOptions{}

	cmd := &cobra.Command{
		Use:   "view [<owner>/<repo>] <path>",
		Short: "View a repository file",
		Example: `  ag repo content view README.md
  ag repo content view owner/repo src/main.go --ref dev
  ag repo content view owner/repo README.md --json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			path := remaining[0]
			if err := validateContentPath(path, false); err != nil {
				return err
			}
			ref, err := validateContentRef(cmd, opts.ref)
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			response, err := api.GetRepositoryContents(client, repository.Owner, repository.Name, path, ref)
			if err != nil {
				return err
			}
			if response.File == nil {
				return contentTypeMismatchError(repository, path, ref, "directory", "list")
			}
			if response.File.Type != "file" {
				return fmt.Errorf("path %q has type %q, not a file", path, response.File.Type)
			}

			out := cmd.OutOrStdout()
			if opts.json {
				return cmdutil.WriteJSON(out, response.Raw)
			}
			if response.File.Encoding != "base64" {
				return fmt.Errorf("unsupported encoding %q; expected %q", response.File.Encoding, "base64")
			}
			if !response.File.ContentPresent {
				return errors.New("file response does not contain content")
			}

			decoded, err := base64.StdEncoding.DecodeString(response.File.Content)
			if err != nil {
				return fmt.Errorf("decode Base64 content for path %q: %w", path, err)
			}
			written, err := out.Write(decoded)
			if err != nil {
				return fmt.Errorf("write content for path %q: %w", path, err)
			}
			if written != len(decoded) {
				return fmt.Errorf("write content for path %q: %w", path, io.ErrShortWrite)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.ref, "ref", "", "Branch, tag, or commit identifier")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output the complete API file object as JSON")
	return cmd
}

func resolveContentListArgs(f *cmdutil.Factory, args []string) (cmdutil.Repository, string, error) {
	switch len(args) {
	case 0:
		repository, err := cmdutil.ResolveRepository(f, "")
		return repository, ".", err
	case 1:
		repository, err := cmdutil.ResolveRepository(f, "")
		return repository, args[0], err
	case 2:
		repository, err := cmdutil.ResolveRepository(f, args[0])
		return repository, args[1], err
	default:
		return cmdutil.Repository{}, "", fmt.Errorf("expected at most 2 arguments, got %d", len(args))
	}
}

func newCmdRepoContentList(f *cmdutil.Factory) *cobra.Command {
	opts := &contentOptions{}

	cmd := &cobra.Command{
		Use:   "list [<owner>/<repo>] [<path>]",
		Short: "List a repository directory",
		Long: "List a repository directory without modifying it. With no arguments, the repository is inferred and its root is listed. " +
			"A single argument is a path in the inferred repository; use OWNER/REPO . to list the root of an explicit repository.",
		Example: `  ag repo content list
  ag repo content list docs
  ag repo content list owner/repo .
  ag repo content list owner/repo docs/guides --ref v1.0.0 --json`,
		Args: cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, path, err := resolveContentListArgs(f, args)
			if err != nil {
				return err
			}
			if err := validateContentPath(path, true); err != nil {
				return err
			}
			ref, err := validateContentRef(cmd, opts.ref)
			if err != nil {
				return err
			}

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}
			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			response, err := api.GetRepositoryContents(client, repository.Owner, repository.Name, path, ref)
			if err != nil {
				return err
			}
			if response.File != nil {
				return contentTypeMismatchError(repository, path, ref, "file", "view")
			}

			out := cmd.OutOrStdout()
			if opts.json {
				return cmdutil.WriteJSON(out, response.Raw)
			}
			for _, entry := range response.Entries {
				if _, err := fmt.Fprintf(
					out,
					"%s\t%s\t%s\n",
					cmdutil.EscapeTSVField(entry.Type),
					cmdutil.EscapeTSVField(entry.Path),
					cmdutil.EscapeTSVField(entry.SHA),
				); err != nil {
					return fmt.Errorf("write directory entry: %w", err)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.ref, "ref", "", "Branch, tag, or commit identifier")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output the complete API directory array as JSON")
	return cmd
}

func newCmdRepoReadFile(f *cmdutil.Factory) *cobra.Command {
	opts := &readFileOptions{}

	cmd := &cobra.Command{
		Use:   "read-file [<owner>/<repo>] <path>",
		Short: "Read a file from a repository",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			path := remaining[0]
			if err := validateContentPath(path, false); err != nil {
				return err
			}

			if cmd.Flags().Changed("ref") && strings.TrimSpace(opts.ref) == "" {
				return errors.New("--ref cannot be empty")
			}
			ref := strings.TrimSpace(opts.ref)

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			content, err := api.GetRepositoryContent(client, repo.Owner, repo.Name, path, ref)
			if err != nil {
				return err
			}

			if content.Type != "file" {
				return fmt.Errorf("path '%s' is a %s, not a file", path, content.Type)
			}
			if content.Encoding != "base64" {
				return fmt.Errorf("unsupported encoding '%s'; expected 'base64'", content.Encoding)
			}
			if !content.ContentPresent {
				return errors.New("empty file content")
			}

			decoded, err := base64.StdEncoding.DecodeString(content.Content)
			if err != nil {
				return fmt.Errorf("malformed base64 content: %w", err)
			}

			out := cmd.OutOrStdout()
			if opts.json {
				return cmdutil.WriteJSON(out, contentFileJSON{
					Name:     content.Name,
					Path:     content.Path,
					SHA:      content.SHA,
					Size:     content.Size,
					Encoding: content.Encoding,
					Content:  content.Content,
					Ref:      ref,
				})
			}

			_, err = fmt.Fprint(out, string(decoded))
			return err
		},
	}

	cmd.Flags().StringVar(&opts.ref, "ref", "", "Branch, tag, or commit identifier")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output file content as JSON")

	return cmd
}

func newCmdRepoReadDir(f *cmdutil.Factory) *cobra.Command {
	opts := &readDirOptions{}

	cmd := &cobra.Command{
		Use:   "read-dir [<owner>/<repo>] <path>",
		Short: "List contents of a repository directory",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, remaining, err := cmdutil.ResolveRepositoryFromArgs(f, args, 1)
			if err != nil {
				return err
			}
			path := remaining[0]
			if err := validateContentPath(path, true); err != nil {
				return err
			}

			if cmd.Flags().Changed("ref") && strings.TrimSpace(opts.ref) == "" {
				return errors.New("--ref cannot be empty")
			}
			ref := strings.TrimSpace(opts.ref)

			token, err := f.Config.GetToken()
			if err != nil {
				return cmdutil.AuthenticationError(err)
			}

			client, err := f.NewAPIClient(token)
			if err != nil {
				return err
			}

			entries, err := api.ListRepositoryContent(client, repo.Owner, repo.Name, path, ref)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if opts.json {
				jsonEntries := make([]contentDirEntryJSON, len(entries))
				for i, e := range entries {
					jsonEntries[i] = contentDirEntryJSON{
						Name: e.Name,
						Path: e.Path,
						Type: e.Type,
						SHA:  e.SHA,
						Size: e.Size,
					}
				}
				return cmdutil.WriteJSON(out, jsonEntries)
			}

			for _, e := range entries {
				if _, err := fmt.Fprintf(out, "%s\t%d\t%s\n", cmdutil.EscapeTSVField(e.Type), e.Size, cmdutil.EscapeTSVField(e.Path)); err != nil {
					return fmt.Errorf("write directory entry: %w", err)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.ref, "ref", "", "Branch, tag, or commit identifier")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output directory entries as JSON")

	return cmd
}
