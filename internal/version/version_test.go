package version

import (
	"runtime/debug"
	"testing"
)

type linkerMetadata struct {
	version   string
	commit    string
	buildDate string
	source    string
}

// restoreVars swaps package-level variables to the requested test values and
// returns a cleanup function that restores the originals.
func restoreVars(metadata linkerMetadata, fn func() (*debug.BuildInfo, bool)) func() {
	oldV, oldC, oldB := Version, Commit, BuildDate
	oldSource := Source
	oldFn := ReadBuildInfo
	Version = metadata.version
	Commit = metadata.commit
	BuildDate = metadata.buildDate
	Source = metadata.source
	ReadBuildInfo = fn
	return func() {
		Version = oldV
		Commit = oldC
		BuildDate = oldB
		Source = oldSource
		ReadBuildInfo = oldFn
	}
}

func TestGet_DevelopmentDefaults(t *testing.T) {
	cleanup := restoreVars(linkerMetadata{
		version: "dev", commit: "unknown", buildDate: "unknown", source: "source",
	}, func() (*debug.BuildInfo, bool) {
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
	cleanup := restoreVars(linkerMetadata{
		version: "dev", commit: "unknown", buildDate: "unknown", source: "source",
	}, func() (*debug.BuildInfo, bool) {
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
	cleanup := restoreVars(linkerMetadata{
		version: "dev", commit: "unknown", buildDate: "unknown", source: "source",
	}, func() (*debug.BuildInfo, bool) {
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
	cleanup := restoreVars(linkerMetadata{
		version: "dev", commit: "unknown", buildDate: "unknown", source: "source",
	}, func() (*debug.BuildInfo, bool) {
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
	cleanup := restoreVars(linkerMetadata{
		version: "v1.2.3", commit: "feedbeef", buildDate: "2026-06-15T12:00:00Z", source: "release",
	}, func() (*debug.BuildInfo, bool) {
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
	cleanup := restoreVars(linkerMetadata{
		version: "dev", commit: "unknown", buildDate: "unknown", source: "source",
	}, func() (*debug.BuildInfo, bool) {
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
	cleanup := restoreVars(linkerMetadata{
		version: "dev", commit: "unknown", buildDate: "unknown", source: "source",
	}, func() (*debug.BuildInfo, bool) {
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
	cleanup := restoreVars(linkerMetadata{
		version: "dev", commit: "unknown", buildDate: "unknown", source: "source",
	}, func() (*debug.BuildInfo, bool) {
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

func TestPolicy_DerivesSelfUpdateFromSource(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantState      PolicyState
		wantSource     DistributionSource
		wantSelfUpdate bool
	}{
		{name: "source", source: "source", wantState: PolicyEnabled, wantSource: SourceSource, wantSelfUpdate: true},
		{name: "release", source: "release", wantState: PolicyEnabled, wantSource: SourceRelease, wantSelfUpdate: true},
		{name: "development", source: "development", wantState: PolicyEnabled, wantSource: SourceDevelopment, wantSelfUpdate: true},
		{name: "npm", source: "npm", wantState: PolicyDisabled, wantSource: SourceNPM},
		{name: "homebrew", source: "homebrew", wantState: PolicyDisabled, wantSource: SourceHomebrew},
		{name: "winget", source: "winget", wantState: PolicyDisabled, wantSource: SourceWinget},
		{name: "nix", source: "nix", wantState: PolicyDisabled, wantSource: SourceNix},
		{name: "extension source", source: "corp_repo-1.2", wantState: PolicyDisabled, wantSource: "corp_repo-1.2"},
		{name: "empty source", source: "", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "reserved unknown", source: "unknown", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "leading punctuation", source: "-npm", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "uppercase source", source: "Homebrew", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "path source", source: "corp/npm", wantState: PolicyInvalid, wantSource: SourceUnknown},
		{name: "source too long", source: "a1234567890123456789012345678901234567890123456789012345678901234", wantState: PolicyInvalid, wantSource: SourceUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := restoreVars(linkerMetadata{
				version: "v1.0.0", commit: "abcdef0123456789",
				buildDate: "2026-07-24T00:00:00Z", source: tt.source,
			}, func() (*debug.BuildInfo, bool) {
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
