package release

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestReleaseDeleteAsset(t *testing.T) {
	const assets = `{"tag_name":"v1/rc","assets":[{"id":"file/a b","name":"same.zip","type":"attach"},{"id":42,"name":"same.zip","type":"attach"}]}`
	for _, tt := range []struct {
		name, body, id, input   string
		yes                     bool
		getStatus, deleteStatus int
		deleteErr               bool
		wantDeletes             int
		wantErr                 string
	}{
		{name: "opaque ID and duplicate names", body: assets, id: "file/a b", input: "yes\n", wantDeletes: 1},
		{name: "numeric ID and --yes", body: assets, id: "42", yes: true, wantDeletes: 1},
		{name: "cancel", body: assets, id: "42", input: "n\n"},
		{name: "noninteractive EOF", body: assets, id: "42"},
		{name: "EOF without newline", body: assets, id: "42", input: "yes", wantDeletes: 1},
		{name: "missing ID", body: assets, id: "other", yes: true, wantErr: "does not belong"},
		{name: "wrong tag", body: `{"tag_name":"other","assets":[{"id":42,"type":"attach"}]}`, id: "42", yes: true, wantErr: "tag mismatch"},
		{name: "source archive", body: `{"tag_name":"v1/rc","assets":[{"id":42,"type":"source"}]}`, id: "42", yes: true, wantErr: "not an uploaded attachment"},
		{name: "ambiguous ID", body: `{"tag_name":"v1/rc","assets":[{"id":42,"type":"attach"},{"id":"42","type":"attach"}]}`, id: "42", yes: true, wantErr: "multiple assets"},
		{name: "read 404", getStatus: 404, body: `{"message":"missing"}`, id: "42", yes: true, wantErr: "failed to get release"},
		{name: "read malformed", body: `{`, id: "42", yes: true, wantErr: "failed to get release"},
		{name: "delete 404", body: assets, id: "42", yes: true, deleteStatus: 404, wantDeletes: 1, wantErr: "failed to delete"},
		{name: "delete forbidden", body: assets, id: "42", yes: true, deleteStatus: 403, wantDeletes: 1, wantErr: "failed to delete"},
		{name: "unexpected success status", body: assets, id: "42", yes: true, deleteStatus: 200, wantDeletes: 1, wantErr: "failed to delete"},
		{name: "server error no retry", body: assets, id: "42", yes: true, deleteStatus: 503, wantDeletes: 1, wantErr: "failed to delete"},
		{name: "lost response no retry", body: assets, id: "42", yes: true, deleteErr: true, wantDeletes: 1, wantErr: "response lost"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gets, deletes := 0, 0
			transport := releaseRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodGet {
					gets++
					if r.URL.EscapedPath() != "/api/v5/repos/alice/demo/releases/tags/v1%2Frc" {
						t.Fatalf("GET path = %s", r.URL.EscapedPath())
					}
					status := tt.getStatus
					if status == 0 {
						status = 200
					}
					return releaseResponse(status, tt.body), nil
				}
				if r.Method != http.MethodDelete {
					t.Fatalf("method = %s", r.Method)
				}
				deletes++
				suffix := "42"
				if tt.id == "file/a b" {
					suffix = "file%2Fa%20b"
				}
				if r.URL.EscapedPath() != "/api/v5/repos/alice/demo/releases/v1%2Frc/attach_files/"+suffix {
					t.Fatalf("DELETE path = %s", r.URL.EscapedPath())
				}
				if r.Body != nil {
					t.Fatal("unexpected DELETE body")
				}
				if tt.deleteErr {
					return nil, errors.New("response lost")
				}
				status := tt.deleteStatus
				if status == 0 {
					status = 204
				}
				return releaseResponse(status, ""), nil
			})
			cmd := NewCmdRelease(releaseTestFactory(transport))
			var out, prompt bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&prompt)
			cmd.SetIn(strings.NewReader(tt.input))
			args := []string{"delete-asset", "v1/rc", tt.id}
			if tt.yes {
				args = append(args, "--yes")
			}
			cmd.SetArgs(args)
			err := cmd.Execute()
			if tt.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("error = %v, want %s", err, tt.wantErr)
			}
			if gets != 1 || deletes != tt.wantDeletes {
				t.Fatalf("GET/DELETE = %d/%d", gets, deletes)
			}
			if tt.wantErr == "" && !tt.yes {
				for _, part := range []string{"alice/demo", "v1/rc", tt.id, "same.zip"} {
					if !strings.Contains(prompt.String(), part) {
						t.Fatalf("prompt lacks %q: %s", part, prompt.String())
					}
				}
			}
			if tt.wantErr == "" && deletes == 0 && !strings.Contains(out.String(), "cancelled") {
				t.Fatalf("output = %s", out.String())
			}
		})
	}
}

func TestReleaseDeleteAssetValidationBeforeRequest(t *testing.T) {
	for _, args := range [][]string{
		{}, {"v1"}, {"alice/demo", "v1", "42", "extra"}, {"alice/demo", " ", "42"}, {"alice/demo", "v1", " "}, {"invalid/repo/name", "v1", "42"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			cmd := NewCmdRelease(releaseTestFactory(releaseRoundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected request"); return nil, nil })))
			cmd.SetArgs(append([]string{"delete-asset"}, args...))
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected argument error")
			}
		})
	}
}

func TestReleaseDeleteAssetExplicitRepository(t *testing.T) {
	calls := 0
	cmd := NewCmdRelease(releaseTestFactory(releaseRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if !strings.HasPrefix(r.URL.EscapedPath(), "/api/v5/repos/bob/project/") {
			t.Fatalf("path = %s", r.URL.EscapedPath())
		}
		if r.Method == http.MethodGet {
			return releaseResponse(200, `{"tag_name":"v1","assets":[{"id":42,"type":"attach","name":"a.zip"}]}`), nil
		}
		return releaseResponse(204, ""), nil
	})))
	cmd.SetArgs([]string{"delete-asset", "bob/project", "v1", "42", "--yes"})
	cmd.SetOut(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("requests = %d", calls)
	}
}
