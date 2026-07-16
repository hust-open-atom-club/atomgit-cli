package run

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
)

func writeDownload(destination string, source io.Reader, overwrite bool) (string, error) {
	destination, err := validateDownloadDestination(destination, overwrite)
	if err != nil {
		return "", err
	}

	temporary, err := os.CreateTemp(filepath.Dir(destination), "."+filepath.Base(destination)+".tmp-*")
	if err != nil {
		return "", fmt.Errorf("create temporary download: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() {
		_ = os.Remove(temporaryName)
	}()

	if _, err := io.Copy(temporary, source); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("write temporary download: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("sync temporary download: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close temporary download: %w", err)
	}

	if err := installDownload(temporaryName, destination, overwrite); err != nil {
		return "", err
	}
	return destination, nil
}

func installDownload(temporaryName, destination string, overwrite bool) error {
	if !overwrite {
		// The temporary file is created in the destination directory, so a hard
		// link installs the complete file atomically on the same filesystem and
		// fails without replacing an existing destination.
		if err := os.Link(temporaryName, destination); err != nil {
			if os.IsExist(err) {
				return fmt.Errorf("destination %s already exists; use --overwrite to replace it", destination)
			}
			return fmt.Errorf("install download at %s without replacing it: %w", destination, err)
		}
		return nil
	}

	if err := os.Rename(temporaryName, destination); err != nil {
		if runtime.GOOS != "windows" {
			return fmt.Errorf("install download at %s: %w", destination, err)
		}
		if removeErr := os.Remove(destination); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("replace destination %s: %w", destination, removeErr)
		}
		if renameErr := os.Rename(temporaryName, destination); renameErr != nil {
			return fmt.Errorf("install download at %s: %w", destination, renameErr)
		}
	}
	return nil
}

func validateDownloadDestination(destination string, overwrite bool) (string, error) {
	destination = filepath.Clean(strings.TrimSpace(destination))
	if destination == "." || destination == "" {
		return "", fmt.Errorf("destination path must not be empty")
	}

	if info, err := os.Stat(destination); err == nil {
		if info.IsDir() {
			return "", fmt.Errorf("destination %s is a directory", destination)
		}
		if !overwrite {
			return "", fmt.Errorf("destination %s already exists; use --overwrite to replace it", destination)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("check destination %s: %w", destination, err)
	}

	directory := filepath.Dir(destination)
	info, err := os.Stat(directory)
	if err != nil {
		return "", fmt.Errorf("open destination directory %s: %w", directory, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("destination directory %s is not a directory", directory)
	}
	return destination, nil
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
