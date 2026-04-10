package core

import (
	"fmt"
	"os/exec"
	"strings"
)

const keychainService = "nsh"

// KeychainSetPassword stores a password in macOS Keychain
func KeychainSetPassword(password, alias string) error {
	// Delete existing entry silently
	KeychainDeletePasswordSilent(alias)

	cmd := exec.Command("/usr/bin/security", "add-generic-password",
		"-s", keychainService,
		"-a", alias,
		"-w", password,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store password in Keychain: %w", err)
	}
	return nil
}

// KeychainGetPassword retrieves a password from macOS Keychain
func KeychainGetPassword(alias string) (string, bool) {
	cmd := exec.Command("/usr/bin/security", "find-generic-password",
		"-s", keychainService,
		"-a", alias,
		"-w",
	)
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// KeychainDeletePassword deletes a password from macOS Keychain
func KeychainDeletePassword(alias string) error {
	cmd := exec.Command("/usr/bin/security", "delete-generic-password",
		"-s", keychainService,
		"-a", alias,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("password not found in Keychain: %w", err)
	}
	return nil
}

// KeychainDeletePasswordSilent deletes silently (ignores errors)
func KeychainDeletePasswordSilent(alias string) {
	_ = KeychainDeletePassword(alias)
}
