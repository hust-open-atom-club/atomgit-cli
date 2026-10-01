package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicConfigSaveCleanup(t *testing.T) {
	for _, kind := range []string{"credentials", "aliases"} {
		for _, failRename := range []bool{false, true} {
			t.Run(kind+"/"+map[bool]string{false: "success", true: "rename failure"}[failRename], func(t *testing.T) {
				isolateConfig(t)
				var path, pattern string
				var save func() error
				if kind == "credentials" {
					var err error
					path, err = PrimaryTokenPath()
					if err != nil {
						t.Fatal(err)
					}
					pattern = ".token-*.tmp"
					save = func() error {
						return saveCredentialStore(&CredentialStore{Active: "alice", Accounts: []StoredCredentials{{User: "alice", AccessToken: "test-token"}}})
					}
				} else {
					var err error
					path, err = AliasFilePath()
					if err != nil {
						t.Fatal(err)
					}
					pattern = filepath.Base(path) + ".tmp-*"
					save = func() error { return writeAliasesAtomic(path, map[string]string{"pl": "pr list"}) }
				}
				if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
					t.Fatal(err)
				}
				if failRename {
					if err := os.Mkdir(path, 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(path, "keep"), []byte("unchanged"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				err := save()
				if (err != nil) != failRename {
					t.Fatalf("save error = %v", err)
				}
				matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), pattern))
				if err != nil || len(matches) != 0 {
					t.Fatalf("temporary files = %v, error = %v", matches, err)
				}
				if failRename {
					data, err := os.ReadFile(filepath.Join(path, "keep"))
					if err != nil || string(data) != "unchanged" {
						t.Fatalf("destination changed: %q, %v", data, err)
					}
				} else if kind == "credentials" {
					store, err := LoadCredentialStore()
					if err != nil || store.Active != "alice" || len(store.Accounts) != 1 {
						t.Fatalf("saved credentials invalid: %v", err)
					}
				} else {
					aliases, err := LoadAliases()
					if err != nil || aliases["pl"] != "pr list" {
						t.Fatalf("saved aliases invalid: %v", err)
					}
				}
			})
		}
	}
}
