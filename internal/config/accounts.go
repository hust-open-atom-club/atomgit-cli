package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const credentialStoreVersion = 2

// CredentialStore is the versioned multi-account token.json format.
type CredentialStore struct {
	Version  int                 `json:"version"`
	Active   string              `json:"active"`
	Accounts []StoredCredentials `json:"accounts"`
}

// Key returns the normalized AtomGit username used to identify an account.
func (c StoredCredentials) Key() string {
	return strings.ToLower(strings.TrimSpace(c.User))
}

func normalizeAccount(c StoredCredentials) (StoredCredentials, error) {
	c.User = strings.TrimSpace(c.User)
	if c.AccessToken == "" {
		return StoredCredentials{}, fmt.Errorf("access_token is empty")
	}
	if c.User == "" {
		return StoredCredentials{}, fmt.Errorf("user is empty")
	}
	if c.CreatedAt == 0 {
		c.CreatedAt = time.Now().Unix()
	}
	return c, nil
}

// LoadCredentialStore reads either the multi-account format or the legacy
// single-account record. Legacy data remains unchanged until the next write.
func LoadCredentialStore() (*CredentialStore, error) {
	data, err := readCredentialData()
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Version  int             `json:"version"`
		Accounts json.RawMessage `json:"accounts"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}
	if envelope.Version != 0 && envelope.Version != credentialStoreVersion {
		return nil, fmt.Errorf("unsupported credential store version %d", envelope.Version)
	}
	if envelope.Version == 0 && envelope.Accounts != nil {
		return nil, fmt.Errorf("credential store version is required when accounts are present")
	}
	var store CredentialStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}
	if store.Version == credentialStoreVersion {
		if len(store.Accounts) == 0 {
			return nil, fmt.Errorf("token file has no accounts")
		}
		store.Active = strings.ToLower(strings.TrimSpace(store.Active))
		seen := make(map[string]bool, len(store.Accounts))
		for index := range store.Accounts {
			account, err := normalizeAccount(store.Accounts[index])
			if err != nil {
				return nil, fmt.Errorf("invalid account %d: %w", index, err)
			}
			key := account.Key()
			if seen[key] {
				return nil, fmt.Errorf("duplicate account %s", key)
			}
			seen[key] = true
			store.Accounts[index] = account
		}
		if !seen[store.Active] {
			return nil, fmt.Errorf("active account %q was not found", store.Active)
		}
		store.Version = credentialStoreVersion
		return &store, nil
	}

	var legacy StoredCredentials
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}
	legacy, err = normalizeAccount(legacy)
	if err != nil {
		return nil, fmt.Errorf("token file has empty access_token or invalid account: %w", err)
	}
	return &CredentialStore{Version: credentialStoreVersion, Active: legacy.Key(), Accounts: []StoredCredentials{legacy}}, nil
}

// ActiveAccount returns a copy of the selected account.
func (s *CredentialStore) ActiveAccount() (*StoredCredentials, error) {
	if s == nil {
		return nil, fmt.Errorf("credential store is nil")
	}
	for index := range s.Accounts {
		if s.Accounts[index].Key() == s.Active {
			copy := s.Accounts[index]
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("active account %q was not found", s.Active)
}

// ResolveAccount finds an account by its case-insensitive username.
func (s *CredentialStore) ResolveAccount(selector string) (*StoredCredentials, error) {
	selector = strings.ToLower(strings.TrimSpace(selector))
	if selector == "" {
		return nil, fmt.Errorf("account is required")
	}
	for _, account := range s.Accounts {
		if account.Key() == selector {
			copy := account
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("account %q was not found", selector)
}

// ListAccounts returns accounts sorted by username.
func ListAccounts() ([]StoredCredentials, string, error) {
	store, err := LoadCredentialStore()
	if err != nil {
		return nil, "", err
	}
	accounts := append([]StoredCredentials(nil), store.Accounts...)
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].Key() < accounts[j].Key() })
	return accounts, store.Active, nil
}

// SaveAccount adds or replaces one username record.
func SaveAccount(credentials *StoredCredentials, makeActive bool) error {
	if credentials == nil {
		return fmt.Errorf("credentials are nil")
	}
	account, err := normalizeAccount(*credentials)
	if err != nil {
		return err
	}
	store, err := LoadCredentialStore()
	if errors.Is(err, ErrTokenNotFound) {
		store = &CredentialStore{Version: credentialStoreVersion}
	} else if err != nil {
		return err
	}
	found := false
	for index := range store.Accounts {
		if store.Accounts[index].Key() == account.Key() {
			store.Accounts[index] = account
			found = true
			break
		}
	}
	if !found {
		store.Accounts = append(store.Accounts, account)
	}
	if makeActive || store.Active == "" {
		store.Active = account.Key()
	}
	return saveCredentialStore(store)
}

// SwitchAccount selects an existing account and returns its key.
func SwitchAccount(selector string) (string, error) {
	store, err := LoadCredentialStore()
	if err != nil {
		return "", err
	}
	account, err := store.ResolveAccount(selector)
	if err != nil {
		return "", err
	}
	store.Active = account.Key()
	return store.Active, saveCredentialStore(store)
}

// RemoveAccount removes one account. An active account cannot be removed while
// other accounts remain because selecting a replacement must be explicit.
func RemoveAccount(selector string) (string, bool, error) {
	store, err := LoadCredentialStore()
	if err != nil {
		return "", false, err
	}
	account, err := store.ResolveAccount(selector)
	if err != nil {
		return "", false, err
	}
	key := account.Key()
	if store.Active == key && len(store.Accounts) > 1 {
		return "", false, fmt.Errorf("cannot remove active account %s while other accounts remain; run `ag auth switch <account>` first", key)
	}
	remaining := make([]StoredCredentials, 0, len(store.Accounts)-1)
	for _, candidate := range store.Accounts {
		if candidate.Key() != key {
			remaining = append(remaining, candidate)
		}
	}
	if len(remaining) == 0 {
		_, err := ClearCredentials()
		return key, true, err
	}
	store.Accounts = remaining
	return key, false, saveCredentialStore(store)
}

func saveCredentialStore(store *CredentialStore) error {
	if store == nil || len(store.Accounts) == 0 {
		return fmt.Errorf("credential store has no accounts")
	}
	store.Version = credentialStoreVersion
	path, err := PrimaryTokenPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		if isPermissionErr(err) {
			return permissionError("cannot create config directory", dir, err, "check the directory ownership and owner write permission")
		}
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(dir, ".token-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary credential file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure temporary credential file: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary credential file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary credential file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary credential file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		if isPermissionErr(err) {
			return permissionError("cannot write token file", path, err, "check the file and parent directory ownership and owner write permission")
		}
		return fmt.Errorf("replace credential file: %w", err)
	}
	return nil
}
