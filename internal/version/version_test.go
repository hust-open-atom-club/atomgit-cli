package version

import (
	"runtime/debug"
	"testing"
)

// restoreVars swaps package-level variables to the requested test values and
// returns a cleanup function that restores the originals.
func restoreVars(v, c, b, selfUpdate, source string, fn func() (*debug.BuildInfo, bool)) func() {
	oldV, oldC, oldB := Version, Commit, BuildDate
	oldSelfUpdate, oldSource := SelfUpdate, Source
	oldFn := ReadBuildInfo
	Version = v
	Commit = c
	BuildDate = b
	SelfUpdate = selfUpdate
	Source = source
	ReadBuildInfo = fn
	return func() {
		Version = oldV
		Commit = oldC
		BuildDate = oldB
		SelfUpdate = oldSelfUpdate
		Source = oldSource
		ReadBuildInfo = oldFn
	}
}

func TestGet_DevelopmentDefaults(t *testing.T) {
	cleanup := restoreVars("dev", "unknown", "unknown", "true", "source", func() (*debug.BuildInfo, bool) {
		return nil, false
	})
	defer cleanup()

	info := Get()
	if info.Version != "dev" {
		t.Errorf("Version = %q, want %q", info.Version, "dev")
	}
	if info.Commit != "unknown" {
		t.Errorf("Commit = %q, want %q", info.Commit, "unknown")
	}
	if info.BuildDate != "unknown" {
		t.Errorf("BuildDate = %q, want %q", info.BuildDate, "unknown")
	}
}

func TestGet_BuildInfoFallback(t *testing.T) {
	cleanup := restoreVars("dev", "unknown", "unknown", "true", "source", func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "atomgit.com/hust-open-atom-club/atomgit-cli",
				Version: "v0.5.0",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc1234"},
				{Key: "vcs.time", Value: "2026-07-01T00:00:00Z"},
			},
		}, true
	})
	defer cleanup()

	info := Get()
	if info.Version != "v0.5.0" {
		t.Errorf("Version = %q, want %q", info.Version, "v0.5.0")
	}
	if info.Commit != "abc1234" {
		t.Errorf("Commit = %q, want %q", info.Commit, "abc1234")
	}
	if info.BuildDate != "2026-07-01T00:00:00Z" {
		t.Errorf("BuildDate = %q, want %q", info.BuildDate, "2026-07-01T00:00:00Z")
	}
}

func TestGet_BuildInfoFallbackSkipsDevel(t *testing.T) {
	cleanup := restoreVars("dev", "unknown", "unknown", "true", "source", func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "atomgit.com/hust-open-atom-club/atomgit-cli",
				Version: "(devel)",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc1234"},
			},
		}, true
	})
	defer cleanup()

	info := Get()
	// (devel) is skipped so version stays "dev"
	if info.Version != "dev" {
		t.Errorf("Version = %q, want %q", info.Version, "dev")
	}
	// But VCS settings still enrich other fields
	if info.Commit != "abc1234" {
		t.Errorf("Commit = %q, want %q", info.Commit, "abc1234")
	}
}

func TestGet_DirtyState(t *testing.T) {
	cleanup := restoreVars("dev", "unknown", "unknown", "true", "source", func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "atomgit.com/hust-open-atom-club/atomgit-cli",
				Version: "v0.5.0",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc1234"},
				{Key: "vcs.modified", Value: "true"},
			},
		}, true
	})
	defer cleanup()

	info := Get()
	want := "v0.5.0-dirty"
	if info.Version != want {
		t.Errorf("Version = %q, want %q", info.Version, want)
	}
}

func TestGet_LinkerPrecedence(t *testing.T) {
	// Linker-injected release values: they must be used exclusively even when
	// ReadBuildInfo is also available.
	cleanup := restoreVars("v1.2.3", "feedbeef", "2026-06-15T12:00:00Z", "true", "release", func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "atomgit.com/hust-open-atom-club/atomgit-cli",
				Version: "v9.9.9",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "9999999"},
				{Key: "vcs.time", Value: "2020-01-01T00:00:00Z"},
				{Key: "vcs.modified", Value: "true"},
			},
		}, true
	})
	defer cleanup()

	info := Get()
	if info.Version != "v1.2.3" {
		t.Errorf("Version = %q, want linker value %q", info.Version, "v1.2.3")
	}
	if info.Commit != "feedbeef" {
		t.Errorf("Commit = %q, want linker value %q", info.Commit, "feedbeef")
	}
	if info.BuildDate != "2026-06-15T12:00:00Z" {
		t.Errorf("BuildDate = %q, want linker value %q", info.BuildDate, "2026-06-15T12:00:00Z")
	}
}

