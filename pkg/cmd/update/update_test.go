package update

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	internalversion "atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSelectLatestStableRelease(t *testing.T) {
	releases := []api.Release{
		{TagName: "v2.0.0", Prerelease: true},
		{TagName: "v1.9.0", ReleaseStatus: api.ReleaseStatusPre},
		{TagName: "not-semver"},
		{TagName: "v1.3.0"},
		{TagName: "v1.10.0"},
		{TagName: "v1.4.0", Draft: true},
	}

	got, err := selectLatestStableRelease(releases)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.10.0" {
		t.Fatalf("selectLatestStableRelease() = %q, want v1.10.0", got)
	}
}

func TestSelectLatestStableReleaseRejectsEmptySet(t *testing.T) {
	_, err := selectLatestStableRelease([]api.Release{{TagName: "v2.0.0-rc.1"}})
	if err == nil || !strings.Contains(err.Error(), "no stable AtomGit CLI release found") {
		t.Fatalf("selectLatestStableRelease() error = %v", err)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    status
	}{
		{name: "older", current: "v1.2.3", latest: "v1.3.0", want: statusUpdateAvailable},
		{name: "without v prefix", current: "1.3.0", latest: "v1.3.0", want: statusUpToDate},
		{name: "equal", current: "v1.3.0", latest: "v1.3.0", want: statusUpToDate},
		{name: "newer", current: "v1.4.0", latest: "v1.3.0", want: statusNewer},
		{name: "prerelease", current: "v1.3.0-rc.1", latest: "v1.3.0", want: statusUpdateAvailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compareVersions(tt.current, tt.latest)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("compareVersions() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompareVersionsRejectsUncomparableCurrentVersion(t *testing.T) {
	for _, current := range []string{"dev", "unknown", "v1.2.3-dirty", "f606e04"} {
		t.Run(current, func(t *testing.T) {
			_, err := compareVersions(current, "v1.3.0")
			if err == nil || !strings.Contains(err.Error(), "not a comparable semantic version") {
				t.Fatalf("compareVersions() error = %v", err)
			}
		})
	}
}

func TestUpdateCheckDoesNotDetectOrInstall(t *testing.T) {
	setCurrentVersion(t, "v1.2.3")
	requestedExecutable := false
	deps := testDeps()
	deps.executable = func() (string, error) {
		requestedExecutable = true
		return "", errors.New("must not be called")
	}
	cmd := newCmdUpdateWithDeps(releaseFactory(t, "v1.3.0"), deps)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--check"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if requestedExecutable {
		t.Fatal("--check detected the installation source")
	}
	want := "Current version: v1.2.3\nLatest release: v1.3.0\nStatus: update available\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestCheckUpdateCompatibilityCommandDoesNotDetectOrInstall(t *testing.T) {
	setCurrentVersion(t, "v1.2.3")
	deps := testDeps()
	deps.executable = func() (string, error) {
		t.Fatal("check-update detected the installation source")
		return "", nil
	}
	deps.chooseUpdate = func(io.Reader, io.Writer, installation, string) (updateChoice, error) {
		t.Fatal("check-update prompted for an update")
		return "", nil
	}
	deps.run = func(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error {
		t.Fatal("check-update invoked a package manager")
		return nil
	}
	cmd := newCmdCheckUpdateWithDeps(releaseFactory(t, "v1.3.0"), deps)
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := "Current version: v1.2.3\nLatest release: v1.3.0\nStatus: update available\n"
	if !strings.Contains(out.String(), `Command "check-update" is deprecated`) ||
		!strings.HasSuffix(out.String(), want) {
		t.Fatalf("output = %q, want deprecation warning followed by %q", out.String(), want)
	}
	if cmd.Deprecated == "" {
		t.Fatal("check-update is not marked deprecated")
	}
}

func TestUpdateAlreadyCurrentDoesNotDetectOrInstall(t *testing.T) {
	setCurrentVersion(t, "v1.3.0")
	deps := testDeps()
	deps.executable = func() (string, error) {
		t.Fatal("up-to-date command detected the installation source")
		return "", nil
	}
	cmd := newCmdUpdateWithDeps(releaseFactory(t, "v1.3.0"), deps)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestPromptUpdateChoice(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		installed installation
		want      updateChoice
		wantText  string
		wantErr   string
	}{
		{name: "update", input: "1\n", installed: installation{source: sourceNPM}, want: choiceUpdate, wantText: "Update via npm"},
		{name: "default update", input: "\n", installed: installation{source: sourceNPM}, want: choiceUpdate, wantText: "default: 1"},
		{name: "EOF defaults to update", input: "", installed: installation{source: sourceNPM}, want: choiceUpdate, wantText: "default: 1"},
		{name: "skip", input: "2\n", installed: installation{source: sourceNPM}, want: choiceSkip, wantText: "Skip"},
		{name: "Homebrew Core", input: "1\n", installed: installation{source: sourceHomebrewCore}, want: choiceUpdate, wantText: "Update via Homebrew Core"},
		{name: "retry invalid", input: "later\n1\n", installed: installation{source: sourceNPM}, want: choiceUpdate, wantText: "Please choose 1 or 2."},
		{name: "invalid then EOF", input: "n\n", installed: installation{source: sourceNPM}, wantText: "Please choose 1 or 2.", wantErr: "input ended after an invalid answer"},
		{name: "invalid at EOF", input: "n", installed: installation{source: sourceNPM}, wantText: "Please choose 1 or 2.", wantErr: "input ended after an invalid answer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			got, err := promptUpdateChoice(strings.NewReader(tt.input), &out, tt.installed, "v1.3.0")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("promptUpdateChoice() error = %v, want %q", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if got != tt.want {
					t.Fatalf("promptUpdateChoice() = %q, want %q", got, tt.want)
				}
			}
			if !strings.Contains(out.String(), tt.wantText) {
				t.Fatalf("prompt output does not contain %q:\n%s", tt.wantText, out.String())
			}
		})
	}
}

