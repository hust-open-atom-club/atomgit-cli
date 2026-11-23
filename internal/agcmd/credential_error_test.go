package agcmd

import (
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api"
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestCredentialSaveDiagnostics(t *testing.T) {
	const path = "/fixture/token.json"
	save := &config.CredentialSaveError{
		Stage:     config.CredentialSaveWrite,
		Path:      path,
		Err:       fmt.Errorf("private cause: %w", os.ErrPermission),
		CloseErr:  errors.New("private close cause"),
		RemoveErr: errors.New("private remove cause"),
	}
	var out bytes.Buffer
	renderCommandError(&out, fmt.Errorf("save account: %w", save))
	want := "Error: Unable to save credentials.\n\nFailed to write temporary file.\n" +
		"Path: \"/fixture/token.json\".\nClosing the temporary credential file also failed.\n" +
		"Removing the temporary credential file also failed. It may still be present.\n\n" +
		"Check ownership and write permissions for the affected path and its parent directory.\n"
	if out.String() != want {
		t.Fatalf("diagnostic = %q, want %q", out.String(), want)
	}
}

func TestCredentialSaveDiagnosticKeepsGitRollbackFailure(t *testing.T) {
	save := &config.CredentialSaveError{
		Stage: config.CredentialSaveReplace,
		Err:   os.ErrPermission,
	}
	rollback := errors.New("Git rollback fixture failure")
	err := fmt.Errorf("switch account: %w; additionally failed to restore Git identity: %w", save, rollback)
	var out bytes.Buffer
	renderCommandError(&out, err)
	for _, message := range []string{"replace file", "failed to restore Git identity", rollback.Error()} {
		if !strings.Contains(out.String(), message) {
			t.Fatalf("diagnostic = %q, missing %q", out.String(), message)
		}
	}
}

func TestCredentialAccessDiagnostics(t *testing.T) {
	private := errors.New("private-token and private-cause")
	for _, tc := range []struct {
		name   string
		access *config.CredentialAccessError
		detail string
		hint   string
	}{
		{"unknown cause", &config.CredentialAccessError{Operation: "private-operation", Path: "/credential", Err: private},
			"Credential file: \"/credential\".", ""},
		{"permission", &config.CredentialAccessError{Operation: "open", Path: "/credential", Err: fmt.Errorf("private wrapper: %w", os.ErrPermission)},
			"Credential file: \"/credential\".", "Check the file and parent directory ownership and read permissions."},
		{"no path", &config.CredentialAccessError{Operation: "locate credential paths", Err: private},
			"No credential file paths are available.", "Check the home directory and XDG_CONFIG_HOME settings."},
		{"inner role", &config.CredentialAccessError{Path: "/credential", Err: fmt.Errorf("%w: %w", config.ErrNotAuthenticated, private)},
			"Credential file: \"/credential\".", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := fmt.Errorf("private wrapper: %w", tc.access)
			want := "Error: Unable to read credentials.\n\n" + tc.detail + "\n"
			if tc.hint != "" {
				want += "\n" + tc.hint + "\n"
			}
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
				if out.String() != want {
					t.Fatalf("raw=%t: output = %q, want %q", raw, out.String(), want)
				}
			}
			if !errors.Is(err, config.ErrCredentialAccess) || !errors.Is(err, tc.access.Err) {
				t.Fatal("presentation lost original cause")
			}
		})
	}
}

func TestCredentialAccessCauseDoesNotReplaceOuterRole(t *testing.T) {
	access := &config.CredentialAccessError{Err: errors.New("private-cause")}
	for _, tc := range []struct {
		name    string
		err     error
		summary string
	}{
		{"format", &config.CredentialFormatError{Err: access}, "Credential configuration is invalid."},
		{"API", &api.ResponseReadError{Err: access}, "API request failed."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			renderCommandError(&out, tc.err)
			if !strings.HasPrefix(out.String(), "Error: "+tc.summary+"\n") || strings.Contains(out.String(), "Unable to read credentials") {
				t.Fatalf("inner access failure replaced outer operation: %q", out.String())
			}
		})
	}
}

func TestCredentialFileSafetyDiagnostics(t *testing.T) {
	ownerHint := "Check ownership and file read permissions."
	if runtime.GOOS != "windows" {
		ownerHint = "Check ownership and set the credential file permissions to 0600."
	}
	for _, tc := range []struct {
		name         string
		reason       error
		detail, hint string
	}{
		{"symlink", config.ErrTokenFileSymlink, "The credential file is a symbolic link.", "Use a regular credential file owned by the current user."},
		{"changed", config.ErrTokenFileChanged, "The credential file changed while it was being opened.", "Check the credential file before retrying."},
		{"owner unreadable", &config.TokenPermissionError{Path: "/private/credential", Err: config.ErrTokenFileUnreadable}, "The credential file is not readable by its owner.", ownerHint},
		{"nonregular", config.ErrTokenFileNotRegular, "The credential file is not a regular file.", "Use a regular credential file owned by the current user."},
		{"oversized", config.ErrTokenFileTooLarge, "The credential file exceeds the size limit.", "Check the credential file size; credentials are limited to 1 MiB."},
		{"private cause", errors.New("private-token and private-cause"), "The credential file failed a safety check.", "Check the credential file before retrying."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := fmt.Errorf("private wrapper: %w", &config.CredentialFileSafetyError{Path: "/private/credential", Err: tc.reason})
			want := "Error: Credential configuration is invalid.\n\n" + tc.detail + "\n\n" + tc.hint + "\n"
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
				if out.String() != want {
					t.Fatalf("raw=%t: output = %q, want %q", raw, out.String(), want)
				}
			}
			if !errors.Is(err, tc.reason) {
				t.Fatal("diagnostic changed original cause")
			}
		})
	}
}

