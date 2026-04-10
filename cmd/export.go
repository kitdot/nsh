package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:     "export",
	Aliases: []string{"exp"},
	Short:   "Export hosts to file",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := core.NewConfigManager(sshConfigFlag)
		cfg, err := manager.Load()
		if err != nil {
			return err
		}

		hosts := cfg.Hosts()
		count := 0
		for _, h := range hosts {
			if !h.IsWildcard {
				count++
			}
		}
		if count == 0 {
			fmt.Println("No hosts to export.")
			return nil
		}

		// Step 1: Pick export type
		exportType := bridge.Pick("Export type", []bridge.PickerOption{
			{Value: "basic", Label: "Basic", Description: "config only (no passwords or keys)"},
			{Value: "full", Label: "Full", Description: "includes passwords & key files (requires Touch ID)"},
		})
		if exportType == "" {
			return nil
		}

		full := exportType == "full"

		// Step 2: Choose export scope (all or selected groups)
		var selectedGroups []string
		groups := cfg.SortedGroups()

		if len(groups) > 1 {
			scopeChoice := bridge.Pick("Export scope", []bridge.PickerOption{
				{Value: "all", Label: "All groups", Description: fmt.Sprintf("export all %d groups", len(groups))},
				{Value: "select", Label: "Select groups", Description: "choose which groups to export"},
			})
			if scopeChoice == "" {
				return nil
			}

			if scopeChoice == "select" {
				var opts []bridge.MultiPickerOption
				for _, g := range groups {
					hostCount := len(cfg.HostsInGroup(g))
					label := g
					if g == "Uncategorized" {
						label = "Uncategorized"
					}
					opts = append(opts, bridge.MultiPickerOption{
						Value:       g,
						Label:       label,
						Description: fmt.Sprintf("%d hosts", hostCount),
					})
				}

				selectedGroups = bridge.MultiPick("Select groups to export", opts)
				if selectedGroups == nil {
					return nil
				}
			}
		}
		// selectedGroups == nil means export all

		// Step 3: Choose output directory
		home, _ := os.UserHomeDir()
		timestamp := time.Now().Format("20060102_150405")
		ext := ".nsh.json"
		if full {
			ext = ".nsh.enc"
		}
		fileName := fmt.Sprintf("nsh_export_%s%s", timestamp, ext)

		dir := bridge.PromptPath("Export directory:")
		if dir == "" {
			dir = home
		}
		dir = core.ExpandPath(dir)

		outPath := filepath.Join(dir, fileName)

		// Step 4: Encryption password (full only)
		var password string
		if full {
			for {
				pr1 := bridge.PromptPassword("Set encryption password", false)
				if pr1.Cancelled {
					fmt.Println("Cancelled.")
					return nil
				}
				if pr1.Value == "" {
					fmt.Println("Password is required for encrypted export.")
					continue
				}

				pr2 := bridge.PromptPassword("Confirm password", false)
				if pr2.Cancelled {
					fmt.Println("Cancelled.")
					return nil
				}
				if pr1.Value != pr2.Value {
					fmt.Println("Passwords do not match. Please try again.")
					continue
				}
				password = pr1.Value
				break
			}
		}

		// Step 5: Touch ID → read secrets → build data (full only)
		if full {
			fmt.Println()
			fmt.Println("Touch ID required to access passwords and key files...")
			if err := core.AuthenticateTouchID("nsh needs to export passwords and key files"); err != nil {
				fmt.Printf("Authentication failed: %v\n", err)
				return nil
			}
		}

		data, err := core.BuildExportData(cfg, full, selectedGroups)
		if err != nil {
			return fmt.Errorf("failed to build export data: %w", err)
		}

		// Step 6: Serialize & write
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize: %w", err)
		}

		if full {
			encrypted, err := core.EncryptExport(jsonData, password)
			if err != nil {
				return fmt.Errorf("encryption failed: %w", err)
			}

			if err := os.WriteFile(outPath, encrypted, 0600); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
		} else {
			if err := os.WriteFile(outPath, jsonData, 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
		}

		// Summary
		fmt.Println()
		fmt.Printf("Exported %d hosts to %s\n", len(data.Hosts), outPath)
		if full && data.Secrets != nil {
			fmt.Printf("  Passwords: %d\n", len(data.Secrets.Passwords))
			fmt.Printf("  Key files: %d\n", exportedKeyFileCount(data))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}

func exportedKeyFileCount(data *core.ExportData) int {
	if data == nil || data.Secrets == nil {
		return 0
	}
	return len(collectImportedKeyFiles(data))
}
