package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		return "", err
	}
	c.user = user
	return user, nil
}

func (c *config) GetHost() string {
	return c.host
}

func loadTokenFromFile() (string, string, error) {
	paths := getTokenFilePaths()

	if len(paths) == 0 {
		return "", "", fmt.Errorf("no token file paths available, please make sure user $HOME or $XDG_CONFIG_HOME is set")
	}

	var failedPaths []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			failedPaths = append(failedPaths, path)
			continue
		}

		var tokenData struct {
			AccessToken string `json:"access_token"`
			User        string `json:"user"`
		}

		if err := json.Unmarshal(data, &tokenData); err != nil {
			return "", "", fmt.Errorf("failed to parse token file at %s: %w", path, err)
		}

		return tokenData.AccessToken, tokenData.User, nil
	}

	return "", "", fmt.Errorf("token file not found.\nSearched locations:\n  - %s", strings.Join(failedPaths, "\n  - "))
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
