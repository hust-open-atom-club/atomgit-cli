package version

import (
	"regexp"
	"runtime/debug"
	"strings"
)

// Linker-overridable build metadata. Release builds injected via -ldflags -X.
var (
	Version    = "dev"
	Commit     = "unknown"
	BuildDate  = "unknown"
	SelfUpdate = "true"
	Source     = "source"
)

// ReadBuildInfo is a package-level variable for test injection.
var ReadBuildInfo = debug.ReadBuildInfo

type DistributionSource string

const (
	SourceSource      DistributionSource = "source"
	SourceRelease     DistributionSource = "release"
	SourceDevelopment DistributionSource = "development"
	SourceNPM         DistributionSource = "npm"
	SourceHomebrew    DistributionSource = "homebrew"
	SourceWinget      DistributionSource = "winget"
	SourceNix         DistributionSource = "nix"
	SourceUnknown     DistributionSource = "unknown"
)

type PolicyState string

const (
	PolicyEnabled  PolicyState = "enabled"
	PolicyDisabled PolicyState = "disabled"
	PolicyInvalid  PolicyState = "invalid"
)

type BuildPolicy struct {
	State      PolicyState
	Source     DistributionSource
	SelfUpdate bool
}

// Info holds the embedded version metadata exposed to callers.
type Info struct {
	Version    string `json:"version"`
	Commit     string `json:"commit"`
	BuildDate  string `json:"buildDate"`
	SelfUpdate bool   `json:"selfUpdate"`
	Source     string `json:"source"`
}

var distributionSourcePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

func Policy() BuildPolicy {
	source := DistributionSource(Source)
	if (SelfUpdate != "true" && SelfUpdate != "false") ||
		source == SourceUnknown ||
		!distributionSourcePattern.MatchString(string(source)) {
		return BuildPolicy{
			State:  PolicyInvalid,
			Source: SourceUnknown,
		}
	}
	if SelfUpdate == "false" {
		return BuildPolicy{
			State:  PolicyDisabled,
			Source: source,
		}
	}
	return BuildPolicy{
		State:      PolicyEnabled,
		Source:     source,
		SelfUpdate: true,
	}
}

// Get returns build information. When linker values were injected (Version !=
// "dev") they are used exclusively. Otherwise it falls back to Go build
// information for optional enrichment of otherwise-default fields.
func Get() Info {
	policy := Policy()
	info := Info{
		Version:    Version,
		Commit:     Commit,
		BuildDate:  BuildDate,
		SelfUpdate: policy.SelfUpdate,
		Source:     string(policy.Source),
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
