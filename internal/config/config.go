package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultHost = "atomgit.com"
	configFile  = "config.json"
	tokenFile   = ".atomgit_personal_token.json"
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to get home directory: %w", err)
	}

	tokenPath := filepath.Join(homeDir, tokenFile)
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read token file at %s: %w", tokenPath, err)
	}

	var tokenData struct {
		AccessToken string `json:"access_token"`
		User        string `json:"user"`
	}

	if err := json.Unmarshal(data, &tokenData); err != nil {
		return "", "", fmt.Errorf("failed to parse token file: %w", err)
	}

	return tokenData.AccessToken, tokenData.User, nil
}
