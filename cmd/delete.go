package cmd

import (
	"fmt"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

var (
	deleteIsGroup bool
	deleteYes     bool
)

var deleteCmd = &cobra.Command{
	Use:     "del [alias|group]",
	Aliases: []string{"d"},
	Short:   "Delete a host or an entire group",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := core.NewConfigManager(sshConfigFlag)
		cfg, err := manager.Load()
		if err != nil {
			return err
		}

		// Direct with --is-group flag
		if deleteIsGroup {
			target := ""
			if len(args) > 0 {
				target = args[0]
			} else {
				target = pickGroup("Delete group", cfg)
				if target == "" {
					return nil
				}
			}
			return deleteGroup(target, cfg, manager)
		}

		// Direct with alias argument
		if len(args) > 0 {
			return deleteHost(args[0], cfg, manager)
		}

		// No args — ask what to delete
		pick := bridge.Pick("Delete", []bridge.PickerOption{
			{Value: "host", Label: "Host", Description: "delete a single host"},
			{Value: "group", Label: "Group", Description: "delete an entire group"},
		})
		switch pick {
		case "host":
			r := bridge.TreeBrowser("Delete host", cfg)
			target := r.Alias
			if target == "" {
				return nil
			}
			return deleteHost(target, cfg, manager)
		case "group":
			target := pickGroup("Delete group", cfg)
			if target == "" {
				return nil
			}
			return deleteGroup(target, cfg, manager)
		}
		return nil
	},
	ValidArgsFunction: completeHostAliases,
}

func init() {
	deleteCmd.Flags().BoolVar(&deleteIsGroup, "is-group", false, "Delete an entire group")
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "Skip confirmation")
	rootCmd.AddCommand(deleteCmd)
}

func pickGroup(title string, cfg *core.NSHConfig) string {
	groups := cfg.SortedGroups()
	if len(groups) == 0 {
		fmt.Println("No groups found.")
		return ""
	}
	var options []bridge.PickerOption
	for _, g := range groups {
		count := len(cfg.HostsInGroup(g))
		label := g
		if g == "Uncategorized" {
			label = "-"
		}
		options = append(options, bridge.PickerOption{
			Value:       g,
			Label:       label,
			Description: fmt.Sprintf("%d hosts", count),
		})
	}
	return bridge.Pick(title, options)
}

func deleteHost(alias string, cfg *core.NSHConfig, manager *core.NSHConfigManager) error {
	return deleteHostWithOptions(alias, cfg, manager, deleteYes)
}

