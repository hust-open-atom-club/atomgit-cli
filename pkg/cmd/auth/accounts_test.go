package auth

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/config"
	"atomgit.com/hust-open-atom-club/atomgit-cli/pkg/cmdutil"
)

func saveAuthTestAccounts(t *testing.T) {
	t.Helper()
	isolateAuthConfig(t)
	accounts := []config.StoredCredentials{
		{AccessToken: "alice-secret", User: "alice", Name: "Alice A", Email: "alice@example.com"},
		{AccessToken: "bob-secret", User: "bob", Name: "Bob B", Email: "bob@example.com"},
	}
	for _, account := range accounts {
		if err := config.SaveAccount(&account, true); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAuthListRedactsTokensAndMarksActive(t *testing.T) {
	saveAuthTestAccounts(t)
	cmd := newCmdAuthList()
	_ = cmd.Flags().Set("json", "true")
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"alice-secret", "bob-secret", "access_token", "refresh_token", `"host"`} {
		if strings.Contains(output.String(), secret) {
			t.Fatalf("output leaked %q: %s", secret, output.String())
		}
	}
	for _, want := range []string{`"login": "alice"`, `"login": "bob"`, `"active": true`} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("output missing %q: %s", want, output.String())
		}
	}
}

func TestAuthSwitchChangesOnlyActiveAccount(t *testing.T) {
	saveAuthTestAccounts(t)
	cmd := newCmdAuthSwitch()
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, []string{"alice"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "alice") {
		t.Fatalf("output = %q", output.String())
	}
	active, err := config.LoadStoredCredentials()
	if err != nil || active.User != "alice" || active.AccessToken != "alice-secret" {
		t.Fatalf("active = %#v, error = %v", active, err)
	}
}

func TestAuthGitSyncUpdatesLocalOrGlobalGitIdentity(t *testing.T) {
	for _, test := range []struct {
		name   string
		global bool
		scope  string
	}{
		{name: "local by default", scope: "--local"},
		{name: "global when explicit", global: true, scope: "--global"},
	} {
		t.Run(test.name, func(t *testing.T) {
			saveAuthTestAccounts(t)
			if _, err := config.SwitchAccount("alice"); err != nil {
				t.Fatal(err)
			}
			var calls []string
			factory := &cmdutil.Factory{GitConfig: func(args ...string) (string, error) {
				calls = append(calls, strings.Join(args, " "))
				if strings.Contains(strings.Join(args, " "), "--get user.name") {
					return "Old Name", nil
				}
				if strings.Contains(strings.Join(args, " "), "--get user.email") {
					return "old@example.com", nil
				}
				return "", nil
			}}
			cmd := newCmdAuthGitSync(factory)
			if test.global {
				_ = cmd.Flags().Set("global", "true")
			}
			if err := cmd.RunE(cmd, nil); err != nil {
				t.Fatal(err)
			}
			joined := strings.Join(calls, "\n")
			for _, want := range []string{
				"config " + test.scope + " user.name Alice A",
				"config " + test.scope + " user.email alice@example.com",
			} {
				if !strings.Contains(joined, want) {
					t.Fatalf("calls do not contain %q:\n%s", want, joined)
				}
			}
		})
	}
}

func TestUpdateGitIdentityRollsBackPartialFailure(t *testing.T) {
	var calls []string
	factory := &cmdutil.Factory{GitConfig: func(args ...string) (string, error) {
		call := strings.Join(args, " ")
		calls = append(calls, call)
		switch {
		case strings.Contains(call, "--get user.name"):
			return "Old Name", nil
		case strings.Contains(call, "--get user.email"):
			return "old@example.com", nil
		case strings.Contains(call, "user.email new@example.com"):
			return "", errors.New("write failed")
		default:
			return "", nil
		}
	}}
	err := updateGitIdentity(factory, false, "New Name", "new@example.com")
	if err == nil || !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("error = %v", err)
	}
	joined := strings.Join(calls, "\n")
	for _, want := range []string{"config --local user.name Old Name", "config --local user.email old@example.com"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("rollback missing %q:\n%s", want, joined)
		}
	}
}

