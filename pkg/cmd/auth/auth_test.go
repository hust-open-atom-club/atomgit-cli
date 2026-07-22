package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/oauth"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

type testConfig struct {
	token    string
	user     string
	tokenErr error
	userErr  error
}

func (c testConfig) GetToken() (string, error) { return c.token, c.tokenErr }
func (c testConfig) GetUser() (string, error)  { return c.user, c.userErr }
func (c testConfig) GetHost() string           { return "atomgit.com" }

type authRoundTripFunc func(*http.Request) (*http.Response, error)

func (f authRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func captureStdout(t *testing.T, run func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = old })

	runErr := run()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	return string(output), runErr
}

func isolateAuthConfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	return home
}

func TestIsolateAuthConfigUsesTemporaryHome(t *testing.T) {
	home := isolateAuthConfig(t)
	got, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != home {
		t.Fatalf("os.UserHomeDir() = %q, want %q", got, home)
	}
}

func TestNewCmdAuthRegistersSubcommands(t *testing.T) {
	cmd := NewCmdAuth(&cmdutil.Factory{Config: testConfig{tokenErr: errors.New("not authenticated")}})
	want := map[string]bool{"login": false, "logout": false, "refresh": false, "status": false, "token": false}
	for _, child := range cmd.Commands() {
		if _, ok := want[child.Name()]; ok {
			want[child.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q was not registered", name)
		}
	}
	login, _, err := cmd.Find([]string{"login"})
	if err != nil {
		t.Fatal(err)
	}
	if login.Flags().Lookup("force") == nil {
		t.Fatal("login --force flag was not registered")
	}
}

func TestAuthLoginSkipsBrowserWhenAlreadyAuthenticated(t *testing.T) {
	factory := &cmdutil.Factory{Config: testConfig{token: "token", user: "alice"}}
	cmd := newCmdAuthLogin(factory)
	output, err := captureStdout(t, func() error { return cmd.RunE(cmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Already logged in as alice") || !strings.Contains(output, "skipping browser login") {
		t.Fatalf("output = %q", output)
	}
}

func TestAuthLoginSavesOAuthResult(t *testing.T) {
	isolateAuthConfig(t)
	loginCalled := false
	factory := &cmdutil.Factory{Config: testConfig{tokenErr: errors.New("not authenticated")}}
	cmd := newCmdAuthLoginWithFunc(factory, func(ctx context.Context) (*oauth.LoginResult, error) {
		loginCalled = true
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("login context has no timeout")
		}
		return &oauth.LoginResult{
			AccessToken: "access", Login: "alice", RefreshToken: "refresh", ExpiresIn: 3600, TokenType: "Bearer",
		}, nil
	})
	cmd.SetContext(context.Background())

	output, err := captureStdout(t, func() error { return cmd.RunE(cmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	if !loginCalled || !strings.Contains(output, "Logged in to atomgit.com as alice") {
		t.Fatalf("called = %v, output = %q", loginCalled, output)
	}
	credentials, err := config.LoadStoredCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if credentials.AccessToken != "access" || credentials.User != "alice" || credentials.RefreshToken != "refresh" || credentials.ExpiresIn != 3600 {
		t.Fatalf("credentials = %#v", credentials)
	}
}

func TestAuthLoginReportsOAuthError(t *testing.T) {
	factory := &cmdutil.Factory{Config: testConfig{tokenErr: errors.New("not authenticated")}}
	cmd := newCmdAuthLoginWithFunc(factory, func(context.Context) (*oauth.LoginResult, error) {
		return nil, errors.New("authorization denied")
	})
	cmd.SetContext(context.Background())
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "authorization denied") {
		t.Fatalf("error = %v", err)
	}
}

func TestAuthStatus(t *testing.T) {
	tests := []struct {
		name   string
		config testConfig
		want   []string
	}{
		{
			name:   "authenticated",
			config: testConfig{token: "1234567890abcdef", user: "alice"},
			want:   []string{"Logged in to atomgit.com as alice", "1234****cdef"},
		},
		{
			name:   "short token",
			config: testConfig{token: "short", user: "alice"},
			want:   []string{"Token: short"},
		},
		{
			name:   "missing token",
			config: testConfig{tokenErr: errors.New("missing token")},
			want:   []string{"Not authenticated", "missing token"},
		},
		{
			name:   "missing user",
			config: testConfig{token: "token", userErr: errors.New("missing user")},
			want:   []string{"Token found but user not configured"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCmdAuthStatus(&cmdutil.Factory{Config: tt.config})
			output, err := captureStdout(t, func() error { return cmd.RunE(cmd, nil) })
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Fatalf("output %q does not contain %q", output, want)
				}
			}
		})
	}
}

func TestAuthToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmd := newCmdAuthToken(&cmdutil.Factory{Config: testConfig{token: "secret-token"}})
		output, err := captureStdout(t, func() error { return cmd.RunE(cmd, nil) })
		if err != nil {
			t.Fatal(err)
		}
		if output != "secret-token\n" {
			t.Fatalf("output = %q", output)
		}
	})

	t.Run("not authenticated", func(t *testing.T) {
		cmd := newCmdAuthToken(&cmdutil.Factory{Config: testConfig{tokenErr: errors.New("missing")}})
		err := cmd.RunE(cmd, nil)
		if err == nil || !strings.Contains(err.Error(), "not authenticated: missing") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestAuthLogout(t *testing.T) {
	t.Run("removes credentials", func(t *testing.T) {
		isolateAuthConfig(t)
		if err := config.SaveCredentials(&config.StoredCredentials{AccessToken: "token", User: "alice"}); err != nil {
			t.Fatal(err)
		}
		path, err := config.PrimaryTokenPath()
		if err != nil {
			t.Fatal(err)
		}

		cmd := newCmdAuthLogout()
		output, err := captureStdout(t, func() error { return cmd.RunE(cmd, nil) })
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output, "Logged out") || !strings.Contains(output, path) {
			t.Fatalf("output = %q", output)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("credential still exists: %v", err)
		}
	})

	t.Run("already logged out", func(t *testing.T) {
		isolateAuthConfig(t)
		cmd := newCmdAuthLogout()
		output, err := captureStdout(t, func() error { return cmd.RunE(cmd, nil) })
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output, "already logged out") {
			t.Fatalf("output = %q", output)
		}
	})
}

