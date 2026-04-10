package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:     "new",
	Aliases: []string{"n"},
	Short:   "Create a new SSH host",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := core.NewConfigManager(sshConfigFlag)
		cfg, err := manager.Load()
		if err != nil {
			return err
		}

		fmt.Printf("\033[2;90m─────────────────────────────────────\033[0m\n")
		fmt.Printf("\033[36mCreate a new SSH host\033[0m \033[2m(Esc to cancel at any step)\033[0m\n")
		fmt.Printf("\033[2;90m─────────────────────────────────────\033[0m\n")

		// 1. Alias (required, must be unique, safe characters only)
		alias, cancelled := promptAlias(aliasPromptOptions{
			Label:           "Alias (Host name)",
			Required:        true,
			RequiredMessage: "Alias is required.",
			AliasAlreadyExists: func(v string) bool {
				return cfg.HostByAlias(v) != nil
			},
		})
		if cancelled {
			fmt.Println("Cancelled.")
			return nil
		}

		// 2-7. Host form
		form, cancelled, aborted, err := collectHostForm(hostFormOptions{
			HostNameLabel:         "HostName (IP/Domain)",
			HostNameRequired:      true,
			UserDefault:           "root",
			PortDefault:           "22",
			NormalizeIdentityFile: handleKeyFile,
			PasswordPrompt: func(auth string) hostFormPasswordResult {
				if auth != "password" {
					return hostFormPasswordResult{}
				}
				password, cancelled, invalid := promptAuthPassword(alias, false, true, "Password is required for password auth. Cancelled.")
				return hostFormPasswordResult{
					PendingPassword: password,
					Cancelled:       cancelled,
					Abort:           invalid,
				}
			},
		})
		if cancelled {
			fmt.Println("Cancelled.")
			return nil
		}
		if aborted {
			return nil
		}
		if err != nil {
			return err
		}

		// 8. Group
		group := selectGroup(cfg.Groups())

		// 9. Description
		desc, cancelled := promptDescription("")
		if cancelled {
			fmt.Println("Cancelled.")
			return nil
		}

		// 10. Preview & confirm
		host := &core.NSHHost{
			Group:        ifEmpty(group, "Uncategorized"),
			Desc:         desc,
			Auth:         form.Auth,
			Alias:        alias,
			HostName:     form.HostName,
			User:         form.User,
			Port:         form.Port,
			IdentityFile: form.IdentityFile,
		}

		printHostPreview(host)

		if !confirmSave("Confirm") {
			fmt.Println("Cancelled.")
			return nil
		}

		newBlocks := appendManagedHostBlock(cfg.Blocks, host)
		newConfig := newConfigWithMetadata(newBlocks, cfg.GroupOrder, cfg.PinnedAliases)
		var mutations []keychainMutation
		if form.PendingPassword != "" {
			mutations = append(mutations, setPasswordMutation(alias, form.PendingPassword))
		}
		if err := applyConfigAndKeychainMutations(manager, cfg, newConfig, mutations); err != nil {
			return err
		}

		fmt.Printf("Host '%s' created.\n", alias)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func selectSSHKey() string {
	skip := map[string]bool{
		"config": true, "known_hosts": true, "known_hosts.old": true,
		"authorized_keys": true, "environment": true,
	}

	isKeyFile := func(name string) bool {
		if strings.HasPrefix(name, ".") || skip[name] {
			return false
		}
		if strings.HasSuffix(name, ".pub") || strings.Contains(name, ".bak") ||
			strings.Contains(name, ".nsh.") || strings.HasSuffix(name, ".tmp") {
			return false
		}
		return true
	}

	type keyEntry struct {
		display string // e.g. "~/.ssh/nsh/id_rsa"
		value   string
	}
	var keys []keyEntry

	// Scan ~/.ssh/nsh/ first (nsh-managed keys)
	nshDir := core.ExpandPath("~/.ssh/nsh")
	if entries, err := os.ReadDir(nshDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !isKeyFile(e.Name()) {
				continue
			}
			keys = append(keys, keyEntry{
				display: "~/.ssh/nsh/" + e.Name(),
				value:   "~/.ssh/nsh/" + e.Name(),
			})
		}
	}

	// Scan ~/.ssh/ (user keys)
	sshDir := core.ExpandPath("~/.ssh")
	if entries, err := os.ReadDir(sshDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !isKeyFile(e.Name()) {
				continue
			}
			keys = append(keys, keyEntry{
				display: "~/.ssh/" + e.Name(),
				value:   "~/.ssh/" + e.Name(),
			})
		}
	}

	if len(keys) == 0 {
		return bridge.PromptPath("IdentityFile path (tab to complete):")
	}

	options := []bridge.PickerOption{
		{Value: "", Label: "Skip / Enter path manually"},
	}
	for _, k := range keys {
		desc := ""
		if strings.HasSuffix(k.display, ".pem") {
			desc = "pem"
		}
		options = append(options, bridge.PickerOption{
			Value:       k.value,
			Label:       k.display,
			Description: desc,
		})
	}

	return bridge.Pick("Select SSH key", options)
}

