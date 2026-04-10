package bridge

import "github.com/kitdot/nsh/core"

func newTreeModel(title string, cfg *core.NSHConfig, startPinned bool) treeModel {
	groups := buildTreeGroups(cfg)
	if len(groups) > 0 {
		groups[0].expanded = true
	}

	maxAlias, maxHost := 0, 0
	for _, g := range groups {
		for _, h := range g.hosts {
			if len(h.Alias) > maxAlias {
				maxAlias = len(h.Alias)
			}
			if len(h.HostName) > maxHost {
				maxHost = len(h.HostName)
			}
		}
	}

	pinned := make([]string, len(cfg.PinnedAliases))
	copy(pinned, cfg.PinnedAliases)

	level := 1
	viewMode := 0
	if startPinned {
		viewMode = 1
	}

	return treeModel{
		title:         title,
		groups:        groups,
		level:         level,
		maxAlias:      maxAlias,
		maxHost:       maxHost,
		pinnedAliases: pinned,
		cfg:           cfg,
		viewMode:      viewMode,
	}
}
