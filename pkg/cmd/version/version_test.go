package version

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
)

func setVersionMetadata(t *testing.T, source string) {
	t.Helper()
	oldV, oldC, oldB := version.Version, version.Commit, version.BuildDate
	oldSource := version.Source
	version.Version = "v1.2.3"
	version.Commit = "abc1234"
	version.BuildDate = "2026-07-24T00:00:00Z"
	version.Source = source
	t.Cleanup(func() {
		version.Version = oldV
		version.Commit = oldC
		version.BuildDate = oldB
		version.Source = oldSource
	})
}

func TestNewCmdVersion_TextProfileAndInvalidMatrix(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantSelfUpdate bool
		wantSource     string
	}{
		{name: "source", source: "source", wantSelfUpdate: true, wantSource: "source"},
		{name: "release", source: "release", wantSelfUpdate: true, wantSource: "release"},
		{name: "development", source: "development", wantSelfUpdate: true, wantSource: "development"},
		{name: "npm", source: "npm", wantSource: "npm"},
		{name: "homebrew", source: "homebrew", wantSource: "homebrew"},
		{name: "winget", source: "winget", wantSource: "winget"},
		{name: "scoop", source: "scoop", wantSource: "scoop"},
		{name: "nix", source: "nix", wantSource: "nix"},
		{name: "extension source", source: "corp-repo", wantSource: "corp-repo"},
		{name: "invalid source", source: "unknown", wantSource: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setVersionMetadata(t, tt.source)

			var buf bytes.Buffer
			cmd := NewCmdVersion()
			cmd.SetOut(&buf)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			want := fmt.Sprintf(
				"ag version v1.2.3 (commit: abc1234, built: 2026-07-24T00:00:00Z, self-update: %t, source: %s)\n",
				tt.wantSelfUpdate,
				tt.wantSource,
			)
			if got := buf.String(); got != want {
				t.Errorf("output = %q, want %q", got, want)
			}
		})
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
				SelfUpdate: true, Source: "release",
			},
			want: "ag version v1.0.0 (commit: abc1234, built: 2026-07-15T00:00:00Z, self-update: true, source: release)\n",
		},
		{
			name: "commit only",
			info: version.Info{
				Version: "v1.0.0", Commit: "abc1234", BuildDate: "unknown",
				Source: "npm",
			},
			want: "ag version v1.0.0 (commit: abc1234, self-update: false, source: npm)\n",
		},
		{
			name: "build date only",
			info: version.Info{
				Version: "v1.0.0", Commit: "unknown", BuildDate: "2026-07-15T00:00:00Z",
				SelfUpdate: true, Source: "source",
			},
			want: "ag version v1.0.0 (built: 2026-07-15T00:00:00Z, self-update: true, source: source)\n",
		},
		{
			name: "no optional metadata",
			info: version.Info{
				Version: "v1.0.0", Commit: " UNKNOWN ", BuildDate: "",
				Source: "unknown",
			},
			want: "ag version v1.0.0 (self-update: false, source: unknown)\n",
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
	setVersionMetadata(t, "release")

	want := "ag version v1.2.3 (commit: abc1234, built: 2026-07-24T00:00:00Z, self-update: true, source: release)\n"
	if got := Text(); got != want {
		t.Errorf("Text() = %q, want %q", got, want)
	}
}

func TestNewCmdVersion_JSONPreservesFieldsAndAddsPolicy(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantSelfUpdate bool
		wantSource     string
	}{
		{name: "source", source: "source", wantSelfUpdate: true, wantSource: "source"},
		{name: "release", source: "release", wantSelfUpdate: true, wantSource: "release"},
		{name: "development", source: "development", wantSelfUpdate: true, wantSource: "development"},
		{name: "npm", source: "npm", wantSource: "npm"},
		{name: "homebrew", source: "homebrew", wantSource: "homebrew"},
		{name: "winget", source: "winget", wantSource: "winget"},
		{name: "scoop", source: "scoop", wantSource: "scoop"},
		{name: "nix", source: "nix", wantSource: "nix"},
		{name: "extension source", source: "corp-repo", wantSource: "corp-repo"},
		{name: "invalid source", source: "unknown", wantSource: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setVersionMetadata(t, tt.source)

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
				"version":    `"v1.2.3"`,
				"commit":     `"abc1234"`,
				"buildDate":  `"2026-07-24T00:00:00Z"`,
				"selfUpdate": map[bool]string{true: "true", false: "false"}[tt.wantSelfUpdate],
				"source":     `"` + tt.wantSource + `"`,
			}
			for name, want := range wantFields {
				if got, ok := fields[name]; !ok || string(got) != want {
					t.Errorf("field %q = %s, want %s", name, got, want)
				}
			}
		})
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
