package cmd

import (
	"fmt"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:     "edit [alias]",
	Aliases: []string{"e"},
	Short:   "Edit an existing SSH host",
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
			alias = bridge.TreeBrowser("Edit host", cfg).Alias
			if alias == "" {
				return nil
			}
		}

		host := cfg.HostByAlias(alias)
		if host == nil {
			return fmt.Errorf("host '%s' not found", alias)
		}
		if host.IsWildcard {
			return fmt.Errorf("cannot edit 'Host *' via nsh. Edit ~/.ssh/config directly")
		}

		fmt.Printf("\033[2;90m─────────────────────────────────────\033[0m\n")
		fmt.Printf("\033[36mEditing '%s'\033[0m \033[2m(Esc to cancel at any step)\033[0m\n", alias)
		fmt.Printf("\033[2;90m─────────────────────────────────────\033[0m\n")

		// 1. Alias
		newAlias, cancelled := promptAlias(aliasPromptOptions{
			Label:              "Alias",
			DefaultValue:       host.Alias,
			Required:           true,
			AllowExistingAlias: alias,
			AliasAlreadyExists: func(v string) bool {
				return cfg.HostByAlias(v) != nil
			},
		})
		if cancelled {
			fmt.Println("Cancelled.")
			return nil
		}

		// 2-7. Host form
		currentIdentityFile := ""
		if host.Auth == "key" {
			currentIdentityFile = host.IdentityFile
		}
		form, cancelled, aborted, err := collectHostForm(hostFormOptions{
			HostNameLabel:          "HostName",
			HostNameDefault:        host.HostName,
			UserDefault:            host.User,
			PortDefault:            host.Port,
			CurrentAuth:            host.Auth,
			IdentityFileDefault:    host.IdentityFile,
			IdentityFilePromptFrom: currentIdentityFile,
			PasswordPrompt: func(auth string) hostFormPasswordResult {
				if auth != "password" {
					return hostFormPasswordResult{}
				}
				hasExisting := host.Auth == "password"
				password, cancelled, invalid := promptAuthPassword(newAlias, hasExisting, !hasExisting, "Password is required. Cancelled.")
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
		newGroup := selectGroupForEdit(host.Group, cfg.Groups())

		// 9. Description
		newDesc, cancelled := promptDescription(host.Desc)
		if cancelled {
			fmt.Println("Cancelled.")
			return nil
		}

		// Preview
		updatedHost := &core.NSHHost{
			Group:        newGroup,
			Desc:         newDesc,
			Auth:         form.Auth,
			Order:        host.Order,
			Alias:        newAlias,
			HostName:     form.HostName,
			User:         form.User,
			Port:         form.Port,
			IdentityFile: form.IdentityFile,
		}

		printHostPreview(updatedHost)

		if !confirmSave("Confirm") {
			fmt.Println("Cancelled.")
			return nil
		}

		existingPassword, hasExistingPassword := keychainGetPassword(alias)

		newBlocks := replaceManagedHostBlock(cfg.Blocks, alias, updatedHost)

		pinnedAliases := cfg.PinnedAliases
		if newAlias != alias {
			pinnedAliases, _ = replacePinnedAlias(cfg.PinnedAliases, alias, newAlias)
		}

		newConfig := newConfigWithMetadata(newBlocks, cfg.GroupOrder, pinnedAliases)
		mutations := buildEditKeychainMutations(
			alias,
			newAlias,
			host.Auth,
			form.Auth,
			form.PendingPassword,
			existingPassword,
			hasExistingPassword,
		)
		if err := applyConfigAndKeychainMutations(manager, cfg, newConfig, mutations); err != nil {
			return err
		}
		fmt.Printf("Host '%s' updated.\n", newAlias)
		return nil
	},
	ValidArgsFunction: completeHostAliases,
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func selectGroupForEdit(currentGroup string, existingGroups []string) string {
	return promptGroupSelection(groupPromptOptions{
		ExistingGroups: existingGroups,
		CurrentGroup:   currentGroup,
		MarkCurrent:    true,
		NoneValue:      "Uncategorized",
		EscValue:       currentGroup,
	})
}
