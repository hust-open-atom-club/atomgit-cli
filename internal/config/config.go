package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var (
	// ErrTokenNotFound is returned when no token file exists in any search path.
	ErrTokenNotFound = errors.New("token file not found")

	// ErrTokenFileSymlink is returned when a token file is a symlink
	ErrTokenFileSymlink = errors.New("token file is a symlink")
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
			return "", fmt.Errorf("not authenticated: run `ag auth login`")
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
			return "", fmt.Errorf("not authenticated: run `ag auth login`")
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
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	CreatedAt    int64  `json:"created_at,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
}

// LoadStoredCredentials reads the first available token file and parses extended fields.
func LoadStoredCredentials() (*StoredCredentials, error) {
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
			return nil, fmt.Errorf("lstat token file info %s: %w", path, err)
		}
		if li.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%w: %s\n"+
				"remove the symlink and place the token file directly", ErrTokenFileSymlink, path)
		}

		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				failedPaths = append(failedPaths, path)
				continue
			}
			return nil, fmt.Errorf("open token file %s: %w", path, err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return nil, fmt.Errorf("stat token file info %s: %w", path, err)
		}

		if perm := info.Mode().Perm(); perm&0o077 != 0 && runtime.GOOS != "windows" {
			if err := f.Chmod(perm & 0o700); err != nil {
				return nil, fmt.Errorf("change token file mode %s: %w", path, err)
			}
		}

		data, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("read token file %s: %w", path, err)
		}

		var c StoredCredentials
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, fmt.Errorf("failed to parse token file at %s: %w", path, err)
		}
		if c.AccessToken == "" {
			return nil, fmt.Errorf("token file at %s has empty access_token", path)
		}

		return &c, nil
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
	existing, err := LoadStoredCredentials()
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
	existing.AccessToken = accessToken
	existing.User = user
	existing.CreatedAt = time.Now().Unix()
	return SaveCredentials(existing)
}

// SaveCredentials writes the full credential record to PrimaryTokenPath().
func SaveCredentials(c *StoredCredentials) error {
	if c == nil {
		return fmt.Errorf("credentials are nil")
	}
	if c.AccessToken == "" {
		return fmt.Errorf("access_token is empty")
	}
	if c.User == "" {
		return fmt.Errorf("user is empty")
	}
	w := *c
	if w.CreatedAt == 0 {
		w.CreatedAt = time.Now().Unix()
	}
	path, err := PrimaryTokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	b, err := json.MarshalIndent(&w, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		if isPermissionErr(err) {
			dir := filepath.Dir(path)
			return fmt.Errorf("cannot write %s: %w\n"+
				"often caused by this directory or file having been created with sudo; fix ownership, e.g.:\n"+
				"  sudo chown -R $(whoami) %q", path, err, dir)
		}
		return err
	}
	return nil
}

func isPermissionErr(err error) bool {
	return errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM)
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
			return removed, fmt.Errorf("remove %s: %w", p, err)
		}
		removed = append(removed, p)
	}
	return removed, nil
}
