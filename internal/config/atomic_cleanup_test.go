package config

import (
	"errors"
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
				if kind == "credentials" && failRename {
					failure, ok := errors.AsType[*CredentialSaveError](err)
					if !ok || failure.Stage != CredentialSaveReplace || failure.Path != path {
						t.Fatalf("save error = %v, want replacement failure at %q", err, path)
					}
					cause, ok := errors.AsType[*os.LinkError](err)
					if !ok || cause.New != path || !errors.Is(err, cause.Err) {
						t.Fatalf("save error lost the original rename failure: %v", err)
					}
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

func TestSaveCredentialStoreRetainsDirectoryFailure(t *testing.T) {
	dir := filepath.Join(isolateConfig(t), "not-a-directory")
	if err := os.WriteFile(dir, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", dir)
	path, err := PrimaryTokenPath()
	if err != nil {
		t.Fatal(err)
	}
	err = saveCredentialStore(&CredentialStore{Active: "alice", Accounts: []StoredCredentials{{User: "alice", AccessToken: "fixture"}}})
	failure, ok := errors.AsType[*CredentialSaveError](err)
	if !ok || failure.Stage != CredentialSaveMkdir || failure.Path != filepath.Dir(path) {
		t.Fatalf("save error = %v, want credential directory failure", err)
	}
	cause, ok := errors.AsType[*os.PathError](err)
	if !ok || !errors.Is(err, cause.Err) {
		t.Fatalf("save error lost the original directory failure: %v", err)
	}
}

func TestCredentialSaveRetainsWriteAndCleanupFailures(t *testing.T) {
	dir := t.TempDir()
	temporaryPath := filepath.Join(dir, "temporary")
	if err := os.Mkdir(temporaryPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temporaryPath, "keep"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	writeErr := &os.PathError{Op: "write", Path: temporaryPath, Err: os.ErrPermission}
	closeErr := &os.PathError{Op: "close", Path: temporaryPath, Err: os.ErrClosed}
	file := &failingCredentialTempFile{path: temporaryPath, writeErr: writeErr, closeErr: closeErr}
	failure := finishCredentialSave(file, filepath.Join(dir, "token.json"), []byte("fixture"))
	if failure == nil || failure.Stage != CredentialSaveWrite || failure.Path != temporaryPath {
		t.Fatalf("save error = %v, want write failure at %q", failure, temporaryPath)
	}
	if !errors.Is(failure.Err, writeErr) || !errors.Is(failure.CloseErr, closeErr) || file.closes != 1 {
		t.Fatalf("write or close failure lost: %v, close attempts = %d", failure, file.closes)
	}
	removeErr, ok := errors.AsType[*os.PathError](failure.RemoveErr)
	if !ok || removeErr.Op != "remove" || removeErr.Path != temporaryPath {
		t.Fatalf("temporary file cleanup error = %v, want original removal failure", failure.RemoveErr)
	}
	for _, cause := range []error{writeErr, closeErr, removeErr} {
		if !errors.Is(failure, cause) {
			t.Errorf("save error does not wrap %v", cause)
		}
	}
}

type failingCredentialTempFile struct {
	path     string
	writeErr error
	closeErr error
	closes   int
}

func (f *failingCredentialTempFile) Name() string            { return f.path }
func (f *failingCredentialTempFile) Chmod(os.FileMode) error { return nil }
func (f *failingCredentialTempFile) Write([]byte) (int, error) {
	return 0, f.writeErr
}
func (f *failingCredentialTempFile) Sync() error { return nil }
func (f *failingCredentialTempFile) Close() error {
	f.closes++
	return f.closeErr
}
