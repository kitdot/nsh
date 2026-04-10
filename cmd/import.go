package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:     "import [file]",
	Aliases: []string{"imp"},
	Short:   "Import hosts from file",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Step 1: Get file path
		filePath := ""
		if len(args) > 0 {
			filePath = args[0]
		} else {
			filePath = bridge.PromptPath("Import file path:")
			if filePath == "" {
				return nil
			}
		}
		filePath = core.ExpandPath(filePath)

		// Step 2: Read file
		rawData, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Step 3: Detect format and decode
		var data core.ExportData
		isEncrypted := len(rawData) >= 4 && rawData[0] == 'N' && rawData[1] == 'S' && rawData[2] == 'H' && (rawData[3] == 0x01 || rawData[3] == 0x02)

		if isEncrypted {
			// Touch ID first
			fmt.Println("Touch ID required to import sensitive data...")
			if err := core.AuthenticateTouchID("nsh needs to import passwords and key files"); err != nil {
				fmt.Printf("Authentication failed: %v\n", err)
				return nil
			}
			fmt.Println("Authenticated.")

			// Decrypt
			pr := bridge.PromptPassword("Encryption password", false)
			if pr.Cancelled {
				fmt.Println("Cancelled.")
				return nil
			}
			if pr.Value == "" {
				fmt.Println("Password is required.")
				return nil
			}

			jsonData, err := core.DecryptExport(rawData, pr.Value)
			if err != nil {
				return fmt.Errorf("decryption failed: %v", err)
			}

			if err := json.Unmarshal(jsonData, &data); err != nil {
				return fmt.Errorf("invalid export data: %w", err)
			}
		} else {
			// Plain JSON
			if err := json.Unmarshal(rawData, &data); err != nil {
				return fmt.Errorf("invalid JSON file: %w", err)
			}
		}

		// For plain JSON that still carries secrets, require auth as well.
		if !isEncrypted && data.Secrets != nil {
			fmt.Println("Touch ID required to import sensitive data...")
			if err := core.AuthenticateTouchID("nsh needs to import passwords and key files"); err != nil {
				fmt.Printf("Authentication failed: %v\n", err)
				return nil
			}
			fmt.Println("Authenticated.")
		}

		if len(data.Hosts) == 0 {
			fmt.Println("No hosts found in export file.")
			return nil
		}

		// Step 4: Load current config
		manager := core.NewConfigManager(sshConfigFlag)
		cfg, err := manager.Load()
		if err != nil {
			return err
		}

		// Step 5: Show what will be imported
		fmt.Printf("\nFound %d hosts in export file", len(data.Hosts))
		if data.Secrets != nil {
			fmt.Printf(" (with %d passwords, %d keys)", len(data.Secrets.Passwords), len(collectImportedKeyFiles(&data)))
		}
		fmt.Println(":")
		fmt.Println()

		for _, h := range data.Hosts {
			status := "  new"
			if cfg.HostByAlias(h.Alias) != nil {
				status = "  conflict"
			}
			desc := h.HostName
			if h.Desc != "" {
				desc += " - " + h.Desc
			}
			fmt.Printf("  %s  %s  (%s)\n", status, h.Alias, desc)
		}
		fmt.Println()

		// Step 6: Handle conflicts
		conflictActions := map[string]string{}
		hasConflicts := false
		for _, h := range data.Hosts {
			if cfg.HostByAlias(h.Alias) != nil {
				hasConflicts = true
				break
			}
		}

		if hasConflicts {
			// Ask for global conflict strategy first
			strategy := bridge.Pick("Handle conflicts", []bridge.PickerOption{
				{Value: "ask", Label: "Ask for each"},
				{Value: "skip_all", Label: "Skip all conflicts"},
				{Value: "overwrite_all", Label: "Overwrite all conflicts"},
				{Value: "rename_all", Label: "Rename all conflicts"},
			})
			if strategy == "" {
				fmt.Println("Cancelled.")
				return nil
			}

			for _, h := range data.Hosts {
				if cfg.HostByAlias(h.Alias) == nil {
					continue
				}

				switch strategy {
				case "skip_all":
					conflictActions[h.Alias] = "skip"
				case "overwrite_all":
					conflictActions[h.Alias] = "overwrite"
				case "rename_all":
					conflictActions[h.Alias] = "rename"
				case "ask":
					action := bridge.Pick(
						fmt.Sprintf("'%s' already exists (%s)", h.Alias, h.HostName),
						[]bridge.PickerOption{
							{Value: "skip", Label: "Skip"},
							{Value: "overwrite", Label: "Overwrite existing"},
							{Value: "rename", Label: "Import as renamed copy"},
						},
					)
					if action == "" {
						fmt.Println("Cancelled.")
						return nil
					}
					conflictActions[h.Alias] = action
				}
			}
		}

		// Step 7: Confirm
		confirm := bridge.Pick("Proceed with import?", []bridge.PickerOption{
			{Value: "yes", Label: "Yes, import"},
			{Value: "no", Label: "Cancel"},
		})
		if confirm != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}

		// Step 8: Import hosts into config
		newConfig, importResult := core.ImportHosts(cfg, &data, conflictActions)

		// Step 9: Restore secrets
		if data.Secrets != nil {
			// Restore passwords to Keychain
			for alias, pw := range data.Secrets.Passwords {
				// Check if host was skipped
				if importResult.Skipped[alias] {
					continue
				}
				// Resolve actual alias (may have been renamed)
				actualAlias, ok := importResult.AliasMap[alias]
				if !ok {
					continue
				}
				if err := core.KeychainSetPassword(pw, actualAlias); err != nil {
					fmt.Printf("Warning: failed to set password for '%s': %v\n", actualAlias, err)
				}
			}

			keyDir := core.ExpandPath("~/.ssh/nsh")
			keyRefs, keyLogs, err := restoreImportedKeyFiles(&data, keyDir)
			if err != nil {
				return fmt.Errorf("failed to restore key files: %w", err)
			}
			for _, log := range keyLogs {
				fmt.Println(log)
			}

			aliasRefs := aliasKeyRefsForImport(&data, importResult, keyRefs)
			newConfig = applyImportedKeyRefs(newConfig, aliasRefs)
		}

		// Step 10: Save config
		if err := manager.Save(newConfig); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		// Summary
		fmt.Println()
		for _, log := range importResult.Logs {
			fmt.Println("  " + log)
		}
		fmt.Printf("\nImport complete.\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
