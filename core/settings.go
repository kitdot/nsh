package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// NSHSettings stores user preferences
type NSHSettings struct {
	Mode string `json:"mode"` // "auto", "fzf", "list"
}

// DefaultSettings returns default settings
func DefaultSettings() NSHSettings {
	return NSHSettings{Mode: "auto"}
}

func settingsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "nsh")
}

func settingsPath() string {
	return filepath.Join(settingsDir(), "settings.json")
}

// LoadSettings reads settings from ~/.config/nsh/settings.json
func LoadSettings() NSHSettings {
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return DefaultSettings()
	}
	var s NSHSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return DefaultSettings()
	}
	if s.Mode == "" {
		s.Mode = "auto"
	}
	return s
}

// Save writes settings to ~/.config/nsh/settings.json
func (s NSHSettings) Save() error {
	dir := settingsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath(), data, 0644)
}
