package cmd

import "github.com/kitdot/nsh/core"

func replacePinnedAlias(pinned []string, oldAlias, newAlias string) ([]string, bool) {
	if oldAlias == "" || oldAlias == newAlias {
		return append([]string(nil), pinned...), false
	}

	updated := append([]string(nil), pinned...)
	changed := false
	for i, alias := range updated {
		if alias == oldAlias {
			updated[i] = newAlias
			changed = true
		}
	}
	return updated, changed
}

func removePinnedAliases(pinned []string, aliases ...string) ([]string, bool) {
	if len(pinned) == 0 {
		return nil, false
	}

	toRemove := make(map[string]bool)
	for _, alias := range aliases {
		if alias != "" {
			toRemove[alias] = true
		}
	}
	if len(toRemove) == 0 {
		return append([]string(nil), pinned...), false
	}

	var updated []string
	changed := false
	for _, alias := range pinned {
		if toRemove[alias] {
			changed = true
			continue
		}
		updated = append(updated, alias)
	}
	return updated, changed
}

func newConfigWithMetadata(blocks []core.NSHBlock, groupOrder, pinned []string) *core.NSHConfig {
	blockCopy := make([]core.NSHBlock, len(blocks))
	copy(blockCopy, blocks)

	groupCopy := make([]string, len(groupOrder))
	copy(groupCopy, groupOrder)

	pinnedCopy := make([]string, len(pinned))
	copy(pinnedCopy, pinned)

	return &core.NSHConfig{
		Blocks:        updatePinnedLine(blockCopy, pinnedCopy),
		GroupOrder:    groupCopy,
		PinnedAliases: pinnedCopy,
	}
}
