package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds application-wide settings
type Config struct {
	// General settings
	Language string `json:"language,omitempty"`

	// Update settings
	DisableUpdateCheck  bool `json:"disableUpdateCheck"`
	UpdateCheckInterval int  `json:"updateCheckInterval"` // Hours between checks
	NotifyThresholdDays int  `json:"notifyThresholdDays"` // Minimum age difference to show notification
}

// GetConfigDir returns the path to ~/.arcitems, creating it if needed
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	arcItemsDir := filepath.Join(homeDir, ".arcitems")
	if err := os.MkdirAll(arcItemsDir, 0755); err != nil {
		return "", err
	}

	return arcItemsDir, nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load loads the config from disk, returning defaults if file doesn't exist
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return defaults(), nil
		}
		return nil, err
	}

	cfg := defaults()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save persists the config to disk
func Save(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// defaults returns a Config with default values
func defaults() *Config {
	return &Config{
		Language:            "",
		DisableUpdateCheck:  false,
		UpdateCheckInterval: 24,
		NotifyThresholdDays: 0,
	}
}
