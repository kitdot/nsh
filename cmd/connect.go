package cmd

import (
	"fmt"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/connect"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:     "conn [alias]",
	Aliases: []string{"c"},
	Short:   "Connect to a host via SSH",
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
			alias = bridge.TreeBrowser("Connect to", cfg).Alias
			if alias == "" {
				return nil
			}
		}

		host := cfg.HostByAlias(alias)
		if host == nil {
			return fmt.Errorf("host '%s' not found", alias)
		}

		connect.Exec(host)
		return nil
	},
	ValidArgsFunction: completeHostAliases,
}

func init() {
	rootCmd.AddCommand(connectCmd)
}
