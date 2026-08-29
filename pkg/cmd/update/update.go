package update

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	internalversion "atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const (
	projectOwner           = "hust-open-atom-club"
	projectRepo            = "atomgit-cli"
	releaseLimit           = 100
	npmPackage             = "@hust-open-atom-club/atomgit-cli"
	npmRegistry            = "https://registry.npmjs.org/"
	releaseDownloadBaseURL = "https://atomgit.com/hust-open-atom-club/atomgit-cli/releases/download"
	homebrewCoreFormula    = "atomgit-cli"
	homebrewExecutableName = "ag"
	maxChecksumBytes       = 1 << 20
	maxWindowsArchiveBytes = 128 << 20
	maxWindowsBinaryBytes  = 128 << 20
)

type status string

const (
	statusUpdateAvailable status = "update available"
	statusUpToDate        status = "up to date"
	statusNewer           status = "current version is newer"
)

type installationSource string

const (
	sourceNPM          installationSource = "npm"
	sourceHomebrewCore installationSource = "homebrew-core"
)

type installation struct {
	source     installationSource
	executable string
	formula    string
}

type updateChoice string

const (
	choiceUpdate updateChoice = "update"
	choiceSkip   updateChoice = "skip"
)

type commandResult struct {
	stdout string
	stderr string
}

type installedVersionVerification struct {
	executable string
	actual     string
	err        error
}

func (v installedVersionVerification) ok(expected string) bool {
	return v.err == nil && normalizeVersion(v.actual) == normalizeVersion(expected)
}

func (v installedVersionVerification) usable() bool {
	return v.err == nil && v.actual != ""
}

type updateDeps struct {
	executable   func() (string, error)
	evalSymlinks func(string) (string, error)
	lookPath     func(string) (string, error)
	capture      func(context.Context, string, ...string) (commandResult, error)
	run          func(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error
	download     func(context.Context, string, int64) ([]byte, error)
	chooseUpdate func(io.Reader, io.Writer, installation, string) (updateChoice, error)
	goos         string
	goarch       string
}

func defaultUpdateDeps() updateDeps {
	return updateDeps{
		executable:   os.Executable,
		evalSymlinks: filepath.EvalSymlinks,
		lookPath:     exec.LookPath,
		capture: func(ctx context.Context, name string, args ...string) (commandResult, error) {
			var stdout, stderr bytes.Buffer
			command := exec.CommandContext(ctx, name, args...)
			command.Stdout = &stdout
			command.Stderr = &stderr
			err := command.Run()
			return commandResult{stdout: stdout.String(), stderr: stderr.String()}, err
		},
		run: func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, name string, args ...string) error {
			command := exec.CommandContext(ctx, name, args...)
			command.Stdin = stdin
			command.Stdout = stdout
			command.Stderr = stderr
			return command.Run()
		},
		download: downloadUpdateAsset,
		chooseUpdate: func(in io.Reader, out io.Writer, installed installation, latest string) (updateChoice, error) {
			return promptUpdateChoice(in, out, installed, latest)
		},
		goos:   runtime.GOOS,
		goarch: runtime.GOARCH,
	}
}

func NewCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	return newCmdUpdateWithDeps(f, defaultUpdateDeps())
}

func newCmdUpdateWithDeps(f *cmdutil.Factory, deps updateDeps) *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update AtomGit CLI to the latest stable release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdate(cmd, f, deps, check)
		},
	}
	cmd.Flags().BoolVarP(&check, "check", "c", false, "Check for an update without installing it")
	return cmd
}

// NewCmdCheckUpdate preserves the legacy read-only command while users migrate
// to "ag update --check".
func NewCmdCheckUpdate(f *cmdutil.Factory) *cobra.Command {
	return newCmdCheckUpdateWithDeps(f, defaultUpdateDeps())
}

func newCmdCheckUpdateWithDeps(f *cmdutil.Factory, deps updateDeps) *cobra.Command {
	return &cobra.Command{
		Use:        "check-update",
		Short:      "Check for a newer AtomGit CLI release",
		Deprecated: `use "ag update --check" instead`,
		Args:       cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdate(cmd, f, deps, true)
		},
	}
}

