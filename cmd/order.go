package cmd

import (
	"fmt"
	"strings"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

var orderCmd = &cobra.Command{
	Use:     "order",
	Aliases: []string{"o"},
	Short:   "Reorder groups or hosts within a group",
	RunE: func(cmd *cobra.Command, args []string) error {
		pick := bridge.Pick("Reorder", []bridge.PickerOption{
			{Value: "group", Label: "Groups", Description: "reorder group display order"},
			{Value: "host", Label: "Hosts", Description: "reorder hosts within a group"},
			{Value: "pinned", Label: "Pinned", Description: "reorder pinned hosts"},
		})
		switch pick {
		case "group":
			return orderGroupCmd.RunE(orderGroupCmd, nil)
		case "host":
			return orderHostCmd.RunE(orderHostCmd, nil)
		case "pinned":
			return orderPinnedCmd.RunE(orderPinnedCmd, nil)
		}
		return nil
	},
}

var orderGroupCmd = &cobra.Command{
	Use:   "group",
	Short: "Reorder groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := core.NewConfigManager(sshConfigFlag)
		cfg, err := manager.Load()
		if err != nil {
			return err
		}

		allGroups := cfg.SortedGroups()
		var groups []string
		for _, g := range allGroups {
			if g != "Uncategorized" {
				groups = append(groups, g)
			}
		}
		if len(groups) == 0 {
			fmt.Println("No groups to reorder.")
			return nil
		}

		// Build reorder items
		var items []bridge.ReorderItem
		for _, g := range groups {
			count := len(cfg.HostsInGroup(g))
			items = append(items, bridge.ReorderItem{
				Key:         g,
				Label:       g,
				Description: fmt.Sprintf("(%d hosts)", count),
			})
		}

		result := bridge.Reorder("Reorder groups", items)
		if result == nil {
			fmt.Println("Cancelled.")
			return nil
		}

		newOrder := make([]string, len(result))
		for i, item := range result {
			newOrder[i] = item.Key
		}

		confirm := bridge.Pick("Save new group order?", []bridge.PickerOption{
			{Value: "save", Label: "Save"},
			{Value: "cancel", Label: "Cancel"},
		})
		if confirm != "save" {
			fmt.Println("Cancelled.")
			return nil
		}

		newBlocks := updateGroupOrderLine(cfg.Blocks, newOrder)
		newConfig := &core.NSHConfig{Blocks: newBlocks, GroupOrder: newOrder}
		if err := manager.Save(newConfig); err != nil {
			return err
		}
		fmt.Println("Group order updated.")
		return nil
	},
}

var orderHostCmd = &cobra.Command{
	Use:               "host [group]",
	Short:             "Reorder hosts within a group",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeGroupNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := core.NewConfigManager(sshConfigFlag)
		cfg, err := manager.Load()
		if err != nil {
			return err
		}

		var selectedGroup string

		if len(args) > 0 {
			selectedGroup = args[0]
		} else {
			groups := cfg.SortedGroups()
			if len(groups) == 0 {
				fmt.Println("No groups found.")
				return nil
			}
			var options []bridge.PickerOption
			for _, g := range groups {
				count := len(cfg.HostsInGroup(g))
				options = append(options, bridge.PickerOption{
					Value:       g,
					Label:       g,
					Description: fmt.Sprintf("%d hosts", count),
				})
			}
			selected := bridge.Pick("Select a group", options)
			if selected == "" {
				fmt.Println("Cancelled.")
				return nil
			}
			selectedGroup = selected
		}

		hosts := cfg.HostsInGroup(selectedGroup)
		if len(hosts) == 0 {
			fmt.Printf("Group '%s' not found or empty.\n", selectedGroup)
			return nil
		}

		// Build reorder items
		var items []bridge.ReorderItem
		for _, h := range hosts {
			desc := ""
			if h.HostName != "" {
				desc = "(" + h.HostName + ")"
			}
			if h.Desc != "" {
				if desc != "" {
					desc += " - " + h.Desc
				} else {
					desc = h.Desc
				}
			}
			items = append(items, bridge.ReorderItem{
				Key:         h.Alias,
				Label:       h.Alias,
				Description: desc,
			})
		}

		title := fmt.Sprintf("Reorder hosts in [%s]", selectedGroup)
		result := bridge.Reorder(title, items)
		if result == nil {
			fmt.Println("Cancelled.")
			return nil
		}

		confirm := bridge.Pick("Save new host order?", []bridge.PickerOption{
			{Value: "save", Label: "Save"},
			{Value: "cancel", Label: "Cancel"},
		})
		if confirm != "save" {
			fmt.Println("Cancelled.")
			return nil
		}

		// Build alias → new order mapping
		aliasOrder := map[string]int{}
		for i, item := range result {
			aliasOrder[item.Key] = i + 1
		}

		newBlocks := make([]core.NSHBlock, len(cfg.Blocks))
		copy(newBlocks, cfg.Blocks)

		for i, b := range newBlocks {
			if b.Type == core.BlockHost && b.Host != nil {
				if newOrd, ok := aliasOrder[b.Host.Alias]; ok {
					updated := *b.Host
					updated.Order = newOrd
					nshLine := core.BuildNshLine(&updated)
					newBlocks[i] = core.NSHBlock{
						Type:     core.BlockHost,
						Host:     &updated,
						NshLine:  nshLine,
						HostLine: b.HostLine,
					}
				}
			}
		}

		newConfig := &core.NSHConfig{Blocks: newBlocks, GroupOrder: cfg.GroupOrder}
		if err := manager.Save(newConfig); err != nil {
			return err
		}
		fmt.Println("Host order updated.")
		return nil
	},
}