func deleteHostWithOptions(alias string, cfg *core.NSHConfig, manager *core.NSHConfigManager, skipConfirm bool) error {
	host := cfg.HostByAlias(alias)
	if host == nil {
		return fmt.Errorf("host '%s' not found", alias)
	}
	if host.IsWildcard {
		return fmt.Errorf("cannot delete 'Host *' (global-default)")
	}

	if !skipConfirm {
		info := alias
		if host.HostName != "" {
			info = fmt.Sprintf("%s (%s)", alias, host.HostName)
		}
		confirm := bridge.Pick(fmt.Sprintf("Delete %s?", info), []bridge.PickerOption{
			{Value: "no", Label: "No, cancel"},
			{Value: "yes", Label: "Yes, delete"},
		})
		if confirm != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	var newBlocks []core.NSHBlock
	for _, b := range cfg.Blocks {
		if b.Type == core.BlockHost && b.Host != nil && b.Host.Alias == alias {
			continue
		}
		newBlocks = append(newBlocks, b)
	}
	newBlocks = cleanupBlankLines(newBlocks)

	newPinned, _ := removePinnedAliases(cfg.PinnedAliases, alias)
	newConfig := newConfigWithMetadata(newBlocks, cfg.GroupOrder, newPinned)
	if err := manager.Save(newConfig); err != nil {
		return err
	}

	if host.Auth == "password" {
		core.KeychainDeletePasswordSilent(alias)
	}

	fmt.Printf("Host '%s' deleted.\n", alias)
	return nil
}

func deleteGroup(groupName string, cfg *core.NSHConfig, manager *core.NSHConfigManager) error {
	hostsInGroup := cfg.HostsInGroup(groupName)
	if len(hostsInGroup) == 0 {
		return fmt.Errorf("group '%s' not found or empty", groupName)
	}

	for _, h := range hostsInGroup {
		if h.IsWildcard {
			return fmt.Errorf("group '%s' contains 'Host *' (global-default), cannot delete", groupName)
		}
	}

	// Show hosts in group
	fmt.Printf("Group '%s' contains %d hosts:\n", groupName, len(hostsInGroup))
	for _, h := range hostsInGroup {
		info := ""
		if h.HostName != "" {
			info = fmt.Sprintf(" (%s)", h.HostName)
		}
		fmt.Printf("  - %s%s\n", h.Alias, info)
	}
	fmt.Println()

	// Ask what to do
	var action string
	if deleteYes {
		action = "delete_all"
	} else {
		action = bridge.Pick(fmt.Sprintf("Delete group '%s'", groupName), []bridge.PickerOption{
			{Value: "ungroup", Label: "Remove group only", Description: "hosts become Uncategorized"},
			{Value: "delete_all", Label: "Delete group and all hosts"},
			{Value: "cancel", Label: "Cancel"},
		})
	}

	switch action {
	case "ungroup":
		// Move hosts to Uncategorized, remove group
		newBlocks := make([]core.NSHBlock, len(cfg.Blocks))
		copy(newBlocks, cfg.Blocks)

		for i, b := range newBlocks {
			if b.Type == core.BlockHost && b.Host != nil && b.Host.Group == groupName {
				h := *b.Host
				h.Group = "Uncategorized"
				nshLine := core.BuildNshLine(&h)
				newBlocks[i] = core.NSHBlock{
					Type:     core.BlockHost,
					Host:     &h,
					NshLine:  nshLine,
					HostLine: b.HostLine,
				}
			}
		}

		var newGroupOrder []string
		for _, g := range cfg.GroupOrder {
			if g != groupName {
				newGroupOrder = append(newGroupOrder, g)
			}
		}

		newConfig := newConfigWithMetadata(newBlocks, newGroupOrder, cfg.PinnedAliases)
		if err := manager.Save(newConfig); err != nil {
			return err
		}
		fmt.Printf("Group '%s' removed. %d hosts moved to Uncategorized.\n", groupName, len(hostsInGroup))

	case "delete_all":
		if !deleteYes {
			confirm := bridge.Pick("This will permanently delete all hosts. Are you sure?", []bridge.PickerOption{
				{Value: "no", Label: "No, cancel"},
				{Value: "yes", Label: "Yes, delete all"},
			})
			if confirm != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		aliasesToDelete := map[string]bool{}
		for _, h := range hostsInGroup {
			aliasesToDelete[h.Alias] = true
		}

		var newBlocks []core.NSHBlock
		for _, b := range cfg.Blocks {
			if b.Type == core.BlockHost && b.Host != nil && aliasesToDelete[b.Host.Alias] {
				continue
			}
			newBlocks = append(newBlocks, b)
		}
		newBlocks = cleanupBlankLines(newBlocks)

		var newGroupOrder []string
		for _, g := range cfg.GroupOrder {
			if g != groupName {
				newGroupOrder = append(newGroupOrder, g)
			}
		}

		var aliases []string
		for alias := range aliasesToDelete {
			aliases = append(aliases, alias)
		}
		newPinned, _ := removePinnedAliases(cfg.PinnedAliases, aliases...)
		newConfig := newConfigWithMetadata(newBlocks, newGroupOrder, newPinned)
		if err := manager.Save(newConfig); err != nil {
			return err
		}

		for _, h := range hostsInGroup {
			if h.Auth == "password" {
				core.KeychainDeletePasswordSilent(h.Alias)
			}
		}

		fmt.Printf("Group '%s' deleted (%d hosts removed).\n", groupName, len(hostsInGroup))

	default:
		fmt.Println("Cancelled.")
	}
	return nil
}

func cleanupBlankLines(blocks []core.NSHBlock) []core.NSHBlock {
	var result []core.NSHBlock
	lastWasBlank := false

	for _, b := range blocks {
		if b.Type == core.BlockBlank {
			if lastWasBlank {
				continue
			}
			lastWasBlank = true
		} else {
			lastWasBlank = false
		}
		result = append(result, b)
	}

	for len(result) > 0 && result[len(result)-1].Type == core.BlockBlank {
		result = result[:len(result)-1]
	}

	return result
}