func runUpdate(cmd *cobra.Command, f *cmdutil.Factory, deps updateDeps, check bool) error {
	current := internalversion.Get().Version
	if err := validateComparableVersion(current); err != nil {
		return err
	}

	client, err := publicAPIClient(f)
	if err != nil {
		return err
	}
	releases, err := api.ListReleases(client, projectOwner, projectRepo, releaseLimit)
	if err != nil {
		return fmt.Errorf("check AtomGit CLI releases: %w", err)
	}
	latest, err := selectLatestStableRelease(releases)
	if err != nil {
		return err
	}
	result, err := compareVersions(current, latest)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"Current version: %s\nLatest release: %s\nStatus: %s\n",
		current,
		latest,
		result,
	); err != nil {
		return err
	}

	if check || result != statusUpdateAvailable {
		return nil
	}

	installed, err := detectInstallation(cmd.Context(), deps)
	if err != nil {
		return err
	}
	choice, err := deps.chooseUpdate(cmd.InOrStdin(), cmd.ErrOrStderr(), installed, latest)
	if err != nil {
		return err
	}
	switch choice {
	case choiceSkip:
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Update skipped.")
		return err
	case choiceUpdate:
	default:
		return fmt.Errorf("unsupported update choice %q", choice)
	}
	switch installed.source {
	case sourceNPM:
		return updateViaNPM(cmd, deps, latest)
	case sourceHomebrewCore:
		return updateViaHomebrewCore(cmd, deps, installed, latest)
	default:
		return fmt.Errorf("unsupported AtomGit CLI installation source %q", installed.source)
	}
}

func promptUpdateChoice(
	in io.Reader,
	out io.Writer,
	installed installation,
	latest string,
) (updateChoice, error) {
	manager := installationDisplayName(installed)
	if _, err := fmt.Fprintf(
		out,
		"Update AtomGit CLI to %s using %s?\n  1. Update via %s\n  2. Skip\n",
		latest,
		manager,
		manager,
	); err != nil {
		return "", fmt.Errorf("write update choices: %w", err)
	}
	reader := bufio.NewReader(in)
	hadInvalidAnswer := false
	for {
		if _, err := fmt.Fprint(out, "Select an option [1-2] (default: 1): "); err != nil {
			return "", fmt.Errorf("write update choice prompt: %w", err)
		}
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read update choice: %w", err)
		}
		answer := strings.ToLower(strings.TrimSpace(line))
		switch answer {
		case "":
			if errors.Is(err, io.EOF) && hadInvalidAnswer {
				return "", fmt.Errorf("read update choice: input ended after an invalid answer")
			}
			return choiceUpdate, nil
		case "1", "u", "update":
			return choiceUpdate, nil
		case "2", "s", "skip":
			return choiceSkip, nil
		default:
			if _, writeErr := fmt.Fprintln(out, "Please choose 1 or 2."); writeErr != nil {
				return "", fmt.Errorf("write invalid update choice message: %w", writeErr)
			}
			hadInvalidAnswer = true
		}
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read update choice: input ended after an invalid answer")
		}
	}
}

func installationDisplayName(installed installation) string {
	switch installed.source {
	case sourceNPM:
		return "npm"
	case sourceHomebrewCore:
		return "Homebrew Core"
	default:
		return string(installed.source)
	}
}

func publicAPIClient(f *cmdutil.Factory) (*api.Client, error) {
	if f == nil || f.HttpClient == nil {
		return api.NewClient(""), nil
	}
	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("create update HTTP client: %w", err)
	}
	if httpClient == nil {
		return api.NewClient(""), nil
	}
	return api.NewClientWithHTTPClient("", httpClient), nil
}

func selectLatestStableRelease(releases []api.Release) (string, error) {
	latest := ""
	for _, release := range releases {
		candidate := normalizeVersion(release.TagName)
		if release.Draft ||
			release.Prerelease ||
			release.ReleaseStatus == api.ReleaseStatusPre ||
			!semver.IsValid(candidate) ||
			semver.Prerelease(candidate) != "" {
			continue
		}
		if latest == "" || semver.Compare(candidate, latest) > 0 {
			latest = candidate
		}
	}
	if latest == "" {
		return "", errors.New("no stable AtomGit CLI release found")
	}
	return latest, nil
}

