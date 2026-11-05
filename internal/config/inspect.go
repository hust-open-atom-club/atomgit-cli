package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// InspectCredentials reads the selected credentials without chmod, migration,
// refresh or any other write. Callers must never print the returned credentials
// or raw errors in a shareable diagnostic report.
func InspectCredentials() (*StoredCredentials, os.FileMode, error) {
	paths, err := tokenFilePaths()
	if err != nil {
		return nil, 0, err
	}
	for _, path := range paths {
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, 0, &CredentialAccessError{Operation: "lstat credential file", Path: path, Err: err}
		}
		if !info.Mode().IsRegular() {
			return nil, 0, fmt.Errorf("credential file must be a regular file")
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, info.Mode(), &CredentialAccessError{Operation: "open credential file", Path: path, Err: err}
		}
		opened, statErr := f.Stat()
		if err := validateInspectedFileIdentity(path, info, opened, statErr); err != nil {
			// Preserve the stat or identity failure; closing the read-only file is best-effort cleanup.
			_ = f.Close()
			return nil, info.Mode(), err
		}
		const maxSize = 1 << 20
		data, err := io.ReadAll(io.LimitReader(f, maxSize+1))
		f.Close() //nolint:errcheck // Credential contents were read separately; Close only releases the read-only file.
		if err != nil {
			return nil, info.Mode(), &CredentialAccessError{Operation: "read credential file", Path: path, Err: err}
		}
		if len(data) > maxSize {
			return nil, info.Mode(), fmt.Errorf("credential file exceeds size limit")
		}
		store, _, err := parseCredentialStore(data)
		if err != nil {
			return nil, info.Mode(), err
		}
		account, err := store.ActiveAccount()
		// Parsing legacy accounts normally fills missing timestamps for migration.
		// A diagnostic must preserve unknown expiry instead of inventing it.
		if err == nil {
			var raw struct {
				StoredCredentials
				Accounts []StoredCredentials `json:"accounts"`
			}
			if json.Unmarshal(data, &raw) == nil {
				account.CreatedAt = raw.CreatedAt
				for _, a := range raw.Accounts {
					if a.Key() == account.Key() {
						account.CreatedAt = a.CreatedAt
						break
					}
				}
			}
		}
		return account, info.Mode(), err
	}
	return nil, 0, ErrTokenNotFound
}

func validateInspectedFileIdentity(path string, before, after os.FileInfo, statErr error) error {
	if statErr != nil {
		return &CredentialAccessError{Operation: "stat credential file", Path: path, Err: statErr}
	}
	if !os.SameFile(before, after) {
		return ErrTokenFileChanged
	}
	return nil
}