func TestGet_BuildInfoNil(t *testing.T) {
	// ReadBuildInfo returns (nil, true): it exists but has no data.
	cleanup := restoreVars("dev", "unknown", "unknown", "true", "source", func() (*debug.BuildInfo, bool) {
		return nil, true
	})
	defer cleanup()

	info := Get()
	if info.Version != "dev" {
		t.Errorf("Version = %q, want %q", info.Version, "dev")
	}
}

func TestGet_BuildInfoEmptyVersion(t *testing.T) {
	// Main.Version is empty string; not "(devel)" but also empty.
	cleanup := restoreVars("dev", "unknown", "unknown", "true", "source", func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "atomgit.com/hust-open-atom-club/atomgit-cli",
				Version: "",
			},
		}, true
	})
	defer cleanup()

	info := Get()
	if info.Version != "dev" {
		t.Errorf("Version = %q, want %q", info.Version, "dev")
	}
}

func TestGet_PartialVCS(t *testing.T) {
	// Only vcs.revision is set; vcs.time is missing.
	cleanup := restoreVars("dev", "unknown", "unknown", "true", "source", func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "atomgit.com/hust-open-atom-club/atomgit-cli",
				Version: "v0.5.0",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc1234"},
			},
		}, true
	})
	defer cleanup()

	info := Get()
	if info.Version != "v0.5.0" {
		t.Errorf("Version = %q, want %q", info.Version, "v0.5.0")
	}
	if info.Commit != "abc1234" {
		t.Errorf("Commit = %q, want %q", info.Commit, "abc1234")
	}
	if info.BuildDate != "unknown" {
		t.Errorf("BuildDate = %q, want %q", info.BuildDate, "unknown")
	}
}

func TestPolicy_ProfileAndInvalidLinkerMatrix(t *testing.T) {
	tests := []struct {
		name           string
		selfUpdate     string
		source         string
		wantState      PolicyState
		wantSource     DistributionSource
		wantSelfUpdate bool
	}{
		{name: "source", selfUpdate: "true", source: "source", wantState: PolicyEnabled, wantSource: SourceSource, wantSelfUpdate: true},
		{name: "release", selfUpdate: "true", source: "release", wantState: PolicyEnabled, wantSource: SourceRelease, wantSelfUpdate: true},
		{name: "development", selfUpdate: "true", source: "development", wantState: PolicyEnabled, wantSource: SourceDevelopment, wantSelfUpdate: true},
		{name: "npm", selfUpdate: "false", source: "npm", wantState: PolicyDisabled, wantSource: SourceNPM},
		{name: "homebrew", selfUpdate: "false", source: "homebrew", wantState: PolicyDisabled, wantSource: SourceHomebrew},
		{name: "winget", selfUpdate: "false", source: "winget", wantState: PolicyDisabled, wantSource: SourceWinget},
		{name: "nix", selfUpdate: "false", source: "nix", wantState: PolicyDisabled, wantSource: SourceNix},
		{name: "extension source", selfUpdate: "false", source: "corp_repo-1.2", wantState: PolicyDisabled, wantSource: "corp_repo-1.2"},
		{name: "empty boolean", selfUpdate: "", source: "release", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "uppercase boolean", selfUpdate: "TRUE", source: "release", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "whitespace boolean", selfUpdate: " true", source: "release", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "numeric boolean", selfUpdate: "1", source: "release", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "empty source", selfUpdate: "false", source: "", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "reserved unknown", selfUpdate: "false", source: "unknown", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "leading punctuation", selfUpdate: "false", source: "-npm", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "uppercase source", selfUpdate: "false", source: "Homebrew", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "path source", selfUpdate: "false", source: "corp/npm", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "source too long", selfUpdate: "false", source: "a1234567890123456789012345678901234567890123456789012345678901234", wantState: PolicyInvalid, wantSource: SourceUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := restoreVars("v1.0.0", "abcdef0123456789", "2026-07-24T00:00:00Z", tt.selfUpdate, tt.source, func() (*debug.BuildInfo, bool) {
				return nil, false
			})
			defer cleanup()

			policy := Policy()
			if policy.State != tt.wantState {
				t.Fatalf("State = %q, want %q", policy.State, tt.wantState)
			}
			if policy.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", policy.Source, tt.wantSource)
			}
			if policy.SelfUpdate != tt.wantSelfUpdate {
				t.Errorf("SelfUpdate = %v, want %v", policy.SelfUpdate, tt.wantSelfUpdate)
			}

			info := Get()
			if info.Version != "v1.0.0" || info.Commit != "abcdef0123456789" || info.BuildDate != "2026-07-24T00:00:00Z" {
				t.Errorf("Get() changed existing metadata: %+v", info)
			}
			if info.SelfUpdate != tt.wantSelfUpdate || info.Source != string(tt.wantSource) {
				t.Errorf("Get() policy = %v/%q, want %v/%q", info.SelfUpdate, info.Source, tt.wantSelfUpdate, tt.wantSource)
			}
		})
	}
}
