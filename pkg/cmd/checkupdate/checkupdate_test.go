package checkupdate

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	internalversion "atomgit.com/hust-open-atom-club/atomgit-cli/internal/version"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSelectLatestStableRelease_whenReleasesContainIneligibleEntries(t *testing.T) {
	// Given
	releases := []api.Release{
		{TagName: "v2.0.0", Prerelease: true},
		{TagName: "v1.9.0", ReleaseStatus: api.ReleaseStatusPre},
		{TagName: "not-semver"},
		{TagName: "v1.3.0"},
		{TagName: "v1.10.0"},
		{TagName: "v1.4.0", Draft: true},
	}

	// When
	got, err := selectLatestStableRelease(releases)

	// Then
	if err != nil {
		t.Fatalf("selectLatestStableRelease() error = %v", err)
	}
	if got != "v1.10.0" {
		t.Fatalf("selectLatestStableRelease() = %q, want %q", got, "v1.10.0")
	}
}

func TestSelectLatestStableRelease_whenNoStableReleaseExists(t *testing.T) {
	// Given
	releases := []api.Release{
		{TagName: "v2.0.0", Draft: true},
		{TagName: "v1.9.0-rc.1"},
		{TagName: "not-semver"},
	}

	// When
	_, err := selectLatestStableRelease(releases)

	// Then
	if err == nil || !strings.Contains(err.Error(), "no stable AtomGit CLI release found") {
		t.Fatalf("selectLatestStableRelease() error = %v", err)
	}
}

func TestCompareVersions_whenCurrentVersionVaries(t *testing.T) {
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
			// Given / When
			got, err := compareVersions(tt.current, tt.latest)

			// Then
			if err != nil {
				t.Fatalf("compareVersions() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("compareVersions() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompareVersions_whenCurrentVersionIsNotComparable(t *testing.T) {
	tests := []string{"dev", "unknown", "v1.2.3-dirty", "f606e04"}

	for _, current := range tests {
		t.Run(current, func(t *testing.T) {
			// Given / When
			_, err := compareVersions(current, "v1.3.0")

			// Then
			if err == nil || !strings.Contains(err.Error(), "not a comparable semantic version") {
				t.Fatalf("compareVersions() error = %v", err)
			}
		})
	}
}

func TestNewCmdCheckUpdate_whenNewerReleaseExists(t *testing.T) {
	// Given
	oldVersion := internalversion.Version
	internalversion.Version = "v1.2.3"
	t.Cleanup(func() { internalversion.Version = oldVersion })

	var authorization string
	factory := &cmdutil.Factory{
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				authorization = req.Header.Get("Authorization")
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
				if req.Body != nil && req.Body != http.NoBody {
					t.Fatal("request body is not empty")
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(
						`[{"tag_name":"v1.3.0"},{"tag_name":"v1.2.4-rc.1","prerelease":true}]`,
					)),
					Request: req,
				}, nil
			})}, nil
		},
	}
	var out bytes.Buffer
	cmd := NewCmdCheckUpdate(factory)
	cmd.SetOut(&out)

	// When
	err := cmd.Execute()

	// Then
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if authorization != "" {
		t.Fatalf("Authorization = %q, want empty", authorization)
	}
	want := "Current version: v1.2.3\nLatest release: v1.3.0\nStatus: update available\n"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestNewCmdCheckUpdate_whenCurrentVersionIsNotComparable(t *testing.T) {
	// Given
	oldVersion := internalversion.Version
	internalversion.Version = "dev"
	t.Cleanup(func() { internalversion.Version = oldVersion })

	requested := false
	factory := &cmdutil.Factory{
		HttpClient: func() (*http.Client, error) {
			requested = true
			return &http.Client{}, nil
		},
	}
	cmd := NewCmdCheckUpdate(factory)

	// When
	err := cmd.Execute()

	// Then
	if err == nil || !strings.Contains(err.Error(), "not a comparable semantic version") {
		t.Fatalf("Execute() error = %v", err)
	}
	if requested {
		t.Fatal("HTTP client was requested for an invalid local version")
	}
}

func TestNewCmdCheckUpdate_Help(t *testing.T) {
	// Given
	var out bytes.Buffer
	cmd := NewCmdCheckUpdate(&cmdutil.Factory{})
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})

	// When
	err := cmd.Execute()

	// Then
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, want := range []string{
		"Check for a newer AtomGit CLI release",
		"check-update",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("help does not contain %q:\n%s", want, out.String())
		}
	}
}