func TestUpdateGitIdentityReportsRollbackFailure(t *testing.T) {
	identity := map[string]string{
		"user.name":  "Old Name",
		"user.email": "old@example.com",
	}
	factory := &cmdutil.Factory{GitConfig: func(args ...string) (string, error) {
		if len(args) == 4 && args[2] == "--get" {
			return identity[args[3]], nil
		}
		if len(args) == 4 && args[2] == "user.email" && args[3] == "new@example.com" {
			return "", errors.New("email write failed")
		}
		if len(args) == 4 && args[2] == "user.name" && args[3] == "Old Name" {
			return "", errors.New("name rollback failed")
		}
		if len(args) == 4 {
			identity[args[2]] = args[3]
		}
		return "", nil
	}}

	err := updateGitIdentity(factory, false, "New Name", "new@example.com")
	if err == nil {
		t.Fatal("expected update and rollback error")
	}
	for _, want := range []string{"email write failed", "name rollback failed"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err, want)
		}
	}
	if identity["user.name"] != "New Name" || identity["user.email"] != "old@example.com" {
		t.Fatalf("identity = %#v", identity)
	}
}

func TestAuthGitSyncValidatesIdentityWithoutChangingActiveAccount(t *testing.T) {
	isolateAuthConfig(t)
	if err := config.SaveAccount(&config.StoredCredentials{AccessToken: "alice", User: "alice"}, true); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveAccount(&config.StoredCredentials{AccessToken: "bob", User: "bob", Name: "Bob", Email: "bob@example.com"}, false); err != nil {
		t.Fatal(err)
	}
	cmd := newCmdAuthGitSync(&cmdutil.Factory{})
	err := cmd.RunE(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "no Git email") {
		t.Fatalf("error = %v", err)
	}
	active, loadErr := config.LoadStoredCredentials()
	if loadErr != nil || active.User != "alice" {
		t.Fatalf("active = %#v, error = %v", active, loadErr)
	}
}

func TestAuthLogoutSelectedAccount(t *testing.T) {
	saveAuthTestAccounts(t)
	cmd := newCmdAuthLogout()
	_ = cmd.Flags().Set("account", "alice")
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	accounts, active, err := config.ListAccounts()
	if err != nil || len(accounts) != 1 || accounts[0].User != "bob" || active != "bob" {
		t.Fatalf("accounts = %#v, active = %q, error = %v", accounts, active, err)
	}
}

func TestAuthLogoutActiveAccountRequiresExplicitSwitch(t *testing.T) {
	for _, test := range []struct {
		name            string
		explicitAccount bool
	}{
		{name: "default active account"},
		{name: "explicit active account", explicitAccount: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			saveAuthTestAccounts(t)
			cmd := newCmdAuthLogout()
			if test.explicitAccount {
				_ = cmd.Flags().Set("account", "bob")
			}
			err := cmd.RunE(cmd, nil)
			if err == nil || !strings.Contains(err.Error(), "ag auth switch <account>") {
				t.Fatalf("error = %v", err)
			}
			accounts, active, loadErr := config.ListAccounts()
			if loadErr != nil || len(accounts) != 2 || active != "bob" {
				t.Fatalf("accounts = %#v, active = %q, error = %v", accounts, active, loadErr)
			}
		})
	}
}

func TestAuthLogoutAllAccounts(t *testing.T) {
	saveAuthTestAccounts(t)
	cmd := newCmdAuthLogout()
	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if _, _, err := config.ListAccounts(); !errors.Is(err, config.ErrTokenNotFound) {
		t.Fatalf("ListAccounts() error = %v, want ErrTokenNotFound", err)
	}
}

func TestAuthGitSyncIdentityOverrides(t *testing.T) {
	saveAuthTestAccounts(t)
	var writes []string
	factory := &cmdutil.Factory{GitConfig: func(args ...string) (string, error) {
		call := strings.Join(args, " ")
		if strings.Contains(call, " --get ") {
			return "old", nil
		}
		writes = append(writes, call)
		return "", nil
	}}
	cmd := newCmdAuthGitSync(factory)
	_ = cmd.Flags().Set("git-name", "Commit Name")
	_ = cmd.Flags().Set("git-email", "commit@example.com")
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(writes, "\n")
	if !strings.Contains(joined, "Commit Name") || !strings.Contains(joined, "commit@example.com") {
		t.Fatalf("writes = %s", joined)
	}
}
