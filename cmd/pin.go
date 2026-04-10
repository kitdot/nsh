package cmd

import (
	"strings"

	"github.com/kitdot/nsh/core"
)

// updatePinnedLine inserts or updates the "# nsh-pinned: ..." line in blocks
func updatePinnedLine(blocks []core.NSHBlock, pinnedAliases []string) []core.NSHBlock {
	newBlocks := make([]core.NSHBlock, len(blocks))
	copy(newBlocks, blocks)

	pinnedLine := core.BuildPinnedLine(pinnedAliases)

	// Try to find and update existing nsh-pinned line
	for i, b := range newBlocks {
		if b.Type == core.BlockComment && strings.HasPrefix(strings.ToLower(strings.TrimSpace(b.Raw)), "# nsh-pinned:") {
			if pinnedLine != "" {
				newBlocks[i] = core.NSHBlock{Type: core.BlockComment, Raw: pinnedLine}
			} else {
				newBlocks = append(newBlocks[:i], newBlocks[i+1:]...)
			}
			return newBlocks
		}
	}

	// Not found, insert after nsh-groups line (or at top)
	if pinnedLine != "" {
		insertIndex := 0
		for i, b := range newBlocks {
			if b.Type == core.BlockComment && strings.HasPrefix(strings.ToLower(strings.TrimSpace(b.Raw)), "# nsh-groups:") {
				insertIndex = i + 1
				if insertIndex < len(newBlocks) && newBlocks[insertIndex].Type == core.BlockBlank {
					insertIndex++
				}
				break
			}
		}
		if insertIndex == 0 {
			for insertIndex < len(newBlocks) {
				t := newBlocks[insertIndex].Type
				if t == core.BlockComment || t == core.BlockInclude || t == core.BlockBlank {
					insertIndex++
				} else {
					break
				}
			}
		}

		tail := make([]core.NSHBlock, len(newBlocks[insertIndex:]))
		copy(tail, newBlocks[insertIndex:])
		newBlocks = append(newBlocks[:insertIndex],
			core.NSHBlock{Type: core.BlockComment, Raw: pinnedLine},
			core.NSHBlock{Type: core.BlockBlank, Raw: ""},
		)
		newBlocks = append(newBlocks, tail...)
	}

	return newBlocks
}