var orderPinnedCmd = &cobra.Command{
	Use:   "pinned",
	Short: "Reorder pinned hosts",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := core.NewConfigManager(sshConfigFlag)
		cfg, err := manager.Load()
		if err != nil {
			return err
		}

		if len(cfg.PinnedAliases) == 0 {
			fmt.Println("No pinned hosts to reorder.")
			return nil
		}

		var items []bridge.ReorderItem
		for _, alias := range cfg.PinnedAliases {
			desc := ""
			if h := cfg.HostByAlias(alias); h != nil {
				desc = "(" + h.HostName + ")"
				if h.Desc != "" {
					desc += " - " + h.Desc
				}
			}
			items = append(items, bridge.ReorderItem{
				Key:         alias,
				Label:       alias,
				Description: desc,
			})
		}

		result := bridge.Reorder("Reorder pinned hosts", items)
		if result == nil {
			fmt.Println("Cancelled.")
			return nil
		}

		confirm := bridge.Pick("Save new pinned order?", []bridge.PickerOption{
			{Value: "save", Label: "Save"},
			{Value: "cancel", Label: "Cancel"},
		})
		if confirm != "save" {
			fmt.Println("Cancelled.")
			return nil
		}

		newPinned := make([]string, len(result))
		for i, item := range result {
			newPinned[i] = item.Key
		}

		newBlocks := updatePinnedLine(cfg.Blocks, newPinned)
		newConfig := &core.NSHConfig{Blocks: newBlocks, GroupOrder: cfg.GroupOrder, PinnedAliases: newPinned}
		if err := manager.Save(newConfig); err != nil {
			return err
		}
		fmt.Println("Pinned order updated.")
		return nil
	},
}

func init() {
	orderCmd.AddCommand(orderGroupCmd)
	orderCmd.AddCommand(orderHostCmd)
	orderCmd.AddCommand(orderPinnedCmd)
	rootCmd.AddCommand(orderCmd)
}

func updateGroupOrderLine(blocks []core.NSHBlock, groupOrder []string) []core.NSHBlock {
	newBlocks := make([]core.NSHBlock, len(blocks))
	copy(newBlocks, blocks)

	groupLine := core.BuildGroupOrderLine(groupOrder)

	for i, b := range newBlocks {
		if b.Type == core.BlockComment && strings.HasPrefix(strings.ToLower(strings.TrimSpace(b.Raw)), "# nsh-groups:") {
			if groupLine != "" {
				newBlocks[i] = core.NSHBlock{Type: core.BlockComment, Raw: groupLine}
			} else {
				newBlocks = append(newBlocks[:i], newBlocks[i+1:]...)
			}
			return newBlocks
		}
	}

	if groupLine != "" {
		insertIndex := 0
		for insertIndex < len(newBlocks) {
			t := newBlocks[insertIndex].Type
			if t == core.BlockComment || t == core.BlockInclude || t == core.BlockBlank {
				insertIndex++
			} else {
				break
			}
		}
		tail := make([]core.NSHBlock, len(newBlocks[insertIndex:]))
		copy(tail, newBlocks[insertIndex:])
		newBlocks = append(newBlocks[:insertIndex],
			core.NSHBlock{Type: core.BlockComment, Raw: groupLine},
			core.NSHBlock{Type: core.BlockBlank, Raw: ""},
		)
		newBlocks = append(newBlocks, tail...)
	}

	return newBlocks
}
