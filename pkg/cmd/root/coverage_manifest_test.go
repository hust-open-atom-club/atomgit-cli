package root

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

const coverageStart = "<!-- coverage:start -->"
const coverageEnd = "<!-- coverage:end -->"

var coverageLink = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)

// Check the hand-maintained manifest against the actual tree without executing
// commands, reading credentials, or making requests. Endpoint semantics and
// subcommand scope still require review; registration is not API verification.
func TestOpenAPICoverageManifest(t *testing.T) {
	root, err := NewCmdRoot(&cmdutil.Factory{})
	if err != nil {
		t.Fatal(err)
	}
	var families []string
	for _, cmd := range root.Commands() {
		families = append(families, cmd.Name())
	}
	repository, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(repository, "docs", "openapi-coverage.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateCoverageManifest(string(data), families, repository); err != nil {
		t.Fatal(err)
	}
}

func validateCoverageManifest(document string, families []string, repository string) error {
	if strings.Count(document, coverageStart) != 1 || strings.Count(document, coverageEnd) != 1 {
		return fmt.Errorf("coverage table requires one start and end marker")
	}
	start := strings.Index(document, coverageStart) + len(coverageStart)
	end := strings.Index(document, coverageEnd)
	if end < start {
		return fmt.Errorf("coverage markers are reversed")
	}
	want := make(map[string]bool, len(families))
	for _, name := range families {
		want[name] = true
	}
	seen := make(map[string]bool)
	for _, line := range strings.Split(document[start:end], "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "### ") {
			continue
		}
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			return fmt.Errorf("malformed coverage row: %s", line)
		}
		cells := strings.Split(strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|"), "|")
		if len(cells) != 8 {
			return fmt.Errorf("coverage row has %d columns, want 8: %s", len(cells), line)
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
			if cells[i] == "" {
				return fmt.Errorf("empty coverage field %d: %s", i+1, line)
			}
		}
		if cells[0] == "命令族" || cells[0] == "---" {
			continue
		}
		name := cells[0]
		if !want[name] {
			return fmt.Errorf("unregistered command family %q", name)
		}
		if seen[name] {
			return fmt.Errorf("duplicate command family %q", name)
		}
		seen[name] = true
		if strings.Contains(cells[3], ",") || strings.Contains(cells[4], ",") {
			return fmt.Errorf("coverage status and owner must be singular for %s", name)
		}
		for _, field := range []struct {
			index   int
			allowed string
		}{
			{1, "v5,v8,oauth,external,local"},
			{3, "implemented,partial,missing,deferred,unverified"},
			{4, "project,partner"},
			{5, "source,mock,local,docs,live"},
		} {
			for _, value := range strings.Split(cells[field.index], ",") {
				if !strings.Contains(","+field.allowed+",", ","+value+",") || value == "" {
					return fmt.Errorf("invalid coverage field for %s: %q", name, cells[field.index])
				}
			}
		}
		if !coverageLink.MatchString(cells[6]) {
			return fmt.Errorf("missing evidence link for %s", name)
		}
	}
	for _, name := range families {
		if !seen[name] {
			return fmt.Errorf("missing command family %q", name)
		}
	}
	for _, match := range coverageLink.FindAllStringSubmatch(document, -1) {
		target := match[1]
		parsed, err := url.Parse(target)
		if err != nil {
			return fmt.Errorf("invalid coverage link %q: %w", target, err)
		}
		if parsed.Scheme != "" {
			if (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
				return fmt.Errorf("unsupported coverage URL %q", target)
			}
			continue // External availability is deliberately not an offline test.
		}
		if parsed.Host != "" || filepath.IsAbs(parsed.Path) || parsed.Path == "" {
			return fmt.Errorf("coverage link must be repository-relative: %q", target)
		}
		path := filepath.Join(repository, "docs", filepath.FromSlash(parsed.Path))
		rel, err := filepath.Rel(repository, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("coverage link escapes repository: %q", target)
		}
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("broken coverage link %q: %w", target, err)
		}
	}
	return nil
}

func TestOpenAPICoverageValidation(t *testing.T) {
	repository := t.TempDir()
	if err := os.Mkdir(filepath.Join(repository, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "docs", "evidence.md"), []byte("evidence"), 0o644); err != nil {
		t.Fatal(err)
	}
	row := "| version | local | version metadata | implemented | project | source,local | [evidence](evidence.md) | local-only |\n"
	valid := coverageStart + "\n" + row + coverageEnd
	for _, tt := range []struct {
		name     string
		document string
		families []string
		want     string
	}{
		{"valid", valid, []string{"version"}, ""},
		{"missing marker", row, []string{"version"}, "marker"},
		{"reversed markers", coverageEnd + row + coverageStart, []string{"version"}, "reversed"},
		{"duplicate marker", valid + coverageEnd, []string{"version"}, "marker"},
		{"missing family", valid, []string{"version", "new-command"}, "missing command"},
		{"stale family", valid, []string{"new-command"}, "unregistered"},
		{"duplicate family", strings.Replace(valid, row, row+row, 1), []string{"version"}, "duplicate command"},
		{"column count", strings.Replace(valid, " | local-only", "", 1), []string{"version"}, "columns"},
		{"empty field", strings.Replace(valid, "version metadata", "", 1), []string{"version"}, "empty"},
		{"invalid protocol", strings.Replace(valid, "| local |", "| v9 |", 1), []string{"version"}, "invalid coverage field"},
		{"invalid status", strings.Replace(valid, "implemented", "complete", 1), []string{"version"}, "invalid coverage field"},
		{"invalid owner", strings.Replace(valid, "project", "unknown", 1), []string{"version"}, "invalid coverage field"},
		{"multiple statuses", strings.Replace(valid, "implemented", "implemented,missing", 1), []string{"version"}, "singular"},
		{"multiple owners", strings.Replace(valid, "project", "project,partner", 1), []string{"version"}, "singular"},
		{"invalid verification", strings.Replace(valid, "source,local", "verified", 1), []string{"version"}, "invalid coverage field"},
		{"missing evidence", strings.Replace(valid, "[evidence](evidence.md)", "none", 1), []string{"version"}, "missing evidence"},
		{"broken link", strings.Replace(valid, "(evidence.md)", "(missing.md)", 1), []string{"version"}, "broken coverage link"},
		{"escaping link", strings.Replace(valid, "(evidence.md)", "(../../outside.md)", 1), []string{"version"}, "escapes repository"},
		{"absolute link", strings.Replace(valid, "(evidence.md)", "(/tmp/evidence.md)", 1), []string{"version"}, "repository-relative"},
		{"unexpected line", strings.Replace(valid, row, "bad row\n", 1), []string{"version"}, "malformed"},
		{"external URL", strings.Replace(valid, "(evidence.md)", "(https://example.test/evidence)", 1), []string{"version"}, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCoverageManifest(tt.document, tt.families, repository)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}
