package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AliasConfig is the on-disk shape of the alias configuration file.
type AliasConfig struct {
	Aliases map[string]string `json:"aliases"`
}

// AliasFilePath returns the path to the ag-cli alias configuration file,
// located next to the primary token file in the XDG config directory.
func AliasFilePath() (string, error) {
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
	return filepath.Join(dir, configFile), nil
}

// LoadAliases reads the alias configuration and returns the alias map.
// A missing configuration file yields an empty map.
func LoadAliases() (map[string]string, error) {
	path, err := AliasFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read alias config %s: %w", path, err)
	}
	var cfg AliasConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse alias config %s: %w", path, err)
	}
	if cfg.Aliases == nil {
		return map[string]string{}, nil
	}
	return cfg.Aliases, nil
}

// SaveAlias stores an alias in the configuration file, replacing any
// existing alias with the same name.
func SaveAlias(name, expansion string) error {
	return updateAliases(func(aliases map[string]string) error {
		aliases[name] = expansion
		return nil
	})
}

// DeleteAlias removes an alias from the configuration file. It reports
// whether an alias with the given name existed and was removed.
func DeleteAlias(name string) (bool, error) {
	deleted := false
	err := updateAliases(func(aliases map[string]string) error {
		if _, ok := aliases[name]; !ok {
			return nil
		}
		delete(aliases, name)
		deleted = true
		return nil
	})
	return deleted, err
}

// updateAliases loads the alias map, applies mutate, and persists the
// result back to the configuration file with owner-only permissions.
func updateAliases(mutate func(aliases map[string]string) error) error {
	path, err := AliasFilePath()
	if err != nil {
		return err
	}
	aliases, err := LoadAliases()
	if err != nil {
		return err
	}
	if err := mutate(aliases); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(AliasConfig{Aliases: aliases}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode alias config: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write alias config %s: %w", path, err)
	}
	return nil
}