func TestCredentialSafetyAndOuterOperationRoles(t *testing.T) {
	safety := &config.CredentialFileSafetyError{Err: config.ErrTokenFileSymlink}
	for _, tc := range []struct {
		name   string
		err    error
		detail string
	}{
		{"API owns safety cause", &api.ResponseReadError{Err: safety}, "The server response could not be read."},
		{"access owns safety cause", &config.CredentialAccessError{Path: "/credential", Err: safety}, "Unable to read credentials."},
		{"format owns safety cause", &config.CredentialFormatError{InvalidFieldType: true, Err: safety}, "The credential file has an invalid field type."},
		{"safety owns access cause", &config.CredentialFileSafetyError{Err: &config.CredentialAccessError{Err: os.ErrPermission}}, "The credential file failed a safety check."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			renderCommandError(&out, tc.err)
			if !strings.Contains(out.String(), tc.detail) || strings.Contains(out.String(), "The credential file is a symbolic link.") {
				t.Fatalf("inner cause replaced outer operation: %q", out.String())
			}
		})
	}
}

func TestInvalidCredentialDataStopsFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatal(err)
	}
	path, err := config.PrimaryTokenPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"version":2,"active":"private-account","accounts":[{"user":"alice","access_token":"fixture"}]}`)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	legacy := []byte(`{"user":"bob","access_token":"legacy"}`)
	if err := os.WriteFile(filepath.Join(home, ".atomgit_personal_token.json"), legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	factory := &cmdutil.Factory{}
	err = loadCommandConfig(nil, factory, config.NewConfig)
	if !errors.Is(err, config.ErrInvalidCredentials) || factory.Config != nil {
		t.Fatalf("startup must reject invalid primary credentials: %v", err)
	}
	var out bytes.Buffer
	renderCommandError(&out, fmt.Errorf("failed to load config: %w", err))
	want := "Error: Credential configuration is invalid.\n\nThe stored credential data failed validation.\n\n" +
		"Check the credential store version, account records, and active account.\n"
	if out.String() != want {
		t.Fatalf("diagnostic = %q, want %q", out.String(), want)
	}
	factory.Config = cfg
	factory.HttpClient = func() (*http.Client, error) {
		t.Fatal("invalid credentials must stop before creating an anonymous client")
		return nil, nil
	}
	client, err := factory.OptionalAPIClient()
	if client != nil || !errors.Is(err, config.ErrInvalidCredentials) || errors.Is(err, config.ErrNotAuthenticated) {
		t.Fatalf("optional authentication must reject invalid credentials: %v", err)
	}
}

func TestCredentialPermissionRepairDiagnostics(t *testing.T) {
	const path = "/fixture/token.json"
	for _, tc := range []struct {
		name  string
		cause error
		hint  string
	}{
		{"permission denied", os.ErrPermission, "\nCheck the file ownership and permission to change its mode.\n"},
		{"closed file", os.ErrClosed, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := &config.CredentialPermissionRepairError{
				Path: path,
				Err:  &os.PathError{Op: "chmod", Path: path, Err: fmt.Errorf("private cause: %w", tc.cause)},
			}
			var out bytes.Buffer
			renderCommandError(&out, fmt.Errorf("failed to load config: %w", err))
			want := "Error: Unable to secure the credential file.\n\nCredential file: \"/fixture/token.json\".\n" + tc.hint
			if out.String() != want {
				t.Fatalf("diagnostic = %q, want %q", out.String(), want)
			}
		})
	}
}

func TestCredentialAccountDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"missing account", &config.AccountNotFoundError{Account: "private-account"}, "Error: The requested account is not saved.\n\nRun `ag-cli auth list` to view saved accounts.\n"},
		{"rename conflict", &config.AccountConflictError{Account: "private-account", ExistingAccount: "private-other"}, "Error: Cannot rename the active account.\n\nAn account with the new name already exists.\n\nRun `ag-cli auth list` to view saved accounts.\n"},
		{"active removal", &config.ActiveAccountRemovalError{Account: "private-account"}, "Error: Cannot remove the active account.\n\nOther saved accounts remain.\n\nRun `ag-cli auth switch <account>` before removing this account.\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, wrapper := range []struct {
				name string
				err  error
			}{
				{name: "wrapped", err: fmt.Errorf("account operation: %w", tc.err)},
				{name: "single joined cause", err: errors.Join(tc.err)},
			} {
				t.Run(wrapper.name, func(t *testing.T) {
					var out bytes.Buffer
					renderCommandError(&out, wrapper.err)
					if out.String() != tc.want {
						t.Fatalf("diagnostic = %q, want %q", out.String(), tc.want)
					}
				})
			}
		})
	}
}

func TestCredentialAccountDiagnosticPreservesRollbackFailure(t *testing.T) {
	missing := &config.AccountNotFoundError{Account: "alice"}
	rollback := errors.New("failed to write previous identity")
	err := fmt.Errorf("switch account: %w; additionally failed to restore Git identity: %w", missing, rollback)
	var out bytes.Buffer
	renderCommandError(&out, err)
	if !strings.Contains(out.String(), "failed to restore Git identity") || !strings.Contains(out.String(), rollback.Error()) || !strings.Contains(out.String(), missing.Error()) {
		t.Fatalf("diagnostic lost part of the switch failure: %q", out.String())
	}
}