func TestAuthRefresh(t *testing.T) {
	isolateAuthConfig(t)
	t.Setenv("AG_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("AG_OAUTH_CLIENT_SECRET", "client-secret")
	if err := config.SaveCredentials(&config.StoredCredentials{
		AccessToken: "old-access", RefreshToken: "old-refresh", User: "alice", TokenType: "Bearer",
	}); err != nil {
		t.Fatal(err)
	}

	oldClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: authRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.String() != "https://atomgit.com/oauth/token" {
			t.Fatalf("request = %s %s", req.Method, req.URL)
		}
		if err := req.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if req.Form.Get("refresh_token") != "old-refresh" || req.Form.Get("grant_type") != "refresh_token" {
			t.Fatalf("form = %#v", req.Form)
		}
		body := `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":120}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})}
	t.Cleanup(func() { http.DefaultClient = oldClient })

	cmd := newCmdAuthRefresh()
	cmd.SetContext(context.Background())
	output, err := captureStdout(t, func() error { return cmd.RunE(cmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Token refreshed for alice") {
		t.Fatalf("output = %q", output)
	}
	credentials, err := config.LoadStoredCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if credentials.AccessToken != "new-access" || credentials.RefreshToken != "new-refresh" || credentials.ExpiresIn != 120 || credentials.TokenType != "Bearer" {
		t.Fatalf("credentials = %#v", credentials)
	}
}

func TestAuthRefreshValidation(t *testing.T) {
	t.Run("not logged in", func(t *testing.T) {
		isolateAuthConfig(t)
		cmd := newCmdAuthRefresh()
		cmd.SetContext(context.Background())
		err := cmd.RunE(cmd, nil)
		if err == nil || !strings.Contains(err.Error(), "not logged in") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("missing refresh token", func(t *testing.T) {
		isolateAuthConfig(t)
		if err := config.SaveCredentials(&config.StoredCredentials{AccessToken: "token", User: "alice"}); err != nil {
			t.Fatal(err)
		}
		cmd := newCmdAuthRefresh()
		cmd.SetContext(context.Background())
		err := cmd.RunE(cmd, nil)
		if err == nil || !strings.Contains(err.Error(), "no refresh_token") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestAuthCommandArgs(t *testing.T) {
	cmd := NewCmdAuth(&cmdutil.Factory{Config: testConfig{token: "token", user: "alice"}})
	for _, name := range []string{"login", "logout", "refresh", "status", "token"} {
		child, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatal(err)
		}
		if child.Args == nil {
			t.Errorf("%s does not define argument validation", name)
			continue
		}
		if err := child.Args(child, []string{"unexpected"}); err == nil {
			t.Errorf("%s accepted an unexpected argument", name)
		}
	}
}
