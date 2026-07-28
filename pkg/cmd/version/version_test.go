package version

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
)

func setVersionMetadata(t *testing.T) {
	t.Helper()
	oldV, oldC, oldB := version.Version, version.Commit, version.BuildDate
	version.Version = "v1.2.3"
	version.Commit = "abc1234"
	version.BuildDate = "2026-07-24T00:00:00Z"
	t.Cleanup(func() {
		version.Version = oldV
		version.Commit = oldC
		version.BuildDate = oldB
	})
}

func TestNewCmdVersion_Text(t *testing.T) {
	setVersionMetadata(t)

	var buf bytes.Buffer
	cmd := NewCmdVersion()
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	want := "ag version v1.2.3 (commit: abc1234, built: 2026-07-24T00:00:00Z)\n"
	if got := buf.String(); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestFormatTextOmitsUnknownMetadata(t *testing.T) {
	tests := []struct {
		name string
		info version.Info
		want string
	}{
		{
			name: "all metadata",
			info: version.Info{
				Version: "v1.0.0", Commit: "abc1234", BuildDate: "2026-07-15T00:00:00Z",
			},
			want: "ag version v1.0.0 (commit: abc1234, built: 2026-07-15T00:00:00Z)\n",
		},
		{
			name: "commit only",
			info: version.Info{
				Version: "v1.0.0", Commit: "abc1234", BuildDate: "unknown",
			},
			want: "ag version v1.0.0 (commit: abc1234)\n",
		},
		{
			name: "build date only",
			info: version.Info{
				Version: "v1.0.0", Commit: "unknown", BuildDate: "2026-07-15T00:00:00Z",
			},
			want: "ag version v1.0.0 (built: 2026-07-15T00:00:00Z)\n",
		},
		{
			name: "no optional metadata",
			info: version.Info{
				Version: "v1.0.0", Commit: " UNKNOWN ", BuildDate: "",
			},
			want: "ag version v1.0.0\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatText(tt.info); got != tt.want {
				t.Errorf("formatText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestText_MatchesVersionCLIContract(t *testing.T) {
	setVersionMetadata(t)

	want := "ag version v1.2.3 (commit: abc1234, built: 2026-07-24T00:00:00Z)\n"
	if got := Text(); got != want {
		t.Errorf("Text() = %q, want %q", got, want)
	}
}

func TestNewCmdVersion_JSONPreservesFixedFields(t *testing.T) {
	setVersionMetadata(t)

	var buf bytes.Buffer
	cmd := NewCmdVersion()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &fields); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	wantFields := map[string]string{
		"version":   `"v1.2.3"`,
		"commit":    `"abc1234"`,
		"buildDate": `"2026-07-24T00:00:00Z"`,
	}
	if len(fields) != len(wantFields) {
		t.Fatalf("fields = %v, want only version, commit and buildDate", fields)
	}
	for name, want := range wantFields {
		if got, ok := fields[name]; !ok || string(got) != want {
			t.Errorf("field %q = %s, want %s", name, got, want)
		}
	}
}

func TestNewCmdVersion_InvalidArgs(t *testing.T) {
	var buf bytes.Buffer
	cmd := NewCmdVersion()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"extra-arg"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for extra argument, got nil")
	}
}

func TestNewCmdVersion_NoCredentialsRequired(t *testing.T) {
	// The version command must succeed without any configuration or network.
	// Since it only reads embedded metadata, this is guaranteed by the code
	// structure: NewCmdVersion takes no Factory and reads only linker vars.
	var buf bytes.Buffer
	cmd := NewCmdVersion()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output, got empty buffer")
	}
}

func TestNewCmdVersion_Help(t *testing.T) {
	var buf bytes.Buffer
	cmd := NewCmdVersion()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"Show version information",
		"Usage:",
		"version [flags]",
		"--json",
		"Output version information as JSON",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("help output does not contain %q:\n%s", want, out)
		}
	}
}
