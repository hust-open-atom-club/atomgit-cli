package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultHost     = "atomgit.com"
	configFile      = "config.json"
	appName         = "ag-cli"
	tokenFile       = "token.json"
	legacyTokenFile = ".atomgit_personal_token.json"
)

type Config interface {
	GetToken() (string, error)
	GetUser() (string, error)
	GetHost() string
}

type config struct {
	host  string
	token string
	user  string
}

func NewConfig() (Config, error) {
	token, user, err := loadTokenFromFile()
	if errors.Is(err, ErrTokenNotFound) {
		return &config{host: defaultHost}, nil
	}
	if err != nil {
		return nil, err
	}

	return &config{
		host:  defaultHost,
		token: token,
		user:  user,
	}, nil
}

func (c *config) GetToken() (string, error) {
	if c.token != "" {
		return c.token, nil
	}

	token, _, err := loadTokenFromFile()
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return "", ErrNotAuthenticated
		}
		return "", err
	}
	c.token = token
	return token, nil
}

func (c *config) GetUser() (string, error) {
	if c.user != "" {
		return c.user, nil
	}

	_, user, err := loadTokenFromFile()
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return "", ErrNotAuthenticated
		}
		return "", err
	}
	c.user = user
	return user, nil
}

func (c *config) GetHost() string {
	return c.host
}

func loadTokenFromFile() (string, string, error) {
	c, err := LoadStoredCredentials()
	if err != nil {
		return "", "", err
	}
	return c.AccessToken, c.User, nil
}

// StoredCredentials is the on-disk token.json shape (extended for OAuth refresh).
type StoredCredentials struct {
	AccessToken  string `json:"access_token"`
	User         string `json:"user"`
	Name         string `json:"name,omitempty"`
	Email        string `json:"email,omitempty"`
	GitName      string `json:"git_name,omitempty"`
	GitEmail     string `json:"git_email,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	CreatedAt    int64  `json:"created_at,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
}

// LoadStoredCredentials reads the first available token file and parses extended fields.
func LoadStoredCredentials() (*StoredCredentials, error) {
	store, err := LoadCredentialStore()
	if err != nil {
		return nil, err
	}
	account, err := store.ActiveAccount()
	if err != nil {
		return nil, err
	}
	return account, nil
}

func readCredentialData() ([]byte, error) {
	paths := getTokenFilePaths()

	if len(paths) == 0 {
		return nil, fmt.Errorf("no token file paths available, please make sure user $HOME or $XDG_CONFIG_HOME is set")
	}

	var failedPaths []string
	for _, path := range paths {
		li, err := os.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				failedPaths = append(failedPaths, path)
				continue
			}
			if isPermissionErr(err) {
				return nil, &TokenPermissionError{Path: path, Err: err}
			}
			return nil, fmt.Errorf("lstat token file info %s: %w", path, err)
		}
		if li.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("cannot read %s: %w\n"+
				"remove the symlink and place the token file directly", path, ErrTokenFileSymlink)
		}

		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				failedPaths = append(failedPaths, path)
				continue
			}
			if isPermissionErr(err) {
				return nil, &TokenPermissionError{Path: path, Err: err}
			}
			return nil, fmt.Errorf("open token file %s: %w", path, err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			if isPermissionErr(err) {
				return nil, &TokenPermissionError{Path: path, Err: err}
			}
			return nil, fmt.Errorf("stat token file info %s: %w", path, err)
		}

		if !os.SameFile(li, info) {
			return nil, fmt.Errorf("cannot read %s: %w", path, ErrTokenFileChanged)
		}

		if err := validateAndFixTokenFilePerm(f, path, info); err != nil {
			return nil, err
		}

		data, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("read token file %s: %w", path, err)
		}

		return data, nil
	}

	return nil, fmt.Errorf("%w.\nSearched locations:\n  - %s", ErrTokenNotFound, strings.Join(failedPaths, "\n  - "))
}

// getTokenFilePaths returns candidate token file paths in search priority order.
//
// The search order follows the XDG Base Directory Specification:
//
//   - $XDG_CONFIG_HOME/<appName>/token.json (primary location)
//   - $HOME/.config/<appName>/token.json (default when XDG_CONFIG_HOME is unset)
//   - $HOME/.atomgit_personal_token.json (legacy fallback for backward compatibility)
//
// Returns:
//   - []string: slice of absolute file paths to search, ordered by priority.
//     Empty slice if home directory cannot be determined.
func getTokenFilePaths() []string {
	var paths []string

	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	homeDir, err := os.UserHomeDir()

	if xdgConfigHome != "" {
		paths = append(paths, filepath.Join(xdgConfigHome, appName, tokenFile))
	}

	if xdgConfigHome == "" && err == nil {
		paths = append(paths, filepath.Join(homeDir, ".config", appName, tokenFile))
	}

	if err == nil {
		paths = append(paths, filepath.Join(homeDir, legacyTokenFile))
	}

	return paths
}

// PrimaryTokenPath returns the path to the preferred ag-cli token.json (XDG config dir).
func PrimaryTokenPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	var dir string
	if xdg != "" {
		dir = filepath.Join(xdg, appName)
	} else {
		dir = filepath.Join(homeDir, ".config", appName)
	}
	return filepath.Join(dir, tokenFile), nil
}

// SaveToken writes access_token and user to PrimaryTokenPath(), replacing any existing file.
// Preserves refresh_token and other OAuth fields when merging with an existing file.
func SaveToken(accessToken, user string) error {
	if accessToken == "" {
		return fmt.Errorf("access_token is empty")
	}
	store, err := LoadCredentialStore()
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return SaveCredentials(&StoredCredentials{
				AccessToken: accessToken,
				User:        user,
				CreatedAt:   time.Now().Unix(),
			})
		}
		return err
	}
	existing, err := store.ActiveAccount()
	if err != nil {
		return err
	}
	existing.AccessToken = accessToken
	existing.User = user
	existing.CreatedAt = time.Now().Unix()
	return replaceActiveAccount(store, existing)
}

// SaveCredentials adds or updates an account and makes it active.
func SaveCredentials(c *StoredCredentials) error {
	return SaveAccount(c, true)
}

// ClearCredentials removes all known credential files (XDG token.json and legacy path).
// Returns the list of paths that were deleted.
func ClearCredentials() ([]string, error) {
	var removed []string
	for _, p := range getTokenFilePaths() {
		err := os.Remove(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			if isPermissionErr(err) {
				return removed, permissionError(
					"cannot remove token file", p, err,
					"check the file and parent directory ownership and permissions",
				)
			}
			return removed, fmt.Errorf("remove %s: %w", p, err)
		}
		removed = append(removed, p)
	}
	return removed, nil
}
