package cmd

import (
	"fmt"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:     "auth [alias]",
	Aliases: []string{"au"},
	Short:   "Set or update authentication method for a host",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := core.NewConfigManager(sshConfigFlag)
		cfg, err := manager.Load()
		if err != nil {
			return err
		}

		alias := ""
		if len(args) > 0 {
			alias = args[0]
		} else {
			alias = bridge.TreeBrowser("Set auth for", cfg).Alias
			if alias == "" {
				return nil
			}
		}

		host := cfg.HostByAlias(alias)
		if host == nil {
			return fmt.Errorf("host '%s' not found", alias)
		}

		// Show current auth info
		currentAuth := host.Auth
		if currentAuth == "" {
			currentAuth = "none"
		}
		cCyan := "\033[36m"
		cDim := "\033[2m"
		cLine := "\033[2;90m"
		cRst := "\033[0m"
		fmt.Printf("%s─────────────────────────────────────%s\n", cLine, cRst)
		fmt.Printf("%sHost '%s'%s — current auth: %s%s%s", cCyan, alias, cRst, cCyan, currentAuth, cRst)
		if host.Auth == "password" {
			fmt.Printf(" %s(password in Keychain)%s", cDim, cRst)
		} else if host.Auth == "key" && host.IdentityFile != "" {
			fmt.Printf(" %s(%s)%s", cDim, host.IdentityFile, cRst)
		}
		fmt.Println()
		fmt.Printf("%s─────────────────────────────────────%s\n", cLine, cRst)

		// Pick auth method
		newAuth, cancelled := selectAuthMethod(host.Auth)
		if cancelled {
			return nil
		}

		// Handle each auth type
		newIdentityFile := host.IdentityFile

		var pendingPassword string

		switch newAuth {
		case "password":
			// Always prompt for password (defer Keychain write until after confirm)
			hasExisting := host.Auth == "password"
			var invalid bool
			pendingPassword, cancelled, invalid = promptAuthPassword(alias, hasExisting, !hasExisting, "Password is required. Cancelled.")
			if cancelled {
				fmt.Println("Cancelled.")
				return nil
			}
			if invalid {
				return nil
			}
			// Empty value with hasExisting = keep current

		case "key":
			var cancelled bool
			newIdentityFile, cancelled = promptIdentityFile(host.IdentityFile)
			if cancelled {
				fmt.Println("Cancelled.")
				return nil
			}
			fmt.Printf("IdentityFile: %s\n", newIdentityFile)

		default: // none
			// nothing extra needed
		}

		// Preview & confirm
		fmt.Println()
		fmt.Printf("  Host:  %s\n", alias)
		fmt.Printf("  Auth:  %s → %s\n", ifEmpty(host.Auth, "none"), ifEmpty(newAuth, "none"))
		if newAuth == "key" {
			fmt.Printf("  Key:   %s\n", newIdentityFile)
		}
		fmt.Println()

		if !confirmSave("Save changes?") {
			fmt.Println("Cancelled.")
			return nil
		}

		configChanged := newAuth != host.Auth || newIdentityFile != host.IdentityFile
		if configChanged {
			newBlocks := make([]core.NSHBlock, len(cfg.Blocks))
			copy(newBlocks, cfg.Blocks)
			for i, b := range newBlocks {
				if b.Type == core.BlockHost && b.Host != nil && b.Host.Alias == alias {
					h := *b.Host
					h.Auth = newAuth
					h.IdentityFile = newIdentityFile

					h.RawPropertyLines = rewriteIdentityFileLines(h.RawPropertyLines, newIdentityFile)
					newBlocks[i] = buildHostBlock(&h, b.HostLine)
					break
				}
			}
			newConfig := newConfigWithMetadata(newBlocks, cfg.GroupOrder, cfg.PinnedAliases)
			mutations := buildAuthKeychainMutations(alias, host.Auth, newAuth, pendingPassword)
			if err := applyConfigAndKeychainMutations(manager, cfg, newConfig, mutations); err != nil {
				return err
			}
			if pendingPassword != "" {
				fmt.Printf("Password saved to macOS Keychain for '%s'.\n", alias)
			}
			if host.Auth == "password" && newAuth != "password" {
				fmt.Println("Removed password from Keychain.")
			}

			if newAuth == "" {
				fmt.Println("Auth method cleared.")
			} else {
				fmt.Printf("Auth method set to '%s'.\n", newAuth)
			}
		} else if newAuth == "password" && pendingPassword != "" {
			// Auth type unchanged and only password changed.
			if err := keychainSetPassword(pendingPassword, alias); err != nil {
				return err
			}
			fmt.Printf("Password saved to macOS Keychain for '%s'.\n", alias)
			fmt.Println("Password updated.")
		} else {
			fmt.Println("No changes.")
		}

		return nil
	},
	ValidArgsFunction: completeHostAliases,
}

func init() {
	rootCmd.AddCommand(authCmd)
}
