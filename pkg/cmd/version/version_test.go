package version

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
)

func TestNewCmdVersion_Text(t *testing.T) {
	oldV, oldC, oldB := version.Version, version.Commit, version.BuildDate
	version.Version = "v1.0.0"
	version.Commit = "abc1234"
	version.BuildDate = "2026-07-12"
	defer func() {
		version.Version = oldV
		version.Commit = oldC
		version.BuildDate = oldB
	}()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "release text output",
			args: []string{},
			want: "ag version v1.0.0 (commit: abc1234, built: 2026-07-12)\n",
		},
		{
			name: "dev text output",
			args: []string{},
			want: "ag version dev (commit: unknown, built: unknown)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "dev text output" {
				version.Version = "dev"
				version.Commit = "unknown"
				version.BuildDate = "unknown"
			} else {
				version.Version = "v1.0.0"
				version.Commit = "abc1234"
				version.BuildDate = "2026-07-12"
			}

			var buf bytes.Buffer
			cmd := NewCmdVersion()
			cmd.SetOut(&buf)
			cmd.SetArgs(tt.args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewCmdVersion_JSON(t *testing.T) {
	oldV, oldC, oldB := version.Version, version.Commit, version.BuildDate
	version.Version = "v2.0.0"
	version.Commit = "deadbeef"
	version.BuildDate = "2026-01-01T00:00:00Z"
	defer func() {
		version.Version = oldV
		version.Commit = oldC
		version.BuildDate = oldB
	}()

	var buf bytes.Buffer
	cmd := NewCmdVersion()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var info version.Info
	if err := json.Unmarshal(buf.Bytes(), &info); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if info.Version != "v2.0.0" {
		t.Errorf("Version = %q, want %q", info.Version, "v2.0.0")
	}
	if info.Commit != "deadbeef" {
		t.Errorf("Commit = %q, want %q", info.Commit, "deadbeef")
	}
	if info.BuildDate != "2026-01-01T00:00:00Z" {
		t.Errorf("BuildDate = %q, want %q", info.BuildDate, "2026-01-01T00:00:00Z")
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
	if !strings.Contains(out, "version") {
		t.Errorf("help output does not mention version: %s", out)
	}
}
