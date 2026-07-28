package version

import (
	"runtime/debug"
	"strings"
)

// Linker-overridable build metadata. Release builds injected via -ldflags -X.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// ReadBuildInfo is a package-level variable for test injection.
var ReadBuildInfo = debug.ReadBuildInfo

// Info holds the embedded version metadata exposed to callers.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

// Get returns build information. When linker values were injected (Version !=
// "dev") they are used exclusively. Otherwise it falls back to Go build
// information for optional enrichment of otherwise-default fields.
func Get() Info {
	info := Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	}

	// Linker-injected release values take absolute precedence.
	if Version != "dev" {
		return info
	}

	bi, ok := ReadBuildInfo()
	if !ok || bi == nil {
		return info
	}

	// Use the main module version when it carries a real tag.
	if mv := bi.Main.Version; mv != "" && mv != "(devel)" {
		info.Version = mv
	}

	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if info.Commit == "unknown" {
				info.Commit = s.Value
			}
		case "vcs.time":
			if info.BuildDate == "unknown" {
				info.BuildDate = s.Value
			}
		case "vcs.modified":
			if s.Value == "true" && !strings.Contains(info.Version, "dirty") {
				info.Version += "-dirty"
			}
		}
	}

	return info
}
