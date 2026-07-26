package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCredentialStoreMigratesLegacyWithoutLosingOAuthFields(t *testing.T) {
	home := isolateConfig(t)
	path := filepath.Join(home, ".config", appName, tokenFile)
	legacy := StoredCredentials{
		AccessToken: "alice-access", User: "alice", RefreshToken: "alice-refresh",
		ExpiresIn: 3600, CreatedAt: 123, TokenType: "Bearer",
	}
	writeCredentialsFile(t, path, legacy)

	store, err := LoadCredentialStore()
	if err != nil {
		t.Fatal(err)
	}
	if store.Active != "atomgit.com/alice" || len(store.Accounts) != 1 || store.Accounts[0].RefreshToken != "alice-refresh" {
		t.Fatalf("store = %#v", store)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(before), `"accounts"`) {
		t.Fatal("read-only migration rewrote the legacy file")
	}

	if err := SaveAccount(&StoredCredentials{AccessToken: "bob-access", User: "bob", RefreshToken: "bob-refresh"}, true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var persisted CredentialStore
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Version != credentialStoreVersion || persisted.Active != "atomgit.com/bob" || len(persisted.Accounts) != 2 {
		t.Fatalf("persisted = %#v", persisted)
	}
	alice, err := persisted.ResolveAccount("alice")
	if err != nil || alice.RefreshToken != "alice-refresh" || alice.ExpiresIn != 3600 {
		t.Fatalf("alice = %#v, error = %v", alice, err)
	}
}

func TestCredentialStoreRejectsFutureVersionWithoutRewriting(t *testing.T) {
	home := isolateConfig(t)
	path := filepath.Join(home, ".config", appName, tokenFile)
	data := []byte(`{
  "version": 3,
  "active": "atomgit.com/alice",
  "future_store_field": "keep",
  "accounts": [{
    "access_token": "alice-token",
    "user": "alice",
    "future_account_field": "keep"
  }]
}`)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := SwitchAccount("alice"); err == nil || !strings.Contains(err.Error(), "unsupported credential store version 3") {
		t.Fatalf("error = %v", err)
	}
	if err := SaveAccount(&StoredCredentials{AccessToken: "bob-token", User: "bob"}, true); err == nil || !strings.Contains(err.Error(), "unsupported credential store version 3") {
		t.Fatalf("save error = %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(data) {
		t.Fatalf("future credential file was rewritten:\n%s", after)
	}
}

func TestSaveAccountUpdatesOneAccountAndPreservesOthers(t *testing.T) {
	isolateConfig(t)
	if err := SaveAccount(&StoredCredentials{AccessToken: "alice-old", User: "alice", RefreshToken: "alice-refresh"}, true); err != nil {
		t.Fatal(err)
	}
	if err := SaveAccount(&StoredCredentials{AccessToken: "bob", User: "bob", RefreshToken: "bob-refresh"}, true); err != nil {
		t.Fatal(err)
	}
	if err := SaveAccount(&StoredCredentials{AccessToken: "alice-new", User: "alice", RefreshToken: "alice-new-refresh"}, true); err != nil {
		t.Fatal(err)
	}
	store, err := LoadCredentialStore()
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Accounts) != 2 || store.Active != "atomgit.com/alice" {
		t.Fatalf("store = %#v", store)
	}
	bob, err := store.ResolveAccount("bob")
	if err != nil || bob.AccessToken != "bob" || bob.RefreshToken != "bob-refresh" {
		t.Fatalf("bob = %#v, error = %v", bob, err)
	}
}

func TestCredentialStoreSwitchAndAmbiguousLogin(t *testing.T) {
	isolateConfig(t)
	for _, account := range []StoredCredentials{
		{AccessToken: "a", User: "same", Host: "atomgit.com"},
		{AccessToken: "b", User: "same", Host: "gitcode.com"},
	} {
		if err := SaveAccount(&account, false); err != nil {
			t.Fatal(err)
		}
	}
	store, err := LoadCredentialStore()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResolveAccount("same"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("error = %v", err)
	}
	if key, err := SwitchAccount("gitcode.com/same"); err != nil || key != "gitcode.com/same" {
		t.Fatalf("key = %q, error = %v", key, err)
	}
	active, err := LoadStoredCredentials()
	if err != nil || active.AccessToken != "b" {
		t.Fatalf("active = %#v, error = %v", active, err)
	}
}

func TestRemoveInactiveAccountKeepsActiveAccount(t *testing.T) {
	isolateConfig(t)
	for _, account := range []StoredCredentials{
		{AccessToken: "alice", User: "alice"},
		{AccessToken: "bob", User: "bob"},
	} {
		if err := SaveAccount(&account, true); err != nil {
			t.Fatal(err)
		}
	}
	removed, empty, err := RemoveAccount("alice")
	if err != nil || empty || removed != "atomgit.com/alice" {
		t.Fatalf("removed = %q, empty = %t, error = %v", removed, empty, err)
	}
	store, err := LoadCredentialStore()
	if err != nil || len(store.Accounts) != 1 || store.Active != "atomgit.com/bob" {
		t.Fatalf("store = %#v, error = %v", store, err)
	}
}

func TestRemoveActiveAccountRequiresExplicitSwitch(t *testing.T) {
	home := isolateConfig(t)
	for _, account := range []StoredCredentials{
		{AccessToken: "alice", User: "alice"},
		{AccessToken: "bob", User: "bob"},
	} {
		if err := SaveAccount(&account, true); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(home, ".config", appName, tokenFile)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	removed, empty, err := RemoveAccount("bob")
	if err == nil || !strings.Contains(err.Error(), "ag auth switch <account>") {
		t.Fatalf("removed = %q, empty = %t, error = %v", removed, empty, err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(before) {
		t.Fatalf("credential store changed after rejected removal:\n%s", after)
	}
}