func compareVersions(current, latest string) (status, error) {
	current = normalizeVersion(current)
	if err := validateComparableVersion(current); err != nil {
		return "", err
	}
	switch semver.Compare(current, latest) {
	case -1:
		return statusUpdateAvailable, nil
	case 0:
		return statusUpToDate, nil
	default:
		return statusNewer, nil
	}
}

func validateComparableVersion(value string) error {
	value = normalizeVersion(value)
	if strings.Contains(strings.ToLower(value), "dirty") || !semver.IsValid(value) {
		return fmt.Errorf("current version %q is not a comparable semantic version", value)
	}
	return nil
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	if value != "" && value[0] != 'v' {
		return "v" + value
	}
	return value
}

func detectInstallation(ctx context.Context, deps updateDeps) (installation, error) {
	executable, err := deps.executable()
	if err != nil {
		return installation{}, fmt.Errorf("resolve running ag executable: %w", err)
	}
	executable = canonicalPath(executable, deps.evalSymlinks)

	if _, err := deps.lookPath("npm"); err == nil {
		if installed, ok := detectNPMInstallation(ctx, deps, executable); ok {
			return installed, nil
		}
	}
	if _, err := deps.lookPath("brew"); err == nil {
		if installed, ok := detectHomebrewInstallation(ctx, deps, executable); ok {
			return installed, nil
		}
	}

	return installation{}, fmt.Errorf(
		"could not identify the installation source for %s; automatic updates currently support global npm and Homebrew Core installations",
		executable,
	)
}

func detectNPMInstallation(ctx context.Context, deps updateDeps, executable string) (installation, bool) {
	root, err := npmGlobalRoot(ctx, deps)
	if err != nil {
		return installation{}, false
	}
	expected, err := npmExecutablePath(root, deps.goos, deps.goarch)
	if err != nil {
		return installation{}, false
	}
	expected = canonicalPath(expected, deps.evalSymlinks)
	if !samePath(executable, expected, deps.goos) {
		return installation{}, false
	}
	return installation{source: sourceNPM, executable: expected}, true
}

func detectHomebrewInstallation(ctx context.Context, deps updateDeps, executable string) (installation, bool) {
	result, err := deps.capture(ctx, "brew", "list", "--formula", "--full-name")
	if err != nil {
		return installation{}, false
	}
	installedFormulae := make(map[string]bool)
	for _, formula := range strings.Fields(result.stdout) {
		installedFormulae[formula] = true
	}
	if !installedFormulae[homebrewCoreFormula] {
		return installation{}, false
	}
	prefix, err := homebrewPrefix(ctx, deps, homebrewCoreFormula)
	if err != nil {
		return installation{}, false
	}
	expected := canonicalPath(filepath.Join(prefix, "bin", homebrewExecutableName), deps.evalSymlinks)
	if samePath(executable, expected, deps.goos) {
		return installation{source: sourceHomebrewCore, executable: expected, formula: homebrewCoreFormula}, true
	}
	return installation{}, false
}

func npmGlobalRoot(ctx context.Context, deps updateDeps) (string, error) {
	result, err := deps.capture(ctx, "npm", "root", "-g")
	if err != nil {
		return "", commandFailure("npm root -g", result, err)
	}
	root := strings.TrimSpace(result.stdout)
	if root == "" {
		return "", errors.New("npm root -g returned an empty path")
	}
	return root, nil
}

func npmGlobalPrefix(ctx context.Context, deps updateDeps) (string, error) {
	result, err := deps.capture(ctx, "npm", "prefix", "-g")
	if err != nil {
		return "", commandFailure("npm prefix -g", result, err)
	}
	prefix := strings.TrimSpace(result.stdout)
	if prefix == "" {
		return "", errors.New("npm prefix -g returned an empty path")
	}
	return prefix, nil
}