func selectGroup(existingGroups []string) string {
	return promptGroupSelection(groupPromptOptions{
		ExistingGroups: existingGroups,
		NoneValue:      "",
		EscValue:       "",
	})
}

func promptNewGroupName() string {
	reserved := map[string]bool{"none": true, "-": true, "uncategorized": true}
	for {
		r := promptText("Group name", "")
		if r.Cancelled || r.Value == "" {
			return ""
		}
		if reserved[strings.ToLower(r.Value)] {
			fmt.Printf("'%s' is a reserved name, please choose another.\n", r.Value)
			continue
		}
		if containsComma(r.Value) {
			fmt.Println("Group name cannot contain commas.")
			continue
		}
		return r.Value
	}
}

func handleKeyFile(identityFile string) (string, error) {
	expandedPath := core.ExpandPath(identityFile)
	nshDir := core.ExpandPath("~/.ssh/nsh")

	info, err := os.Stat(expandedPath)
	if err != nil {
		return identityFile, nil
	}

	// Fix permissions
	if info.Mode().Perm()&0077 != 0 {
		if err := os.Chmod(expandedPath, 0600); err != nil {
			return "", fmt.Errorf("failed to set key permissions on %s: %w", expandedPath, err)
		}
		fmt.Printf("Fixed permissions: %s → 0600\n", identityFile)
	}

	// Copy to ~/.ssh/nsh/ if not already there
	if !strings.HasPrefix(expandedPath, nshDir) {
		fileName := filepath.Base(expandedPath)
		destPath := filepath.Join(nshDir, fileName)

		copyOpts := []bridge.PickerOption{
			{Value: "yes", Label: fmt.Sprintf("Copy %s to ~/.ssh/nsh/", fileName)},
			{Value: "no", Label: "Keep original path"},
		}
		pick := bridge.Pick("Key file location", copyOpts)
		if pick == "yes" {
			if err := os.MkdirAll(nshDir, 0700); err != nil {
				return "", fmt.Errorf("failed to create key directory %s: %w", nshDir, err)
			}
			if _, err := os.Stat(destPath); err == nil {
				fmt.Printf("~/.ssh/nsh/%s already exists, using existing file.\n", fileName)
			} else if !os.IsNotExist(err) {
				return "", fmt.Errorf("failed to check key destination %s: %w", destPath, err)
			} else {
				if err := copyFileSimple(expandedPath, destPath); err != nil {
					return "", fmt.Errorf("failed to copy key %s to %s: %w", expandedPath, destPath, err)
				}
				fmt.Printf("Copied → ~/.ssh/nsh/%s\n", fileName)
			}
			if err := os.Chmod(destPath, 0600); err != nil {
				return "", fmt.Errorf("failed to set key permissions on %s: %w", destPath, err)
			}
			return "~/.ssh/nsh/" + fileName, nil
		}
	}

	return identityFile, nil
}

func ifEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func copyFileSimple(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return nil
}
