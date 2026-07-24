package cmdutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failingReader struct {
	read bool
}

func (r *failingReader) Read(p []byte) (int, error) {
	if !r.read {
		r.read = true
		copy(p, "partial")
		return len("partial"), nil
	}
	return 0, errors.New("read failed")
}

// TestWriteDownloadCleansTemporaryFileAfterFailure verifies that a transport
// failure during the body copy removes the temporary file and leaves any
// existing destination untouched.
func TestWriteDownloadCleansTemporaryFileAfterFailure(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "download.zip")
	if _, err := WriteDownload(destination, &failingReader{}, false); err == nil || !strings.Contains(err.Error(), "read failed") {
		t.Fatalf("error = %v", err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("destination exists after failure: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(directory, ".download.zip.tmp-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files = %v, %v", matches, err)
	}
}

// TestInstallDownloadNoReplaceIsAtomic launches two concurrent installs of
// different temporary files onto the same destination without overwrite.
// Exactly one must succeed and the other must report "already exists"; the
// destination must contain one of the two complete payloads.
func TestInstallDownloadNoReplaceIsAtomic(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "download.zip")
	temporaryFiles := []string{
		filepath.Join(directory, "first.tmp"),
		filepath.Join(directory, "second.tmp"),
	}
	contents := []string{"first payload", "second payload"}
	for index, filename := range temporaryFiles {
		if err := os.WriteFile(filename, []byte(contents[index]), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	start := make(chan struct{})
	results := make(chan error, len(temporaryFiles))
	for _, filename := range temporaryFiles {
		go func(temporaryName string) {
			<-start
			results <- InstallDownload(temporaryName, destination, false)
		}(filename)
	}
	close(start)

	successes := 0
	alreadyExists := 0
	for range temporaryFiles {
		err := <-results
		switch {
		case err == nil:
			successes++
		case strings.Contains(err.Error(), "already exists"):
			alreadyExists++
		default:
			t.Fatalf("unexpected install error: %v", err)
		}
	}
	if successes != 1 || alreadyExists != 1 {
		t.Fatalf("successes = %d, already-exists errors = %d", successes, alreadyExists)
	}

	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != contents[0] && string(data) != contents[1] {
		t.Fatalf("destination contains partial or unexpected data: %q", data)
	}
}

// TestWriteDownloadOverwritePreservesOldOnReadFailure verifies that with
// overwrite=true, if the body copy fails mid-stream, the pre-existing
// destination keeps its old contents and no temporary file is left behind.
// WriteDownload must only install the temporary file after the full copy
// succeeds, so a transport failure during streaming never replaces the old
// destination.
func TestWriteDownloadOverwritePreservesOldOnReadFailure(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "download.zip")
	const old = "old-payload"
	if err := os.WriteFile(destination, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := WriteDownload(destination, &failingReader{}, true); err == nil || !strings.Contains(err.Error(), "read failed") {
		t.Fatalf("error = %v", err)
	}
	if data, err := os.ReadFile(destination); err != nil || string(data) != old {
		t.Fatalf("destination = %q, %v; want %q", data, err, old)
	}
	matches, err := filepath.Glob(filepath.Join(directory, ".download.zip.tmp-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files = %v, %v", matches, err)
	}
}

func TestInstallDownloadOverwritePreservesOldOnRenameFailure(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "download.zip")
	temporary := filepath.Join(directory, "download.tmp")
	if err := os.WriteFile(destination, []byte("old-payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(temporary, []byte("new-payload"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := installDownload(temporary, destination, true, os.Link, func(string, string) error {
		return errors.New("rename denied")
	})
	if err == nil || !strings.Contains(err.Error(), "rename denied") {
		t.Fatalf("error = %v, want rename failure", err)
	}
	if data, readErr := os.ReadFile(destination); readErr != nil || string(data) != "old-payload" {
		t.Fatalf("destination = %q, %v; want old payload", data, readErr)
	}
	if data, readErr := os.ReadFile(temporary); readErr != nil || string(data) != "new-payload" {
		t.Fatalf("temporary = %q, %v; want new payload", data, readErr)
	}
}

func TestInstallDownloadFallsBackWhenHardLinksAreUnsupported(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "download.zip")
	temporary := filepath.Join(directory, "download.tmp")
	if err := os.WriteFile(temporary, []byte("complete payload"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := installDownload(temporary, destination, false, func(string, string) error {
		return errors.New("hard links unsupported")
	}, os.Rename)
	if err != nil {
		t.Fatalf("installDownload: %v", err)
	}
	if data, readErr := os.ReadFile(destination); readErr != nil || string(data) != "complete payload" {
		t.Fatalf("destination = %q, %v; want complete payload", data, readErr)
	}
	if data, readErr := os.ReadFile(temporary); readErr != nil || string(data) != "complete payload" {
		t.Fatalf("temporary = %q, %v; want source retained", data, readErr)
	}
}

func TestInstallDownloadFallbackNeverReplacesExistingDestination(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "download.zip")
	temporary := filepath.Join(directory, "download.tmp")
	if err := os.WriteFile(temporary, []byte("new payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("old payload"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := installDownload(temporary, destination, false, func(string, string) error {
		return errors.New("hard links unsupported")
	}, os.Rename)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v, want already-exists error", err)
	}
	if data, readErr := os.ReadFile(destination); readErr != nil || string(data) != "old payload" {
		t.Fatalf("destination = %q, %v; want old payload", data, readErr)
	}
}
