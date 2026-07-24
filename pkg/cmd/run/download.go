package run

import (
	"io"
	"path"
	"strings"
	"unicode"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

// preflightDownloadDestination validates a download destination before any
// network request. The shared cmdutil.PreflightDownloadDestination performs
// the path/existence checks; this wrapper is retained so callers stay close
// to the run command's vocabulary.
func preflightDownloadDestination(destination string, overwrite bool) (string, error) {
	return cmdutil.PreflightDownloadDestination(destination, overwrite)
}

// writeDownload streams source into a temporary file next to destination then
// atomically installs it via the shared cmdutil.WriteDownload helper.
func writeDownload(destination string, source io.Reader, overwrite bool) (string, error) {
	return cmdutil.WriteDownload(destination, source, overwrite)
}

func artifactFilename(artifact actions.Artifact) string {
	name := strings.TrimSpace(artifact.Name)
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	if name == "" || name == "." || name == "/" {
		name = "artifact-" + artifact.ID
	}

	name = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsControl(r):
			return '_'
		case strings.ContainsRune(`<>:"/\\|?*`, r):
			return '_'
		default:
			return r
		}
	}, name)
	name = strings.Trim(name, " .")
	if name == "" {
		name = "artifact-" + artifact.ID
	}
	if !strings.HasSuffix(strings.ToLower(name), ".zip") {
		name += ".zip"
	}
	return name
}