func TestUpdateSkipDoesNotInstall(t *testing.T) {
	setCurrentVersion(t, "v1.2.3")
	root := filepath.Join(t.TempDir(), "lib", "node_modules")
	executable, err := npmExecutablePath(root, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.executable = func() (string, error) { return executable, nil }
	deps.lookPath = func(name string) (string, error) {
		if name == "npm" {
			return "/usr/bin/npm", nil
		}
		return "", errors.New("not found")
	}
	deps.capture = func(_ context.Context, name string, args ...string) (commandResult, error) {
		if name == "npm" && reflect.DeepEqual(args, []string{"root", "-g"}) {
			return commandResult{stdout: root}, nil
		}
		return commandResult{}, fmt.Errorf("unexpected command %s %v", name, args)
	}
	deps.chooseUpdate = func(_ io.Reader, _ io.Writer, installed installation, latest string) (updateChoice, error) {
		if installed.source != sourceNPM || latest != "v1.3.0" {
			t.Fatalf("prompt got installation=%#v latest=%q", installed, latest)
		}
		return choiceSkip, nil
	}
	deps.run = func(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error {
		t.Fatal("skip choice invoked a package manager")
		return nil
	}

	cmd := newCmdUpdateWithDeps(releaseFactory(t, "v1.3.0"), deps)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Update skipped.") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestUpdateViaNPM(t *testing.T) {
	setCurrentVersion(t, "v1.2.3")
	root := filepath.Join(t.TempDir(), "lib", "node_modules")
	prefix := filepath.Dir(filepath.Dir(root))
	executable, err := npmExecutablePath(root, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.goos = "linux"
	deps.goarch = "amd64"
	deps.executable = func() (string, error) { return executable, nil }
	deps.lookPath = func(name string) (string, error) {
		if name == "npm" {
			return "/usr/bin/npm", nil
		}
		return "", errors.New("not found")
	}
	var captures [][]string
	deps.capture = func(_ context.Context, name string, args ...string) (commandResult, error) {
		captures = append(captures, append([]string{name}, args...))
		switch {
		case name == "npm" && reflect.DeepEqual(args, []string{"root", "-g"}):
			return commandResult{stdout: root + "\n"}, nil
		case name == "npm" && reflect.DeepEqual(args, []string{
			"view", npmPackage + "@1.3.0", "version", "--json", "--registry=" + npmRegistry,
		}):
			return commandResult{stdout: `"1.3.0"`}, nil
		case name == "npm" && reflect.DeepEqual(args, []string{"prefix", "-g"}):
			return commandResult{stdout: prefix + "\n"}, nil
		case name == npmLauncherPath(prefix, "linux"):
			return commandResult{stdout: `{"version":"v1.3.0"}`}, nil
		default:
			return commandResult{}, fmt.Errorf("unexpected command %s %v", name, args)
		}
	}
	var runs [][]string
	deps.run = func(_ context.Context, _ io.Reader, stdout, _ io.Writer, name string, args ...string) error {
		runs = append(runs, append([]string{name}, args...))
		_, _ = fmt.Fprintln(stdout, "npm output")
		return nil
	}

	cmd := newCmdUpdateWithDeps(releaseFactory(t, "v1.3.0"), deps)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	wantRun := []string{
		"npm", "install", "-g", npmPackage + "@1.3.0", "--registry=" + npmRegistry, "--no-audit", "--no-fund",
	}
	if len(runs) != 1 || !reflect.DeepEqual(runs[0], wantRun) {
		t.Fatalf("runs = %#v, want %#v", runs, [][]string{wantRun})
	}
	if len(captures) != 4 {
		t.Fatalf("captures = %#v, want npm root, publication, prefix, and command-entry checks", captures)
	}
	wantVerification := []string{npmLauncherPath(prefix, "linux"), "version", "--json"}
	if !reflect.DeepEqual(captures[len(captures)-1], wantVerification) {
		t.Fatalf("verification = %#v, want npm command entry %#v", captures[len(captures)-1], wantVerification)
	}
	for _, want := range []string{"Updating via npm...", "npm output", "Updated AtomGit CLI via npm to v1.3.0."} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output does not contain %q:\n%s", want, out.String())
		}
	}
}

func TestEnsureNPMVersionAvailableRejectsUnpublishedRelease(t *testing.T) {
	deps := testDeps()
	deps.capture = func(_ context.Context, name string, args ...string) (commandResult, error) {
		if name != "npm" || len(args) == 0 || args[0] != "view" {
			t.Fatalf("unexpected command %s %v", name, args)
		}
		return commandResult{stderr: "npm error code E404"}, errors.New("exit status 1")
	}

	err := ensureNPMVersionAvailable(context.Background(), deps, "1.3.0")
	if err == nil || !strings.Contains(err.Error(), "not available from the official npm registry yet") ||
		!strings.Contains(err.Error(), "E404") {
		t.Fatalf("ensureNPMVersionAvailable() error = %v", err)
	}
}

func TestParseNPMViewVersion(t *testing.T) {
	for _, input := range []string{`"1.3.0"`, `["1.3.0"]`} {
		got, err := parseNPMViewVersion(input)
		if err != nil {
			t.Fatalf("parseNPMViewVersion(%s): %v", input, err)
		}
		if got != "1.3.0" {
			t.Fatalf("parseNPMViewVersion(%s) = %q", input, got)
		}
	}
	for _, input := range []string{`[]`, `["1.3.0","1.3.1"]`, `{}`, `not-json`} {
		if _, err := parseNPMViewVersion(input); err == nil {
			t.Fatalf("parseNPMViewVersion(%s) succeeded", input)
		}
	}
}

func TestUpdateViaNPMReportsInstallFailureWithUsableLauncher(t *testing.T) {
	prefix := t.TempDir()
	deps := testDeps()
	deps.capture = func(_ context.Context, name string, args ...string) (commandResult, error) {
		switch {
		case name == "npm" && len(args) > 0 && args[0] == "view":
			return commandResult{stdout: `"1.3.0"`}, nil
		case name == "npm" && reflect.DeepEqual(args, []string{"prefix", "-g"}):
			return commandResult{stdout: prefix}, nil
		case name == npmLauncherPath(prefix, "linux"):
			return commandResult{stdout: `{"version":"v1.2.3"}`}, nil
		default:
			return commandResult{}, fmt.Errorf("unexpected command %s %v", name, args)
		}
	}
	deps.run = func(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error {
		return errors.New("exit status 1")
	}
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := updateViaNPM(cmd, deps, "v1.3.0")
	if err == nil || !strings.Contains(err.Error(), "still reports v1.2.3") {
		t.Fatalf("updateViaNPM() error = %v", err)
	}
}

func TestUpdateViaNPMRepairsBrokenWindowsLauncher(t *testing.T) {
	prefix := t.TempDir()
	for _, name := range []string{"ag", "ag.cmd", "ag.ps1"} {
		if err := os.WriteFile(filepath.Join(prefix, name), []byte("old launcher"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	binary := []byte("standalone ag.exe")
	archive := makeWindowsArchive(t, binary)
	checksum := fmt.Sprintf("%x  ag_windows_amd64.zip\n", sha256.Sum256(archive))

	deps := testDeps()
	deps.goos = "windows"
	deps.goarch = "amd64"
	deps.capture = func(_ context.Context, name string, args ...string) (commandResult, error) {
		switch {
		case name == "npm" && len(args) > 0 && args[0] == "view":
			return commandResult{stdout: `"1.3.0"`}, nil
		case name == "npm" && reflect.DeepEqual(args, []string{"prefix", "-g"}):
			return commandResult{stdout: prefix}, nil
		case name == "cmd.exe":
			wantArgs := []string{"/d", "/s", "/c", fmt.Sprintf("\"%s\" version --json", filepath.Join(prefix, "ag.cmd"))}
			if !reflect.DeepEqual(args, wantArgs) {
				t.Fatalf("cmd.exe args = %#v, want %#v", args, wantArgs)
			}
			return commandResult{stderr: "launcher missing"}, errors.New("exit status 1")
		case strings.HasSuffix(name, ".exe"):
			return commandResult{stdout: `{"version":"v1.3.0"}`}, nil
		default:
			return commandResult{}, fmt.Errorf("unexpected command %s %v", name, args)
		}
	}
	deps.run = func(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error {
		return errors.New("locked running executable")
	}
	deps.download = func(_ context.Context, rawURL string, _ int64) ([]byte, error) {
		switch {
		case strings.HasSuffix(rawURL, "/checksums.txt"):
			return []byte(checksum), nil
		case strings.HasSuffix(rawURL, "/ag_windows_amd64.zip"):
			return archive, nil
		default:
			return nil, fmt.Errorf("unexpected download %s", rawURL)
		}
	}
	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := updateViaNPM(cmd, deps, "v1.3.0"); err != nil {
		t.Fatal(err)
	}
	installed, err := os.ReadFile(filepath.Join(prefix, "ag.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(installed, binary) {
		t.Fatalf("installed binary = %q", installed)
	}
	for _, name := range []string{"ag", "ag.cmd", "ag.ps1"} {
		if _, err := os.Stat(filepath.Join(prefix, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("old launcher %s still exists: %v", name, err)
		}
	}
	if !strings.Contains(stderr.String(), "no longer managed by npm") ||
		!strings.Contains(stdout.String(), "Updated AtomGit CLI to v1.3.0 after repairing the npm command entry.") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRepairWindowsNPMLauncherRestoresOldLaunchersOnVerificationFailure(t *testing.T) {
	prefix := t.TempDir()
	oldLaunchers := map[string]string{
		"ag":     "old shell launcher",
		"ag.cmd": "old cmd launcher",
		"ag.ps1": "old powershell launcher",
	}
	for name, contents := range oldLaunchers {
		if err := os.WriteFile(filepath.Join(prefix, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	archive := makeWindowsArchive(t, []byte("new binary"))
	checksum := fmt.Sprintf("%x  ag_windows_amd64.zip\n", sha256.Sum256(archive))
	deps := testDeps()
	deps.goos = "windows"
	deps.goarch = "amd64"
	deps.download = func(_ context.Context, rawURL string, _ int64) ([]byte, error) {
		if strings.HasSuffix(rawURL, "/checksums.txt") {
			return []byte(checksum), nil
		}
		return archive, nil
	}
	deps.capture = func(_ context.Context, name string, _ ...string) (commandResult, error) {
		if strings.Contains(filepath.Base(name), ".ag-update-") {
			return commandResult{stdout: `{"version":"v1.3.0"}`}, nil
		}
		return commandResult{stderr: "new command failed"}, errors.New("exit status 1")
	}

	if _, _, err := repairWindowsNPMLauncher(context.Background(), deps, prefix, "v1.3.0"); err == nil ||
		!strings.Contains(err.Error(), "verify repaired command") {
		t.Fatalf("repairWindowsNPMLauncher() error = %v", err)
	}
	for name, want := range oldLaunchers {
		contents, err := os.ReadFile(filepath.Join(prefix, name))
		if err != nil {
			t.Fatalf("read restored %s: %v", name, err)
		}
		if string(contents) != want {
			t.Fatalf("restored %s = %q, want %q", name, contents, want)
		}
	}
	if _, err := os.Stat(filepath.Join(prefix, "ag.exe")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed repaired executable remains: %v", err)
	}
}

func TestRepairWindowsNPMLauncherRejectsChecksumBeforeChangingLaunchers(t *testing.T) {
	prefix := t.TempDir()
	launcher := filepath.Join(prefix, "ag.cmd")
	if err := os.WriteFile(launcher, []byte("old launcher"), 0o600); err != nil {
		t.Fatal(err)
	}
	archive := makeWindowsArchive(t, []byte("new binary"))
	deps := testDeps()
	deps.goarch = "amd64"
	deps.download = func(_ context.Context, rawURL string, _ int64) ([]byte, error) {
		if strings.HasSuffix(rawURL, "/checksums.txt") {
			return []byte(strings.Repeat("0", sha256.Size*2) + "  ag_windows_amd64.zip\n"), nil
		}
		return archive, nil
	}

	if _, _, err := repairWindowsNPMLauncher(context.Background(), deps, prefix, "v1.3.0"); err == nil ||
		!strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("repairWindowsNPMLauncher() error = %v", err)
	}
	contents, err := os.ReadFile(launcher)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "old launcher" {
		t.Fatalf("launcher changed before checksum validation: %q", contents)
	}
}

func makeWindowsArchive(t *testing.T, binary []byte) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Create("ag.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(binary); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

func TestUpdateViaHomebrewCore(t *testing.T) {
	setCurrentVersion(t, "v1.2.3")
	prefix := filepath.Join(t.TempDir(), "Cellar", "atomgit-cli", "1.2.3")
	executable := filepath.Join(prefix, "bin", "ag")
	deps := testDeps()
	deps.goos = "darwin"
	deps.executable = func() (string, error) { return executable, nil }
	deps.lookPath = func(name string) (string, error) {
		if name == "brew" {
			return "/opt/homebrew/bin/brew", nil
		}
		return "", errors.New("not found")
	}
	var captures [][]string
	deps.capture = func(_ context.Context, name string, args ...string) (commandResult, error) {
		captures = append(captures, append([]string{name}, args...))
		switch {
		case name == "brew" && reflect.DeepEqual(args, []string{"list", "--formula", "--full-name"}):
			return commandResult{stdout: homebrewCoreFormula + "\n"}, nil
		case name == "brew" && reflect.DeepEqual(args, []string{"--prefix", homebrewCoreFormula}):
			return commandResult{stdout: prefix + "\n"}, nil
		case name == executable:
			return commandResult{stdout: `{"version":"v1.3.0"}`}, nil
		default:
			return commandResult{}, fmt.Errorf("unexpected command %s %v", name, args)
		}
	}
	var runs [][]string
	deps.run = func(_ context.Context, _ io.Reader, _ io.Writer, _ io.Writer, name string, args ...string) error {
		runs = append(runs, append([]string{name}, args...))
		return nil
	}

	cmd := newCmdUpdateWithDeps(releaseFactory(t, "v1.3.0"), deps)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	wantRuns := [][]string{{"brew", "update"}, {"brew", "upgrade", homebrewCoreFormula}}
	if !reflect.DeepEqual(runs, wantRuns) {
		t.Fatalf("runs = %#v, want %#v", runs, wantRuns)
	}
	if len(captures) != 4 {
		t.Fatalf("captures = %#v, want list, two prefixes, and version check", captures)
	}
	if !strings.Contains(out.String(), "Updated AtomGit CLI via Homebrew Core to v1.3.0.") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestUpdateDoesNotTreatHomebrewTapAsCore(t *testing.T) {
	setCurrentVersion(t, "v1.2.3")
	prefix := filepath.Join(t.TempDir(), "Cellar", "atomgit-cli", "1.2.3")
	executable := filepath.Join(prefix, "bin", "ag")
	deps := testDeps()
	deps.goos = "darwin"
	deps.executable = func() (string, error) { return executable, nil }
	deps.lookPath = func(name string) (string, error) {
		if name == "brew" {
			return "/opt/homebrew/bin/brew", nil
		}
		return "", errors.New("not found")
	}
	deps.capture = func(_ context.Context, name string, args ...string) (commandResult, error) {
		if name == "brew" && reflect.DeepEqual(args, []string{"list", "--formula", "--full-name"}) {
			return commandResult{stdout: "hust-open-atom-club/tap/atomgit-cli\n"}, nil
		}
		return commandResult{}, fmt.Errorf("unexpected command %s %v", name, args)
	}
	deps.run = func(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error {
		t.Fatal("Homebrew Tap update invoked a package manager")
		return nil
	}

	cmd := newCmdUpdateWithDeps(releaseFactory(t, "v1.3.0"), deps)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "could not identify the installation source") {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestUpdateRejectsUnknownInstallation(t *testing.T) {
	setCurrentVersion(t, "v1.2.3")
	deps := testDeps()
	deps.executable = func() (string, error) { return "/usr/local/bin/ag", nil }
	deps.lookPath = func(string) (string, error) { return "", errors.New("not found") }
	cmd := newCmdUpdateWithDeps(releaseFactory(t, "v1.3.0"), deps)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "could not identify the installation source") {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestUpdateRejectsInvalidCurrentVersionBeforeHTTP(t *testing.T) {
	setCurrentVersion(t, "dev")
	requested := false
	factory := &cmdutil.Factory{HttpClient: func() (*http.Client, error) {
		requested = true
		return &http.Client{}, nil
	}}
	cmd := newCmdUpdateWithDeps(factory, testDeps())
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "not a comparable semantic version") {
		t.Fatalf("Execute() error = %v", err)
	}
	if requested {
		t.Fatal("HTTP client was requested for an invalid local version")
	}
}

func TestUpdateHelp(t *testing.T) {
	cmd := NewCmdUpdate(&cmdutil.Factory{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Update AtomGit CLI", "update", "--check", "without installing"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("help does not contain %q:\n%s", want, out.String())
		}
	}
}

func TestUpdateRejectsArguments(t *testing.T) {
	cmd := NewCmdUpdate(&cmdutil.Factory{})
	cmd.SetArgs([]string{"extra"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("update accepted a positional argument")
	}
}

func TestVerifyInstalledVersionRejectsMismatch(t *testing.T) {
	deps := testDeps()
	deps.capture = func(context.Context, string, ...string) (commandResult, error) {
		return commandResult{stdout: `{"version":"v1.2.3"}`}, nil
	}
	err := verifyInstalledVersion(context.Background(), deps, "/tmp/ag", "v1.3.0")
	if err == nil || !strings.Contains(err.Error(), `installed version is "v1.2.3", expected "v1.3.0"`) {
		t.Fatalf("verifyInstalledVersion() error = %v", err)
	}
}

func TestNPMExecutablePath(t *testing.T) {
	tests := []struct {
		goos, goarch string
		wantPackage  string
		wantBinary   string
	}{
		{goos: "linux", goarch: "amd64", wantPackage: "atomgit-cli-linux-x64", wantBinary: "ag"},
		{goos: "linux", goarch: "loong64", wantPackage: "atomgit-cli-linux-loong64", wantBinary: "ag"},
		{goos: "darwin", goarch: "arm64", wantPackage: "atomgit-cli-darwin-arm64", wantBinary: "ag"},
		{goos: "windows", goarch: "amd64", wantPackage: "atomgit-cli-win32-x64", wantBinary: "ag.exe"},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			got, err := npmExecutablePath("root", tt.goos, tt.goarch)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, tt.wantPackage) || filepath.Base(got) != tt.wantBinary {
				t.Fatalf("npmExecutablePath() = %q", got)
			}
		})
	}
}

func releaseFactory(t *testing.T, latest string) *cmdutil.Factory {
	t.Helper()
	return &cmdutil.Factory{HttpClient: func() (*http.Client, error) {
		return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", req.Method)
			}
			if req.URL.Path != "/api/v5/repos/hust-open-atom-club/atomgit-cli/releases" {
				t.Fatalf("path = %q", req.URL.Path)
			}
			if got := req.URL.Query().Get("page"); got != "1" {
				t.Fatalf("page = %q, want 1", got)
			}
			if got := req.URL.Query().Get("per_page"); got != "100" {
				t.Fatalf("per_page = %q, want 100", got)
			}
			if got := req.URL.Query().Get("direction"); got != "desc" {
				t.Fatalf("direction = %q, want desc", got)
			}
			if authorization := req.Header.Get("Authorization"); authorization != "" {
				t.Fatalf("Authorization = %q, want empty", authorization)
			}
			if req.Body != nil && req.Body != http.NoBody {
				t.Fatal("request body is not empty")
			}
			body := fmt.Sprintf(`[{"tag_name":%q}]`, latest)
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    req,
			}, nil
		})}, nil
	}}
}

func setCurrentVersion(t *testing.T, value string) {
	t.Helper()
	oldVersion := internalversion.Version
	internalversion.Version = value
	t.Cleanup(func() { internalversion.Version = oldVersion })
}

func testDeps() updateDeps {
	return updateDeps{
		executable:   func() (string, error) { return "/tmp/ag", nil },
		evalSymlinks: func(value string) (string, error) { return value, nil },
		lookPath:     func(string) (string, error) { return "", errors.New("not found") },
		capture: func(context.Context, string, ...string) (commandResult, error) {
			return commandResult{}, errors.New("unexpected capture")
		},
		run: func(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error {
			return errors.New("unexpected run")
		},
		download: func(context.Context, string, int64) ([]byte, error) {
			return nil, errors.New("unexpected download")
		},
		chooseUpdate: func(io.Reader, io.Writer, installation, string) (updateChoice, error) {
			return choiceUpdate, nil
		},
		goos:   "linux",
		goarch: "amd64",
	}
}
