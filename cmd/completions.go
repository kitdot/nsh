package cmd

import (
	"fmt"

	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

// completeHostAliases provides shell completion for host aliases with IP and description.
// Includes hosts from both nsh config and main SSH config.
func completeHostAliases(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	manager := loadManagerForCompletion()
	if manager == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, h := range manager.LoadAllHosts() {
		if h.IsWildcard {
			continue
		}
		desc := h.HostName
		if h.Desc != "" {
			desc = fmt.Sprintf("%s - %s", h.HostName, h.Desc)
		}
		if desc == "" {
			desc = h.Group
		}
		completions = append(completions, fmt.Sprintf("%s\t%s", h.Alias, desc))
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeGroupNames provides shell completion for group names with host count
func completeGroupNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg := loadConfigForCompletion()
	if cfg == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, g := range cfg.Groups() {
		count := len(cfg.HostsInGroup(g))
		completions = append(completions, fmt.Sprintf("%s\t%d hosts", g, count))
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func loadManagerForCompletion() *core.NSHConfigManager {
	manager := core.NewConfigManager(sshConfigFlag)
	_, err := manager.Load()
	if err != nil {
		return nil
	}
	return manager
}

func loadConfigForCompletion() *core.NSHConfig {
	manager := loadManagerForCompletion()
	if manager == nil {
		return nil
	}
	return manager.GetConfig()
}
