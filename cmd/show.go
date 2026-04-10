package cmd

import (
	"fmt"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:     "show [alias]",
	Aliases: []string{"s"},
	Short:   "Show detailed configuration for a host",
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
			alias = bridge.TreeBrowser("Show host", cfg).Alias
			if alias == "" {
				return nil
			}
		}

		host := cfg.HostByAlias(alias)
		if host == nil {
			return fmt.Errorf("host '%s' not found", alias)
		}

		printHost(host)
		return nil
	},
	ValidArgsFunction: completeHostAliases,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

const (
	cLine      = "\033[2;90m" // dim gray
	cLineReset = "\033[0m"
)

func printHost(host *core.NSHHost) {
	fmt.Println()
	fmt.Printf("%s─────────────────────────────%s\n", cLine, cLineReset)
	fmt.Println()
	fmt.Printf("Host: %s\n", host.Alias)
	if host.HostName != "" {
		fmt.Printf("  HostName:     %s\n", host.HostName)
	}
	if host.User != "" {
		fmt.Printf("  User:         %s\n", host.User)
	}
	if host.Port != "" {
		fmt.Printf("  Port:         %s\n", host.Port)
	}
	if host.IdentityFile != "" {
		fmt.Printf("  IdentityFile: %s\n", host.IdentityFile)
	}
	if host.Auth != "" {
		fmt.Printf("  Auth:         %s\n", host.Auth)
	}
	fmt.Printf("  Group:        %s\n", host.Group)
	if host.Order > 0 {
		fmt.Printf("  Order:        %d\n", host.Order)
	}
	if host.Desc != "" {
		fmt.Printf("  Description:  %s\n", host.Desc)
	}
	if host.IsWildcard {
		fmt.Println("  [global-default]")
	}
	fmt.Println()
	fmt.Printf("%s─────────────────────────────%s\n", cLine, cLineReset)
	fmt.Println()
}
