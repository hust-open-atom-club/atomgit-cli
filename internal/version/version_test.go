package version

import (
	"runtime/debug"
	"testing"
)

type linkerMetadata struct {
	version   string
	commit    string
	buildDate string
}

// restoreVars swaps package-level variables to the requested test values and
// returns a cleanup function that restores the originals.
func restoreVars(metadata linkerMetadata, fn func() (*debug.BuildInfo, bool)) func() {
	oldV, oldC, oldB := Version, Commit, BuildDate
	oldFn := ReadBuildInfo
	Version = metadata.version
	Commit = metadata.commit
	BuildDate = metadata.buildDate
	ReadBuildInfo = fn
	return func() {
		Version = oldV
		Commit = oldC
		BuildDate = oldB
		ReadBuildInfo = oldFn
	}
}

func TestGet_DevelopmentDefaults(t *testing.T) {
	cleanup := restoreVars(linkerMetadata{
		version: "dev", commit: "unknown", buildDate: "unknown",
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
		version: "dev", commit: "unknown", buildDate: "unknown",
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
		version: "dev", commit: "unknown", buildDate: "unknown",
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
		version: "dev", commit: "unknown", buildDate: "unknown",
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
		version: "v1.2.3", commit: "feedbeef", buildDate: "2026-06-15T12:00:00Z",
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
		version: "dev", commit: "unknown", buildDate: "unknown",
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
		version: "dev", commit: "unknown", buildDate: "unknown",
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
		version: "dev", commit: "unknown", buildDate: "unknown",
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