func ensureNPMVersionAvailable(ctx context.Context, deps updateDeps, version string) error {
	packageSpec := npmPackage + "@" + version
	result, err := deps.capture(
		ctx,
		"npm",
		"view",
		packageSpec,
		"version",
		"--json",
		"--registry="+npmRegistry,
	)
	if err != nil {
		return fmt.Errorf(
			"AtomGit CLI %s is not available from the official npm registry yet: %w",
			version,
			commandFailure("npm view "+packageSpec+" version", result, err),
		)
	}
	published, err := parseNPMViewVersion(result.stdout)
	if err != nil {
		return fmt.Errorf("verify npm publication for %s: %w", packageSpec, err)
	}
	if normalizeVersion(published) != normalizeVersion(version) {
		return fmt.Errorf(
			"official npm registry returned version %q for %s, expected %q",
			published,
			packageSpec,
			version,
		)
	}
	return nil
}

func parseNPMViewVersion(output string) (string, error) {
	var value any
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		return "", fmt.Errorf("parse npm view output: %w", err)
	}
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) != "" {
			return strings.TrimSpace(typed), nil
		}
	case []any:
		if len(typed) == 1 {
			if published, ok := typed[0].(string); ok && strings.TrimSpace(published) != "" {
				return strings.TrimSpace(published), nil
			}
		}
	}
	return "", errors.New("npm view returned no unambiguous version")
}

func npmLauncherPath(prefix, goos string) string {
	if goos == "windows" {
		return filepath.Join(prefix, homebrewExecutableName+".cmd")
	}
	return filepath.Join(prefix, "bin", homebrewExecutableName)
}

func inspectNPMLauncherVersion(ctx context.Context, deps updateDeps, prefix string) installedVersionVerification {
	launcher := npmLauncherPath(prefix, deps.goos)
	if deps.goos == "windows" {
		commandLine := fmt.Sprintf("\"%s\" version --json", strings.ReplaceAll(launcher, "\"", "\"\""))
		return inspectInstalledVersion(ctx, deps, launcher, "cmd.exe", "/d", "/s", "/c", commandLine)
	}
	return inspectInstalledVersion(ctx, deps, launcher, launcher, "version", "--json")
}

func homebrewPrefix(ctx context.Context, deps updateDeps, formula string) (string, error) {
	result, err := deps.capture(ctx, "brew", "--prefix", formula)
	if err != nil {
		return "", commandFailure("brew --prefix "+formula, result, err)
	}
	prefix := strings.TrimSpace(result.stdout)
	if prefix == "" {
		return "", fmt.Errorf("brew --prefix %s returned an empty path", formula)
	}
	return prefix, nil
}

