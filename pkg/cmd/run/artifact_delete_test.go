package run

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func TestRunArtifactDeleteConfirmsAndDeletes(t *testing.T) {
	var methods []string
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		methods = append(methods, req.Method)
		if req.URL.Path != "/api/v8/repos/team/demo/actions/artifacts/artifact-1" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		switch req.Method {
		case http.MethodGet:
			return runResponse(req, http.StatusOK, `{"id":"artifact-1","name":"logs","workflow_run_id":"run-42","expires_at":1787587200000}`), nil
		case http.MethodDelete:
			return runResponse(req, http.StatusNoContent, ""), nil
		default:
			t.Fatalf("method = %s", req.Method)
			return nil, nil
		}
	})

	cmd := newCmdRunArtifactDelete(runFactory(runTestConfig{token: "secret"}, transport))
	var out, errOut bytes.Buffer
	cmd.SetIn(strings.NewReader("yes\n"))
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"team/demo", "artifact-1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if strings.Join(methods, ",") != "GET,DELETE" {
		t.Fatalf("methods = %v", methods)
	}
	for _, value := range []string{
		"Repository: team/demo",
		"Artifact ID: artifact-1",
		"Name: logs",
		"Workflow run: run-42",
		"Expires:",
		"cannot be undone",
		"[y/N]",
	} {
		if !strings.Contains(errOut.String(), value) {
			t.Fatalf("confirmation missing %q:\n%s", value, errOut.String())
		}
	}
	if out.String() != "Deleted artifact logs (artifact-1) from team/demo\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunArtifactDeleteCancellationDoesNotDelete(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "no", input: "no\n"},
		{name: "empty", input: "\n"},
		{name: "other", input: "continue\n"},
		{name: "EOF", input: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method == http.MethodDelete {
					t.Fatal("DELETE was sent after cancellation")
				}
				return runResponse(req, http.StatusOK, `{"id":"artifact-1","name":"logs"}`), nil
			})
			cmd := newCmdRunArtifactDelete(runFactory(runTestConfig{token: "secret"}, transport))
			var out bytes.Buffer
			cmd.SetIn(strings.NewReader(tt.input))
			cmd.SetOut(&out)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"team/demo", "artifact-1"})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if requests != 1 {
				t.Fatalf("requests = %d, want metadata only", requests)
			}
			if out.String() != "Artifact deletion cancelled\n" {
				t.Fatalf("output = %q", out.String())
			}
		})
	}
}

func TestRunArtifactDeleteYesStillReadsMetadata(t *testing.T) {
	var methods []string
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		methods = append(methods, req.Method)
		if req.URL.Path != "/api/v8/repos/inferred/repo/actions/artifacts/artifact-1" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		if req.Method == http.MethodGet {
			return runResponse(req, http.StatusOK, `{"id":"artifact-1","name":"logs"}`), nil
		}
		return runResponse(req, http.StatusNoContent, ""), nil
	})
	factory := runFactory(runTestConfig{token: "secret"}, transport)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{Owner: "inferred", Name: "repo"}, nil
	}
	cmd := newCmdRunArtifactDelete(factory)
	var out, errOut bytes.Buffer
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"artifact-1", "--yes"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(methods, ",") != "GET,DELETE" {
		t.Fatalf("methods = %v", methods)
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected prompt = %q", errOut.String())
	}
	if out.String() != "Deleted artifact logs (artifact-1) from inferred/repo\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunArtifactDeleteValidatesBeforeAuthentication(t *testing.T) {
	cmd := newCmdRunArtifactDelete(runFactory(runTestConfig{tokenErr: errors.New("authentication should not be reached")}, nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"team/demo", "   "})
	err := cmd.Execute()
	if err == nil || err.Error() != "artifact ID is required" {
		t.Fatalf("error = %v", err)
	}
}

func TestRunArtifactDeleteResolverErrorPrecedesAuthentication(t *testing.T) {
	resolverErr := errors.New("repository unavailable")
	factory := runFactory(runTestConfig{tokenErr: errors.New("authentication should not be reached")}, nil)
	factory.RepositoryResolver = func() (cmdutil.Repository, error) {
		return cmdutil.Repository{}, resolverErr
	}
	cmd := newCmdRunArtifactDelete(factory)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"artifact-1"})
	err := cmd.Execute()
	if !errors.Is(err, resolverErr) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunArtifactDeleteReportsMetadataAndDeleteErrors(t *testing.T) {
	t.Run("metadata not found", func(t *testing.T) {
		requests := 0
		transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			if req.Method == http.MethodDelete {
				t.Fatal("DELETE was sent without metadata")
			}
			return runResponse(req, http.StatusNotFound, `{"message":"artifact not found"}`), nil
		})
		cmd := newCmdRunArtifactDelete(runFactory(runTestConfig{token: "secret"}, transport))
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{"team/demo", "missing", "--yes"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to get artifact missing") || !strings.Contains(err.Error(), "404") {
			t.Fatalf("error = %v", err)
		}
		if requests != 1 {
			t.Fatalf("requests = %d", requests)
		}
	})

	for _, status := range []int{http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					return runResponse(req, http.StatusOK, `{"id":"artifact-1","name":"logs"}`), nil
				}
				return runResponse(req, status, `{"message":"delete failed"}`), nil
			})
			cmd := newCmdRunArtifactDelete(runFactory(runTestConfig{token: "secret"}, transport))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"team/demo", "artifact-1", "--yes"})
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "failed to delete artifact artifact-1") || !strings.Contains(err.Error(), http.StatusText(status)) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRunArtifactDeleteSanitizesRemoteMetadata(t *testing.T) {
	transport := runRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return runResponse(req, http.StatusOK, `{"id":"artifact-1","name":"logs\ninjected\tname","workflow_run_id":"run-1\rspoof"}`), nil
		}
		return runResponse(req, http.StatusNoContent, ""), nil
	})
	cmd := newCmdRunArtifactDelete(runFactory(runTestConfig{token: "secret"}, transport))
	var out, errOut bytes.Buffer
	cmd.SetIn(strings.NewReader("y\n"))
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"team/demo", "artifact-1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "Name: logs injected name") || !strings.Contains(errOut.String(), "Workflow run: run-1 spoof") {
		t.Fatalf("confirmation = %q", errOut.String())
	}
	if strings.Contains(errOut.String(), "\ninjected") || strings.Contains(out.String(), "\ninjected") {
		t.Fatalf("control characters leaked: prompt=%q output=%q", errOut.String(), out.String())
	}
	if out.String() != "Deleted artifact logs injected name (artifact-1) from team/demo\n" {
		t.Fatalf("output = %q", out.String())
	}
}
