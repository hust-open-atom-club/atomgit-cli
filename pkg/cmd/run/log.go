package run

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
)

// writeJobLogOutput turns AtomGit's ZIP response into readable log text. The
// response is first streamed to a temporary file because archive/zip requires
// random access. Plain-text responses remain supported for compatibility.
func writeJobLogOutput(out io.Writer, source io.Reader) error {
	temporary, err := os.CreateTemp("", "ag-job-log-*")
	if err != nil {
		return fmt.Errorf("create temporary job log: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if _, err := io.Copy(temporary, source); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary job log: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary job log: %w", err)
	}

	archive, err := zip.OpenReader(temporaryName)
	if err == nil {
		defer archive.Close()
		return writeZipJobLogs(out, archive.File)
	}
	if !errors.Is(err, zip.ErrFormat) {
		return fmt.Errorf("open job log archive: %w", err)
	}

	plain, err := os.Open(temporaryName)
	if err != nil {
		return fmt.Errorf("open temporary job log: %w", err)
	}
	defer plain.Close()
	if _, err := io.Copy(out, plain); err != nil {
		return fmt.Errorf("write job log: %w", err)
	}
	return nil
}

func writeZipJobLogs(out io.Writer, files []*zip.File) error {
	writer := &lastByteWriter{writer: out}
	writtenFiles := 0
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		if writtenFiles > 0 && writer.wrote && writer.last != '\n' {
			if _, err := writer.Write([]byte{'\n'}); err != nil {
				return fmt.Errorf("separate job log entries: %w", err)
			}
		}

		entry, err := file.Open()
		if err != nil {
			return fmt.Errorf("open job log entry %q: %w", singleLine(file.Name), err)
		}
		_, copyErr := io.Copy(writer, entry)
		closeErr := entry.Close()
		if copyErr != nil {
			return fmt.Errorf("write job log entry %q: %w", singleLine(file.Name), copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close job log entry %q: %w", singleLine(file.Name), closeErr)
		}
		writtenFiles++
	}
	return nil
}

type lastByteWriter struct {
	writer io.Writer
	last   byte
	wrote  bool
}

func (w *lastByteWriter) Write(data []byte) (int, error) {
	written, err := w.writer.Write(data)
	if written > 0 {
		w.last = data[written-1]
		w.wrote = true
	}
	return written, err
}
