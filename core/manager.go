package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NSHConfigManager handles loading and saving nsh-managed SSH config.
// nsh stores its hosts in a dedicated file (e.g. ~/.ssh/nsh/config)
// and injects an Include directive into the main SSH config.
type NSHConfigManager struct {
	ConfigPath    string // nsh managed config: ~/.ssh/nsh/config
	SSHConfigPath string // main SSH config:    ~/.ssh/config
	current       *NSHConfig
}

// NewConfigManager creates a new manager.
// sshConfigPath is the main SSH config (default ~/.ssh/config).
// nsh config is derived as {ssh_dir}/nsh/config.
func NewConfigManager(sshConfigPath string) *NSHConfigManager {
	if sshConfigPath == "" {
		sshConfigPath = "~/.ssh/config"
	}
	sshPath := ExpandPath(sshConfigPath)
	sshDir := filepath.Dir(sshPath)
	nshConfigPath := filepath.Join(sshDir, "nsh", "config")
	return &NSHConfigManager{
		ConfigPath:    nshConfigPath,
		SSHConfigPath: sshPath,
	}
}

// NshDir returns the nsh working directory (e.g. ~/.ssh/nsh/)
func (m *NSHConfigManager) NshDir() string {
	return filepath.Dir(m.ConfigPath)
}

// EnsureSetup creates the nsh directory and injects Include into main SSH config.
func (m *NSHConfigManager) EnsureSetup() error {
	nshDir := m.NshDir()

	// Create ~/.ssh/nsh/ directory
	if err := os.MkdirAll(nshDir, 0700); err != nil {
		return fmt.Errorf("failed to create nsh directory %s: %w", nshDir, err)
	}

	// Create nsh config file if missing
	if _, err := os.Stat(m.ConfigPath); os.IsNotExist(err) {
		if err := os.WriteFile(m.ConfigPath, []byte(""), 0600); err != nil {
			return fmt.Errorf("failed to create nsh config: %w", err)
		}
	}

	// Ensure Include directive in main SSH config
	return m.ensureInclude()
}

// ensureInclude injects "Include {nsh/config}" at the top of the main SSH config.
func (m *NSHConfigManager) ensureInclude() error {
	sshDir := filepath.Dir(m.SSHConfigPath)

	// Ensure .ssh directory exists
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create ssh directory: %w", err)
	}

	// Create main SSH config if missing
	if _, err := os.Stat(m.SSHConfigPath); os.IsNotExist(err) {
		if err := os.WriteFile(m.SSHConfigPath, []byte(""), 0600); err != nil {
			return fmt.Errorf("failed to create ssh config: %w", err)
		}
	}

	content, err := os.ReadFile(m.SSHConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read ssh config: %w", err)
	}

	includePath := m.includePathStr()
	includeLine := "Include " + includePath

	// Check if Include already present
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, includeLine) {
			return nil // already present
		}
	}

	// Prepend Include line (SSH requires Include before Host blocks)
	newContent := includeLine + "\n"
	if len(content) > 0 {
		newContent += "\n" + string(content)
	}

	// Write back atomically
	tmpPath := m.SSHConfigPath + ".nsh.tmp"
	if err := os.WriteFile(tmpPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write tmp ssh config: %w", err)
	}
	if err := os.Rename(tmpPath, m.SSHConfigPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to update ssh config: %w", err)
	}
	os.Chmod(m.SSHConfigPath, 0600)

	return nil
}

// includePathStr returns the path to use in the Include directive.
// Uses ~/.ssh/nsh/config for default locations, absolute path otherwise.
func (m *NSHConfigManager) includePathStr() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return m.ConfigPath
	}
	defaultSSHDir := filepath.Join(home, ".ssh")
	if filepath.Dir(m.SSHConfigPath) == defaultSSHDir {
		return "~/.ssh/nsh/config"
	}
	return m.ConfigPath
}

// Load reads and parses the nsh config file
func (m *NSHConfigManager) Load() (*NSHConfig, error) {
	// Ensure nsh directory and Include are set up
	if err := m.EnsureSetup(); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(m.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read nsh config: %w", err)
	}

	config := Parse(string(content))
	m.current = config
	return config, nil
}

// Save writes the config (backup → tmp → rename → chmod 0600)
func (m *NSHConfigManager) Save(config *NSHConfig) error {
	// Step 1: Backup
	if err := Backup(m.ConfigPath); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	// Step 2: Serialize
	content := Serialize(config)

	// Step 3: Write to tmp file
	tmpPath := m.ConfigPath + ".nsh.tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write tmp file: %w", err)
	}

	// Step 4: Rename (atomic operation)
	if err := os.Rename(tmpPath, m.ConfigPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename tmp file: %w", err)
	}

	// Step 5: Chmod 0600
	os.Chmod(m.ConfigPath, 0600)

	// Cleanup tmp (defensive)
	os.Remove(tmpPath)

	m.current = config
	return nil
}

// GetConfig returns the currently loaded config
func (m *NSHConfigManager) GetConfig() *NSHConfig {
	return m.current
}

// LoadAllHosts returns hosts from both nsh config and main SSH config.
// nsh-managed hosts come first, followed by external hosts (read-only).
func (m *NSHConfigManager) LoadAllHosts() []NSHHost {
	var all []NSHHost

	// nsh-managed hosts
	if m.current != nil {
		all = append(all, m.current.Hosts()...)
	}

	// External hosts from main SSH config
	content, err := os.ReadFile(m.SSHConfigPath)
	if err != nil {
		return all
	}
	sshCfg := Parse(string(content))
	for _, h := range sshCfg.Hosts() {
		if h.IsWildcard {
			continue
		}
		// Skip duplicates (host already managed by nsh)
		if m.current != nil && m.current.HostByAlias(h.Alias) != nil {
			continue
		}
		all = append(all, h)
	}

	return all
}

// ExpandPath expands ~ to home directory
func ExpandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