func updateViaNPM(cmd *cobra.Command, deps updateDeps, latest string) error {
	version := strings.TrimPrefix(latest, "v")
	if err := ensureNPMVersionAvailable(cmd.Context(), deps, version); err != nil {
		return err
	}
	prefix, err := npmGlobalPrefix(cmd.Context(), deps)
	if err != nil {
		return fmt.Errorf("resolve npm command entry: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Updating via npm..."); err != nil {
		return err
	}
	args := []string{
		"install",
		"-g",
		npmPackage + "@" + version,
		"--registry=" + npmRegistry,
		"--no-audit",
		"--no-fund",
	}
	installErr := deps.run(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "npm", args...)
	verification := inspectNPMLauncherVersion(cmd.Context(), deps, prefix)
	repairedLauncher := false
	if installErr != nil {
		if verification.usable() {
			return fmt.Errorf(
				"update AtomGit CLI via npm: %w (the npm command entry at %s still reports %s)",
				installErr,
				verification.executable,
				normalizeVersion(verification.actual),
			)
		}
		if deps.goos != "windows" {
			return fmt.Errorf(
				"update AtomGit CLI via npm: %w; the npm command entry is no longer usable (%v); reinstall with: npm install -g %s@%s --registry=%s",
				installErr,
				verification.err,
				npmPackage,
				version,
				npmRegistry,
			)
		}
	}

	if !verification.ok(latest) {
		if deps.goos != "windows" {
			if verification.err != nil {
				return fmt.Errorf("npm update completed but command-entry verification failed: %w", verification.err)
			}
			return fmt.Errorf(
				"npm update completed but %s reports %q, expected %q",
				verification.executable,
				verification.actual,
				latest,
			)
		}

		if _, err := fmt.Fprintf(
			cmd.ErrOrStderr(),
			"npm did not leave a working AtomGit CLI %s command entry; repairing it with the verified standalone release binary.\n",
			latest,
		); err != nil {
			return err
		}
		executable, cleanupWarnings, err := repairWindowsNPMLauncher(cmd.Context(), deps, prefix, latest)
		if err != nil {
			if installErr != nil {
				return fmt.Errorf("npm update failed (%v) and command-entry repair failed: %w", installErr, err)
			}
			return fmt.Errorf("npm update did not produce a working command entry and repair failed: %w", err)
		}
		for _, warning := range cleanupWarnings {
			if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", warning); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(
			cmd.ErrOrStderr(),
			"The repaired command at %s is no longer managed by npm; a later npm reinstall can restore npm management.\n",
			executable,
		); err != nil {
			return err
		}
		repairedLauncher = true
	}
	if repairedLauncher {
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Updated AtomGit CLI to %s after repairing the npm command entry.\n", latest)
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Updated AtomGit CLI via npm to %s.\n", latest)
	return err
}

func updateViaHomebrewCore(cmd *cobra.Command, deps updateDeps, installed installation, latest string) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Updating Homebrew formulae..."); err != nil {
		return err
	}
	if err := deps.run(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "brew", "update"); err != nil {
		return fmt.Errorf("update Homebrew formulae: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Updating via Homebrew Core..."); err != nil {
		return err
	}
	if err := deps.run(
		cmd.Context(),
		cmd.InOrStdin(),
		cmd.OutOrStdout(),
		cmd.ErrOrStderr(),
		"brew",
		"upgrade",
		installed.formula,
	); err != nil {
		return fmt.Errorf("update AtomGit CLI via Homebrew Core: %w", err)
	}

	prefix, err := homebrewPrefix(cmd.Context(), deps, installed.formula)
	if err != nil {
		return fmt.Errorf("verify Homebrew Core update: %w", err)
	}
	executable := filepath.Join(prefix, "bin", homebrewExecutableName)
	if err := verifyInstalledVersion(cmd.Context(), deps, executable, latest); err != nil {
		return fmt.Errorf(
			"Homebrew Core update completed but verification failed (the Formula may not provide %s yet): %w",
			latest,
			err,
		)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Updated AtomGit CLI via Homebrew Core to %s.\n", latest)
	return err
}

func verifyInstalledVersion(ctx context.Context, deps updateDeps, executable, expected string) error {
	verification := inspectInstalledVersion(ctx, deps, executable, executable, "version", "--json")
	if verification.err != nil {
		return verification.err
	}
	if normalizeVersion(verification.actual) != normalizeVersion(expected) {
		return fmt.Errorf("installed version is %q, expected %q", verification.actual, expected)
	}
	return nil
}

func inspectInstalledVersion(
	ctx context.Context,
	deps updateDeps,
	executable string,
	name string,
	args ...string,
) installedVersionVerification {
	result, err := deps.capture(ctx, name, args...)
	if err != nil {
		return installedVersionVerification{
			executable: executable,
			err:        commandFailure(executable+" version --json", result, err),
		}
	}
	var info struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(result.stdout), &info); err != nil {
		return installedVersionVerification{
			executable: executable,
			err:        fmt.Errorf("parse installed version output from %s: %w", executable, err),
		}
	}
	if strings.TrimSpace(info.Version) == "" {
		return installedVersionVerification{
			executable: executable,
			err:        fmt.Errorf("installed version output from %s did not contain a version", executable),
		}
	}
	return installedVersionVerification{executable: executable, actual: info.Version}
}

func repairWindowsNPMLauncher(
	ctx context.Context,
	deps updateDeps,
	prefix string,
	expected string,
) (string, []string, error) {
	assetName, err := windowsReleaseAssetName(deps.goarch)
	if err != nil {
		return "", nil, err
	}
	version := normalizeVersion(expected)
	checksumsURL := fmt.Sprintf("%s/%s/checksums.txt", releaseDownloadBaseURL, version)
	archiveURL := fmt.Sprintf("%s/%s/%s", releaseDownloadBaseURL, version, assetName)
	checksums, err := deps.download(ctx, checksumsURL, maxChecksumBytes)
	if err != nil {
		return "", nil, fmt.Errorf("download %s: %w", checksumsURL, err)
	}
	wantChecksum, err := checksumForAsset(checksums, assetName)
	if err != nil {
		return "", nil, err
	}
	archive, err := deps.download(ctx, archiveURL, maxWindowsArchiveBytes)
	if err != nil {
		return "", nil, fmt.Errorf("download %s: %w", archiveURL, err)
	}
	gotChecksum := fmt.Sprintf("%x", sha256.Sum256(archive))
	if !strings.EqualFold(gotChecksum, wantChecksum) {
		return "", nil, fmt.Errorf(
			"checksum mismatch for %s: got %s, expected %s",
			assetName,
			gotChecksum,
			wantChecksum,
		)
	}
	binary, err := extractWindowsBinary(archive)
	if err != nil {
		return "", nil, fmt.Errorf("extract %s: %w", assetName, err)
	}

	if err := os.MkdirAll(prefix, 0o755); err != nil {
		return "", nil, fmt.Errorf("create npm prefix %s: %w", prefix, err)
	}
	temporary, err := os.CreateTemp(prefix, ".ag-update-*.exe")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary repaired executable: %w", err)
	}
	temporaryName := temporary.Name()
	keepTemporary := true
	defer func() {
		if keepTemporary {
			_ = os.Remove(temporaryName)
		}
	}()
	if _, err := temporary.Write(binary); err != nil {
		_ = temporary.Close()
		return "", nil, fmt.Errorf("write temporary repaired executable: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return "", nil, fmt.Errorf("sync temporary repaired executable: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", nil, fmt.Errorf("close temporary repaired executable: %w", err)
	}
	if err := verifyInstalledVersion(ctx, deps, temporaryName, expected); err != nil {
		return "", nil, fmt.Errorf("verify downloaded Windows executable: %w", err)
	}

	target := filepath.Join(prefix, homebrewExecutableName+".exe")
	launcherPaths := []string{
		filepath.Join(prefix, homebrewExecutableName),
		filepath.Join(prefix, homebrewExecutableName+".cmd"),
		filepath.Join(prefix, homebrewExecutableName+".ps1"),
		target,
	}
	backups, err := backupExistingLaunchers(prefix, launcherPaths)
	if err != nil {
		return "", nil, err
	}
	rollback := func(cause error) (string, []string, error) {
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			cause = fmt.Errorf("%w; remove failed repaired command %s: %v", cause, target, err)
		}
		return "", nil, rollbackLauncherBackups(backups, cause)
	}
	if err := os.Rename(temporaryName, target); err != nil {
		return rollback(fmt.Errorf("install repaired command at %s: %w", target, err))
	}
	keepTemporary = false
	if err := verifyInstalledVersion(ctx, deps, target, expected); err != nil {
		return rollback(fmt.Errorf("verify repaired command at %s: %w", target, err))
	}

	warnings := make([]string, 0)
	for _, backup := range backups {
		if err := os.Remove(backup.backup); err != nil && !errors.Is(err, os.ErrNotExist) {
			warnings = append(warnings, fmt.Sprintf("could not remove old npm launcher backup %s: %v", backup.backup, err))
		}
	}
	return target, warnings, nil
}

