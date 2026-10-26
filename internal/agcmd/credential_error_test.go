package agcmd

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

func TestCredentialFormatStartupDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		detail string
	}{
		{"syntax", `{secret`, "The credential file contains invalid JSON at byte 2."},
		{"legacy field type", `{"access_token":123,"user":"private-user"}`, "The credential file has an invalid field type."},
		{"account field type", `{"version":2,"active":"alice","accounts":[{"user":"alice","access_token":123}]}`, "The credential file has an invalid field type."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("XDG_CONFIG_HOME", "")
			path, err := config.PrimaryTokenPath()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			factory := &cmdutil.Factory{}
			err = loadCommandConfig(nil, factory, config.NewConfig)
			if err == nil || factory.Config != nil {
				t.Fatal("malformed credentials must stop configuration loading")
			}
			err = fmt.Errorf("failed to load config: %w", err)
			for _, raw := range []bool{false, true} {
				var out bytes.Buffer
				writer := cmdutil.NewSanitizingWriter(&out)
				if err := writer.SetRaw(raw); err != nil {
					t.Fatal(err)
				}
				renderCommandError(writer, err)
				if err := cmdutil.FlushWriter(writer); err != nil {
					t.Fatal(err)
				}
				want := "Error: Credential configuration is invalid.\n\n" + tc.detail +
					"\n\nCheck the credential file syntax and field types.\n"
				if out.String() != want {
					t.Fatalf("raw=%t: output = %q, want %q", raw, out.String(), want)
				}
			}
		})
	}
}

func TestCredentialFormatDiagnosticKeepsCausePrivate(t *testing.T) {
	cause := errors.New("private-token-value in /private/credential/path")
	err := fmt.Errorf("private wrapper: %w", &config.CredentialFormatError{InvalidFieldType: true, Err: cause})
	var out bytes.Buffer
	renderCommandError(&out, err)
	want := "Error: Credential configuration is invalid.\n\nThe credential file has an invalid field type.\n\n" +
		"Check the credential file syntax and field types.\n"
	if out.String() != want || !errors.Is(err, cause) {
		t.Fatalf("output = %q, want safe diagnostic with cause retained", out.String())
	}
}
