package cmd

import (
	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/connect"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

var pinnedCmd = &cobra.Command{
	Use:     "pin",
	Aliases: []string{"p"},
	Short:   "Browse and connect to pinned hosts",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := core.NewConfigManager(sshConfigFlag)
		cfg, err := manager.Load()
		if err != nil {
			return err
		}

		for {
			result := bridge.TreeBrowserPinned("nsh", cfg)

			// Save pinned changes if modified
			if !slicesEqual(result.PinnedAliases, cfg.PinnedAliases) {
				if err := savePinnedAliases(cfg, manager, result.PinnedAliases); err != nil {
					return err
				}
				cfg, err = manager.Load()
				if err != nil {
					return err
				}
			}

			switch result.Action {
			case "connect":
				host := cfg.HostByAlias(result.Alias)
				if host == nil {
					continue
				}
				connect.Exec(host)
				return nil

			case "edit":
				editCmd.SetArgs([]string{result.Alias})
				return editCmd.RunE(editCmd, []string{result.Alias})

			case "delete":
				err := deleteHostWithOptions(result.Alias, cfg, manager, true)
				if err != nil {
					return err
				}
				cfg, err = manager.Load()
				if err != nil {
					return err
				}
				continue

			case "new":
				return addCmd.RunE(addCmd, nil)

			default:
				return nil
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(pinnedCmd)
}
