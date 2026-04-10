package cmd

import (
	"fmt"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

var copyCmd = &cobra.Command{
	Use:     "copy [alias]",
	Aliases: []string{"cp"},
	Short:   "Duplicate a host and edit as new",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := core.NewConfigManager(sshConfigFlag)
		cfg, err := manager.Load()
		if err != nil {
			return err
		}

		// Step 1: Select source host
		var source *core.NSHHost
		if len(args) > 0 {
			source = cfg.HostByAlias(args[0])
			if source == nil {
				return fmt.Errorf("host '%s' not found", args[0])
			}
		} else {
			selected := bridge.TreeBrowser("Select host to copy", cfg).Alias
			if selected == "" {
				return nil
			}
			source = cfg.HostByAlias(selected)
			if source == nil {
				return nil
			}
		}

		fmt.Printf("\033[2;90m─────────────────────────────────────\033[0m\n")
		fmt.Printf("\033[36mCopying '%s'\033[0m \033[2m(Esc to cancel at any step)\033[0m\n", source.Alias)
		fmt.Printf("\033[2;90m─────────────────────────────────────\033[0m\n")

		// Step 2: New alias (required, must be different, safe characters only)
		existingAliases := make([]string, 0, len(cfg.Hosts()))
		for _, h := range cfg.Hosts() {
			existingAliases = append(existingAliases, h.Alias)
		}
		defaultAlias := nextAvailableAlias(source.Alias, existingAliases)

		newAlias, cancelled := promptAlias(aliasPromptOptions{
			Label:            "New alias",
			DefaultValue:     defaultAlias,
			Required:         true,
			DisallowAlias:    source.Alias,
			DisallowAliasMsg: "New alias must be different from the original.",
			AliasAlreadyExists: func(v string) bool {
				return cfg.HostByAlias(v) != nil
			},
		})
		if cancelled {
			fmt.Println("Cancelled.")
			return nil
		}

		// Step 3-6: Host form
		form, cancelled, aborted, err := collectHostForm(hostFormOptions{
			HostNameLabel:          "HostName",
			HostNameDefault:        source.HostName,
			UserDefault:            source.User,
			PortDefault:            source.Port,
			CurrentAuth:            source.Auth,
			IdentityFileDefault:    source.IdentityFile,
			IdentityFilePromptFrom: source.IdentityFile,
			PasswordPrompt: func(auth string) hostFormPasswordResult {
				if auth != "password" {
					return hostFormPasswordResult{}
				}
				if source.Auth == "password" {
					pick := bridge.Pick("Password", []bridge.PickerOption{
						{Value: "__keep__", Label: "Keep same password"},
						{Value: "__change__", Label: "Set new password"},
					})
					if pick == "" {
						return hostFormPasswordResult{Cancelled: true}
					}
					if pick == "__keep__" {
						if pw, ok := keychainGetPassword(source.Alias); ok {
							return hostFormPasswordResult{PendingPassword: pw}
						}
						return hostFormPasswordResult{}
					}
				}

				password, cancelled, invalid := promptAuthPassword(newAlias, false, true, "Password is required. Cancelled.")
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

		// Step 7: Group
		newGroup := selectGroupForEdit(source.Group, cfg.Groups())

		// Step 8: Description
		newDesc, cancelled := promptDescription(source.Desc)
		if cancelled {
			fmt.Println("Cancelled.")
			return nil
		}

		// Step 9: Preview
		host := &core.NSHHost{
			Group:        newGroup,
			Desc:         newDesc,
			Auth:         form.Auth,
			Alias:        newAlias,
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
			mutations = append(mutations, setPasswordMutation(newAlias, form.PendingPassword))
		}
		if err := applyConfigAndKeychainMutations(manager, cfg, newConfig, mutations); err != nil {
			return err
		}

		fmt.Printf("Host '%s' created (copied from '%s').\n", newAlias, source.Alias)
		return nil
	},
	ValidArgsFunction: completeHostAliases,
}

func init() {
	rootCmd.AddCommand(copyCmd)
}