type launcherBackup struct {
	original string
	backup   string
}

func backupExistingLaunchers(prefix string, paths []string) ([]launcherBackup, error) {
	backups := make([]launcherBackup, 0, len(paths))
	for _, original := range paths {
		if _, err := os.Lstat(original); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, rollbackLauncherBackups(backups, fmt.Errorf("inspect npm launcher %s: %w", original, err))
		}
		placeholder, err := os.CreateTemp(prefix, ".ag-npm-backup-*")
		if err != nil {
			return nil, rollbackLauncherBackups(backups, fmt.Errorf("reserve npm launcher backup: %w", err))
		}
		backup := placeholder.Name()
		if err := placeholder.Close(); err != nil {
			_ = os.Remove(backup)
			return nil, rollbackLauncherBackups(backups, fmt.Errorf("close npm launcher backup placeholder: %w", err))
		}
		if err := os.Remove(backup); err != nil {
			return nil, rollbackLauncherBackups(backups, fmt.Errorf("prepare npm launcher backup %s: %w", backup, err))
		}
		if err := os.Rename(original, backup); err != nil {
			return nil, rollbackLauncherBackups(backups, fmt.Errorf("back up npm launcher %s: %w", original, err))
		}
		backups = append(backups, launcherBackup{original: original, backup: backup})
	}
	return backups, nil
}

