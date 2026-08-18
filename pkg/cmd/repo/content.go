package repo

import (
	"encoding/base64"
	"errors"
	"fmt"
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
				return fmt.Errorf("not authenticated. Please check your token file: %w", err)
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
				return fmt.Errorf("not authenticated. Please check your token file: %w", err)
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
