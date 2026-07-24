package cmdutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PreflightDownloadDestination validates a download destination before any
// network request. The check runs first so invalid paths and existing
// destinations fail without network I/O; WriteDownload repeats the validation
// because filesystem state can change before local writing.
func PreflightDownloadDestination(destination string, overwrite bool) (string, error) {
	return ValidateDownloadDestination(destination, overwrite)
}

// WriteDownload streams source into a temporary file created next to
// destination, syncs it, then installs it via InstallDownload. A failure
// during the streaming copy, sync, or close removes the temporary file and
// leaves the existing destination untouched; once the body is fully written,
// InstallDownload owns the install. destination is returned cleaned.
func WriteDownload(destination string, source io.Reader, overwrite bool) (string, error) {
	destination, err := ValidateDownloadDestination(destination, overwrite)
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

	if err := InstallDownload(temporaryName, destination, overwrite); err != nil {
		return "", err
	}
	return destination, nil
}

// InstallDownload moves temporaryName to destination. When overwrite is false
// a hard link installs the complete file atomically on the same filesystem and
// falls back to an exclusive copy on filesystems without hard-link support;
// neither path replaces an existing destination. When overwrite is true the
// install uses os.Rename on the same filesystem. If replacement fails, the
// existing destination is left in place; this helper never removes it first.
func InstallDownload(temporaryName, destination string, overwrite bool) error {
	return installDownload(temporaryName, destination, overwrite, os.Link, os.Rename)
}

func installDownload(
	temporaryName, destination string,
	overwrite bool,
	link func(string, string) error,
	rename func(string, string) error,
) error {
	if !overwrite {
		linkErr := link(temporaryName, destination)
		if linkErr == nil {
			return nil
		}
		if os.IsExist(linkErr) {
			return fmt.Errorf("destination %s already exists; use --overwrite to replace it", destination)
		}
		if err := copyDownloadNoReplace(temporaryName, destination); err != nil {
			if os.IsExist(err) {
				return fmt.Errorf("destination %s already exists; use --overwrite to replace it", destination)
			}
			return fmt.Errorf("install download at %s without replacing it after hard-link failure (%v): %w", destination, linkErr, err)
		}
		return nil
	}

	if err := rename(temporaryName, destination); err != nil {
		return fmt.Errorf("install download at %s: %w", destination, err)
	}
	return nil
}

func copyDownloadNoReplace(temporaryName, destination string) (err error) {
	source, err := os.Open(temporaryName)
	if err != nil {
		return fmt.Errorf("open completed download: %w", err)
	}
	defer source.Close()

	info, err := source.Stat()
	if err != nil {
		return fmt.Errorf("stat completed download: %w", err)
	}
	destinationFile, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	createdInfo, err := destinationFile.Stat()
	if err != nil {
		_ = destinationFile.Close()
		return fmt.Errorf("stat installed download: %w", err)
	}
	installed := false
	defer func() {
		if !installed {
			currentInfo, statErr := os.Stat(destination)
			if statErr == nil && os.SameFile(createdInfo, currentInfo) {
				_ = os.Remove(destination)
			}
		}
	}()

	if _, err := io.Copy(destinationFile, source); err != nil {
		_ = destinationFile.Close()
		return fmt.Errorf("copy completed download: %w", err)
	}
	if err := destinationFile.Sync(); err != nil {
		_ = destinationFile.Close()
		return fmt.Errorf("sync installed download: %w", err)
	}
	if err := destinationFile.Close(); err != nil {
		return fmt.Errorf("close installed download: %w", err)
	}
	installed = true
	return nil
}

// ValidateDownloadDestination cleans destination and rejects empty paths,
// directories, and existing files unless overwrite is set. It also verifies
// the destination directory exists and is a directory.
func ValidateDownloadDestination(destination string, overwrite bool) (string, error) {
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
