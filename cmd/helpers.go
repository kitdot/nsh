package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/core"
)

var validAliasRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// isValidAlias checks if an alias only contains safe characters.
func isValidAlias(alias string) bool {
	return validAliasRe.MatchString(alias)
}

// containsComma checks if a string contains a comma (breaks # nsh: tag parsing).
func containsComma(s string) bool {
	return strings.Contains(s, ",")
}

// nextAvailableAlias returns base + "-N" where N is the lowest positive
// integer that doesn't collide with anything in `existing`. If `base` already
// ends in "-<digits>" the suffix is stripped first so `host-1` duplicates to
// `host-2` instead of `host-1-1`.
func nextAvailableAlias(base string, existing []string) string {
	root := stripTrailingNumberSuffix(base)
	taken := make(map[string]struct{}, len(existing))
	for _, a := range existing {
		taken[a] = struct{}{}
	}
	for n := 1; ; n++ {
		candidate := fmt.Sprintf("%s-%d", root, n)
		if _, hit := taken[candidate]; !hit {
			return candidate
		}
	}
}

// stripTrailingNumberSuffix strips a trailing "-<digits>" suffix if present.
func stripTrailingNumberSuffix(s string) string {
	idx := strings.LastIndex(s, "-")
	if idx < 0 || idx == len(s)-1 {
		return s
	}
	tail := s[idx+1:]
	for _, r := range tail {
		if r < '0' || r > '9' {
			return s
		}
	}
	return s[:idx]
}

const (
	iconDefault = "○"
	iconKey     = "◆"
	iconPass    = "●"

	cYellow = "\033[33m"
	cGreen  = "\033[32m"
	cDim    = "\033[2m"
	cReset  = "\033[0m"
)

// buildGroupedHostOptions builds a picker option list grouped by group
// with auth icons and aligned columns like the TUI selector
func buildGroupedHostOptions(cfg *core.NSHConfig) []bridge.PickerOption {
	// Collect all non-wildcard hosts to compute max alias width
	var allHosts []core.NSHHost
	for _, h := range cfg.Hosts() {
		if !h.IsWildcard {
			allHosts = append(allHosts, h)
		}
	}
	if len(allHosts) == 0 {
		return nil
	}

	maxAlias := 0
	maxHost := 0
	for _, h := range allHosts {
		if len(h.Alias) > maxAlias {
			maxAlias = len(h.Alias)
		}
		if len(h.HostName) > maxHost {
			maxHost = len(h.HostName)
		}
	}

	var options []bridge.PickerOption
	groups := cfg.SortedGroups()

	for _, group := range groups {
		hosts := cfg.HostsInGroup(group)
		var filtered []core.NSHHost
		for _, h := range hosts {
			if !h.IsWildcard {
				filtered = append(filtered, h)
			}
		}
		if len(filtered) == 0 {
			continue
		}

		// Group header — Uncategorized uses empty label (renders as just "- ")
		groupLabel := group
		if group == "Uncategorized" {
			groupLabel = ""
		}
		options = append(options, bridge.PickerOption{
			Label:       groupLabel,
			IsSeparator: true,
		})

		for _, h := range filtered {
			// Auth icon with color
			var icon string
			switch h.Auth {
			case "password":
				icon = cYellow + iconPass + cReset
			case "key":
				icon = cGreen + iconKey + cReset
			default:
				icon = cDim + iconDefault + cReset
			}

			// Padded alias and hostname
			paddedAlias := h.Alias + strings.Repeat(" ", maxAlias-len(h.Alias))
			paddedHost := h.HostName + strings.Repeat(" ", maxHost-len(h.HostName))

			label := fmt.Sprintf("%s %s    %s%s%s", icon, paddedAlias, cDim, paddedHost, cReset)

			options = append(options, bridge.PickerOption{
				Value:       h.Alias,
				Label:       label,
				Description: h.Desc,
			})
		}
	}

	return options
}
