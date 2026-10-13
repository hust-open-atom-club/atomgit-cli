package run

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
)

// Job log display buffers the archive before reading it and can expand each
// entry during ZIP decompression. Keep independent bounds for both stages.
const (
	// Bound temporary disk usage while allowing individual logs to expand.
	maxJobLogResponseBytes int64 = 64 << 20
	// Allow a log entry to expand to twice the maximum archived response.
	maxJobLogEntryBytes int64 = 128 << 20
	// Permit two maximum-sized entries while bounding cumulative expansion.
	maxJobLogTotalBytes int64 = 256 << 20
)

var errJobLogLimitExceeded = errors.New("job log size limit exceeded")
var errInvalidWriteCount = errors.New("invalid write count")

type jobLogLimits struct {
	response int64
	entry    int64
	total    int64
}

func defaultJobLogLimits() jobLogLimits {
	return jobLogLimits{
		response: maxJobLogResponseBytes,
		entry:    maxJobLogEntryBytes,
		total:    maxJobLogTotalBytes,
	}
}

// writeJobLogOutput turns AtomGit's ZIP response into readable log text. The
// response is first streamed to a temporary file because archive/zip requires
// random access. Plain-text responses remain supported for compatibility.
func writeJobLogOutput(out io.Writer, source io.Reader) error {
	return writeJobLogOutputWithLimits(out, source, defaultJobLogLimits())
}

func writeJobLogOutputWithLimits(out io.Writer, source io.Reader, limits jobLogLimits) (resultErr error) {
	if limits.response < 0 || limits.entry < 0 || limits.total < 0 {
		return fmt.Errorf("job log limits must not be negative")
	}
	temporary, err := os.CreateTemp("", "ag-job-log-*")
	if err != nil {
		return fmt.Errorf("create temporary job log: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() {
		if err := os.Remove(temporaryName); err != nil && !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, fmt.Errorf("remove temporary job log: %w", err))
		}
	}()

	if _, err := copyWithLimit(temporary, source, limits.response); err != nil {
		writeErr := fmt.Errorf("write temporary job log: %w", err)
		if errors.Is(err, errJobLogLimitExceeded) {
			writeErr = fmt.Errorf("job log response exceeds %d-byte limit: %w", limits.response, writeErr)
		}
		if closeErr := temporary.Close(); closeErr != nil {
			return errors.Join(writeErr, fmt.Errorf("close temporary job log: %w", closeErr))
		}
		return writeErr
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary job log: %w", err)
	}

	archive, err := zip.OpenReader(temporaryName)
	if err == nil {
		defer archive.Close() //nolint:errcheck // The ZIP archive is read-only; log entry read errors are handled separately and Close only releases resources.
		return writeZipJobLogs(out, archive.File, limits)
	}
	if !errors.Is(err, zip.ErrFormat) {
		return fmt.Errorf("open job log archive: %w", err)
	}
	zipErr := err
	zipSignature, err := hasZipSignature(temporaryName)
	if err != nil {
		return fmt.Errorf("inspect job log response: %w", err)
	}
	if zipSignature {
		return fmt.Errorf("open job log archive: %w", zipErr)
	}

	plain, err := os.Open(temporaryName)
	if err != nil {
		return fmt.Errorf("open temporary job log: %w", err)
	}
	defer plain.Close() //nolint:errcheck // The temporary log is reopened read-only; io.Copy handles read/write errors and Close only releases resources.
	if _, err := io.Copy(out, plain); err != nil {
		return fmt.Errorf("write job log: %w", err)
	}
	return nil
}

func copyWithLimit(dst io.Writer, src io.Reader, limit int64) (int64, error) {
	if limit < 0 {
		return 0, fmt.Errorf("job log limit must not be negative")
	}
	limited := &io.LimitedReader{R: src, N: limit}
	written, err := io.Copy(dst, limited)
	if err != nil {
		return written, err
	}
	if limited.N > 0 {
		return written, nil
	}

	// Probe without copying the extra byte so an exact-limit entry succeeds,
	// while ZIP readers still reach EOF to check the entry checksum.
	extra, err := io.CopyN(io.Discard, src, 1)
	if extra > 0 {
		return written, errJobLogLimitExceeded
	}
	if errors.Is(err, io.EOF) {
		return written, nil
	}
	if err != nil {
		return written, err
	}
	return written, errJobLogLimitExceeded
}

func hasZipSignature(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	var signature [4]byte
	if _, err := io.ReadFull(file, signature[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return false, nil
		}
		return false, err
	}
	return signature[0] == 'P' && signature[1] == 'K' &&
		(signature[2] == 3 && signature[3] == 4 ||
			signature[2] == 5 && signature[3] == 6 ||
			signature[2] == 7 && signature[3] == 8), nil
}

func zipSizeExceedsLimit(size uint64, limit int64) bool {
	if limit < 0 || size > 1<<63-1 {
		return true
	}
	return int64(size) > limit
}

func writeZipJobLogs(out io.Writer, files []*zip.File, limits jobLogLimits) error {
	writer := &lastByteWriter{writer: out}
	writtenFiles := 0
	var totalWritten int64
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		name := singleLine(file.Name)
		if zipSizeExceedsLimit(file.UncompressedSize64, limits.entry) {
			return fmt.Errorf("job log entry %q exceeds %d-byte limit", name, limits.entry)
		}
		remainingTotal := limits.total - totalWritten
		if zipSizeExceedsLimit(file.UncompressedSize64, remainingTotal) {
			return fmt.Errorf("job log entries exceed %d-byte total limit", limits.total)
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
		entryLimit := min(limits.entry, remainingTotal)
		entryScope := "entry"
		if remainingTotal < limits.entry {
			entryScope = "total"
		}
		entryWritten, copyErr := copyWithLimit(writer, entry, entryLimit)
		closeErr := entry.Close()
		if copyErr != nil {
			var writeErr error
			if errors.Is(copyErr, errJobLogLimitExceeded) {
				if entryScope == "entry" {
					writeErr = fmt.Errorf("job log entry %q exceeds %d-byte limit", name, limits.entry)
				} else {
					writeErr = fmt.Errorf("job log entries exceed %d-byte total limit", limits.total)
				}
			} else {
				writeErr = fmt.Errorf("write job log entry %q: %w", name, copyErr)
			}
			if closeErr != nil {
				return errors.Join(writeErr, fmt.Errorf("close job log entry %q: %w", name, closeErr))
			}
			return writeErr
		}
		if closeErr != nil {
			return fmt.Errorf("close job log entry %q: %w", name, closeErr)
		}
		totalWritten += entryWritten
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
	if written < 0 || written > len(data) {
		return 0, errors.Join(fmt.Errorf("%w: %d", errInvalidWriteCount, written), err)
	}
	if written > 0 {
		accepted := data[:written]
		w.last = accepted[len(accepted)-1]
		w.wrote = true
	}
	if err == nil && written < len(data) {
		err = io.ErrShortWrite
	}
	return written, err
}
