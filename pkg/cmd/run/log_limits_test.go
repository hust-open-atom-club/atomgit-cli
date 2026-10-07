package run

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"
)

func testJobLogLimits(response, entry, total int64) jobLogLimits {
	return jobLogLimits{response: response, entry: entry, total: total}
}

func storedJobLogArchive(t *testing.T, content string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entry, err := archive.CreateHeader(&zip.FileHeader{Name: "job.log", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, content); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestCopyWithLimit(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		limit     int64
		want      string
		wantError error
	}{
		{name: "empty input at zero limit", input: "", limit: 0, want: ""},
		{name: "exact limit", input: "abcd", limit: 4, want: "abcd"},
		{name: "over limit does not copy extra byte", input: "abcde", limit: 4, want: "abcd", wantError: errJobLogLimitExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			_, err := copyWithLimit(&out, strings.NewReader(tt.input), tt.limit)
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("error = %v, want %v", err, tt.wantError)
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestWriteJobLogOutputLimitsRawResponse(t *testing.T) {
	limits := testJobLogLimits(4, 8, 8)
	for _, tt := range []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{name: "exact limit", input: "abcd", want: "abcd"},
		{name: "over limit", input: "abcde", wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := writeJobLogOutputWithLimits(&out, strings.NewReader(tt.input), limits)
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, wantError = %t", err, tt.wantError)
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestWriteJobLogOutputLimitsZipEntries(t *testing.T) {
	for _, tt := range []struct {
		name       string
		entries    []string
		want       string
		wantOutput string
		wantError  string
		total      int64
	}{
		{name: "single entry exact limit", entries: []string{"12345"}, want: "12345"},
		{name: "single entry exceeds limit", entries: []string{"123456"}, wantError: "exceeds 5-byte limit"},
		{name: "multiple entries reach cumulative limit", entries: []string{"12345", "67890"}, want: "12345\n67890", total: 10},
		{name: "multiple entries exceed cumulative limit", entries: []string{"12345", "67890"}, wantError: "job log entries exceed 9-byte total limit", wantOutput: "12345", total: 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			totalLimit := tt.total
			if totalLimit == 0 {
				totalLimit = 10
			}
			err := writeJobLogOutputWithLimits(&out, strings.NewReader(jobLogArchive(t, tt.entries...)), testJobLogLimits(1<<20, 5, totalLimit))
			if tt.wantError == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantError != "" && (err == nil || !strings.Contains(err.Error(), tt.wantError)) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
			wantOutput := tt.wantOutput
			if tt.wantError == "" {
				wantOutput = tt.want
			}
			if out.String() != wantOutput {
				t.Fatalf("output = %q, want %q", out.String(), wantOutput)
			}
		})
	}
}

func TestCopyWithLimitBoundsHighCompressionZipEntry(t *testing.T) {
	content := strings.Repeat("repetitive workflow log line\n", 4096)
	body := jobLogArchive(t, content)
	if len(body)*16 >= len(content) {
		t.Fatalf("fixture compression ratio is too low: archive=%d, entry=%d", len(body), len(content))
	}

	const entryLimit = 1024
	archive, err := zip.NewReader(bytes.NewReader([]byte(body)), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := archive.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	_, copyErr := copyWithLimit(&out, entry, entryLimit)
	closeErr := entry.Close()
	if !errors.Is(copyErr, errJobLogLimitExceeded) {
		t.Fatalf("error = %v, want decompressed entry limit error", copyErr)
	}
	if closeErr != nil {
		t.Fatalf("close ZIP entry: %v", closeErr)
	}
	if out.Len() != entryLimit {
		t.Fatalf("output bytes = %d, want exactly the %d-byte limit", out.Len(), entryLimit)
	}
}

func TestWriteJobLogOutputRejectsZIPHeaderClaimOverEntryLimit(t *testing.T) {
	body := storedJobLogArchive(t, "hello")
	centralDirectory := bytes.Index(body, []byte("PK\x01\x02"))
	if centralDirectory < 0 {
		t.Fatal("central directory not found")
	}
	binary.LittleEndian.PutUint32(body[centralDirectory+24:centralDirectory+28], 6)

	var out bytes.Buffer
	err := writeJobLogOutputWithLimits(&out, bytes.NewReader(body), testJobLogLimits(1<<20, 5, 10))
	if err == nil || !strings.Contains(err.Error(), "entry") || !strings.Contains(err.Error(), "5-byte limit") {
		t.Fatalf("error = %v, want entry limit error", err)
	}
	if out.Len() != 0 {
		t.Fatalf("output = %q, want no output", out.String())
	}
}

func TestWriteZipJobLogsRejectsZIP64SizeOverflow(t *testing.T) {
	file := &zip.File{FileHeader: zip.FileHeader{Name: "oversized.log", UncompressedSize64: ^uint64(0)}}
	err := writeZipJobLogs(io.Discard, []*zip.File{file}, testJobLogLimits(100, 100, 100))
	if err == nil || !strings.Contains(err.Error(), "100-byte limit") {
		t.Fatalf("error = %v, want bounded-size error", err)
	}
}

func TestWriteJobLogOutputRejectsMalformedZIPButKeepsPlainText(t *testing.T) {
	for _, tt := range []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{name: "plain text", input: "ordinary log", want: "ordinary log"},
		{name: "truncated ZIP local header", input: "PK\x03\x04broken", wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := writeJobLogOutputWithLimits(&out, strings.NewReader(tt.input), testJobLogLimits(100, 100, 100))
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, wantError = %t", err, tt.wantError)
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestWriteJobLogOutputPreservesZIPChecksumError(t *testing.T) {
	body := storedJobLogArchive(t, "hello")
	content := bytes.Index(body, []byte("hello"))
	if content < 0 {
		t.Fatal("entry content not found")
	}
	body[content] ^= 1

	var out bytes.Buffer
	err := writeJobLogOutputWithLimits(&out, bytes.NewReader(body), testJobLogLimits(1<<20, 100, 100))
	if !errors.Is(err, zip.ErrChecksum) {
		t.Fatalf("error = %v, want ZIP checksum error", err)
	}
}

type testLogWriter struct {
	count int
	err   error
}

func (w testLogWriter) Write([]byte) (int, error) { return w.count, w.err }

type testLogReader struct{ err error }

func (r testLogReader) Read([]byte) (int, error) { return 0, r.err }

func TestLastByteWriterValidatesCountsAndShortWrites(t *testing.T) {
	writeErr := errors.New("write failed")
	for _, tt := range []struct {
		name     string
		count    int
		writeErr error
		wantErr  error
		wantLast byte
	}{
		{name: "normal", count: 3, wantLast: 'c'},
		{name: "short write", count: 2, wantErr: io.ErrShortWrite, wantLast: 'b'},
		{name: "partial write and error", count: 2, writeErr: writeErr, wantErr: writeErr, wantLast: 'b'},
		{name: "negative count", count: -1, wantErr: errInvalidWriteCount},
		{name: "count exceeds input", count: 4, wantErr: errInvalidWriteCount},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writer := &lastByteWriter{writer: testLogWriter{count: tt.count, err: tt.writeErr}}
			n, err := writer.Write([]byte("abc"))
			if errors.Is(tt.wantErr, errInvalidWriteCount) {
				if !errors.Is(err, errInvalidWriteCount) {
					t.Fatalf("error = %v, want invalid count error", err)
				}
				if n != 0 {
					t.Fatalf("written = %d, want 0", n)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantLast != 0 && (writer.last != tt.wantLast || !writer.wrote) {
				t.Fatalf("last = %q, wrote = %t", writer.last, writer.wrote)
			}
		})
	}
}

func TestLastByteWriterAcceptsEmptyInput(t *testing.T) {
	writer := &lastByteWriter{writer: testLogWriter{count: 0}}
	written, err := writer.Write(nil)
	if err != nil || written != 0 {
		t.Fatalf("Write(nil) = %d, %v; want 0, nil", written, err)
	}
	if writer.wrote {
		t.Fatal("empty write marked output as written")
	}
}

func TestWriteJobLogOutputPreservesReadAndWriteErrors(t *testing.T) {
	readErr := errors.New("source read failed")
	if err := writeJobLogOutputWithLimits(io.Discard, testLogReader{err: readErr}, testJobLogLimits(10, 10, 10)); !errors.Is(err, readErr) {
		t.Fatalf("read error = %v, want %v", err, readErr)
	}

	writeErr := errors.New("output write failed")
	if err := writeJobLogOutputWithLimits(testLogWriter{err: writeErr}, strings.NewReader("plain text"), testJobLogLimits(100, 100, 100)); !errors.Is(err, writeErr) {
		t.Fatalf("write error = %v, want %v", err, writeErr)
	}
}