func rollbackLauncherBackups(backups []launcherBackup, cause error) error {
	rollbackErrors := make([]string, 0)
	for index := len(backups) - 1; index >= 0; index-- {
		if err := os.Rename(backups[index].backup, backups[index].original); err != nil {
			rollbackErrors = append(rollbackErrors, err.Error())
		}
	}
	if len(rollbackErrors) == 0 {
		return cause
	}
	return fmt.Errorf("%w; restore npm launchers: %s", cause, strings.Join(rollbackErrors, "; "))
}

func windowsReleaseAssetName(goarch string) (string, error) {
	switch goarch {
	case "amd64", "arm64":
		return fmt.Sprintf("ag_windows_%s.zip", goarch), nil
	default:
		return "", fmt.Errorf("unsupported Windows release architecture %q", goarch)
	}
}

func checksumForAsset(contents []byte, assetName string) (string, error) {
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == assetName && len(fields[0]) == sha256.Size*2 {
			if _, err := hex.DecodeString(fields[0]); err == nil {
				return strings.ToLower(fields[0]), nil
			}
		}
	}
	return "", fmt.Errorf("checksums.txt does not contain a valid SHA-256 entry for %s", assetName)
}

func extractWindowsBinary(archive []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}
	for _, entry := range reader.File {
		if filepath.Base(filepath.ToSlash(entry.Name)) != homebrewExecutableName+".exe" || entry.FileInfo().IsDir() {
			continue
		}
		if entry.UncompressedSize64 > maxWindowsBinaryBytes {
			return nil, fmt.Errorf("ag.exe is larger than %d bytes", maxWindowsBinaryBytes)
		}
		file, err := entry.Open()
		if err != nil {
			return nil, err
		}
		binary, readErr := io.ReadAll(io.LimitReader(file, maxWindowsBinaryBytes+1))
		closeErr := file.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(binary) == 0 || int64(len(binary)) > maxWindowsBinaryBytes {
			return nil, fmt.Errorf("ag.exe has invalid size %d", len(binary))
		}
		return binary, nil
	}
	return nil, errors.New("ag.exe was not found in the Windows release archive")
}

func downloadUpdateAsset(ctx context.Context, rawURL string, limit int64) ([]byte, error) {
	client := &http.Client{Timeout: 2 * time.Minute}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("unexpected HTTP status %s", response.Status)
	}
	contents, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(contents)) > limit {
		return nil, fmt.Errorf("download exceeds %d bytes", limit)
	}
	return contents, nil
}

func npmExecutablePath(root, goos, goarch string) (string, error) {
	arch, err := npmArchitecture(goarch)
	if err != nil {
		return "", err
	}
	platform := goos
	if goos == "windows" {
		platform = "win32"
	}
	executable := "ag"
	if goos == "windows" {
		executable = "ag.exe"
	}
	packageName := fmt.Sprintf("atomgit-cli-%s-%s", platform, arch)
	return filepath.Join(root, "@hust-open-atom-club", packageName, "bin", executable), nil
}

func npmArchitecture(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return "x64", nil
	case "arm64", "loong64":
		return goarch, nil
	default:
		return "", fmt.Errorf("unsupported npm architecture %q", goarch)
	}
}

func canonicalPath(value string, evalSymlinks func(string) (string, error)) string {
	value = filepath.Clean(value)
	if resolved, err := evalSymlinks(value); err == nil && resolved != "" {
		return filepath.Clean(resolved)
	}
	return value
}

func samePath(left, right, goos string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if goos == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func commandFailure(command string, result commandResult, err error) error {
	detail := strings.TrimSpace(result.stderr)
	if detail == "" {
		detail = strings.TrimSpace(result.stdout)
	}
	if detail == "" {
		return fmt.Errorf("%s failed: %w", command, err)
	}
	return fmt.Errorf("%s failed: %w: %s", command, err, detail)
}
