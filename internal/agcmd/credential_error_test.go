package agcmd

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

const missingCredentialOutput = "Error: No credentials are configured.\n\nRun `ag-cli auth login` to log in.\n"

func TestMissingCredentialDiagnostics(t *testing.T) {
	privateCause := errors.New("token=private-token; searched /private/credential/path")
	for name, err := range map[string]error{
		"missing": config.ErrNotAuthenticated,
		"wrapped": fmt.Errorf("lookup: %w; cause: %w", config.ErrNotAuthenticated, privateCause),
	} {
		t.Run(name, func(t *testing.T) {
			before := err.Error()
			var out bytes.Buffer
			renderCommandError(&out, err)
			if out.String() != missingCredentialOutput {
				t.Fatalf("output = %q, want %q", out.String(), missingCredentialOutput)
			}
			if err.Error() != before || (name == "wrapped" && !errors.Is(err, privateCause)) {
				t.Fatal("presentation changed the error chain")
			}
		})
	}
}

func TestMissingCredentialLookupThroughFactory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatal(err)
	}
	factory := &cmdutil.Factory{
		Config: cfg,
		HttpClient: func() (*http.Client, error) {
			t.Fatal("missing credentials must stop before creating an HTTP client")
			return nil, nil
		},
	}
	for _, name := range []string{"API", "Actions"} {
		t.Run(name, func(t *testing.T) {
			var err error
			if name == "API" {
				_, err = factory.AuthenticatedAPIClient()
			} else {
				_, err = factory.AuthenticatedActionsClient()
			}
			if !errors.Is(err, config.ErrNotAuthenticated) || !errors.Is(err, config.ErrTokenNotFound) {
				t.Fatalf("error = %v, want missing credentials and token lookup causes", err)
			}
			var out bytes.Buffer
			renderCommandError(&out, err)
			if out.String() != missingCredentialOutput {
				t.Fatalf("output = %q, want %q", out.String(), missingCredentialOutput)
			}
		})
	}
}
