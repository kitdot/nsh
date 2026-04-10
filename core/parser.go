package core

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse parses SSH config content into NSHConfig
func Parse(content string) *NSHConfig {
	lines := splitLines(content)
	var blocks []NSHBlock
	var groupOrder []string
	var pinnedAliases []string
	index := 0

	for index < len(lines) {
		line := lines[index]
		trimmed := strings.TrimSpace(line)

		// Match block
		if isMatchLine(trimmed) {
			raw := collectMatchBlock(lines, &index)
			blocks = append(blocks, NSHBlock{Type: BlockMatch, Raw: raw})
			continue
		}

		// Include directive
		if isIncludeLine(trimmed) {
			blocks = append(blocks, NSHBlock{Type: BlockInclude, Raw: line})
			index++
			continue
		}

		// nsh-groups line
		if isNshGroupsLine(trimmed) {
			groupOrder = parseGroupOrder(trimmed)
			blocks = append(blocks, NSHBlock{Type: BlockComment, Raw: line})
			index++
			continue
		}

		// nsh-pinned line
		if isNshPinnedLine(trimmed) {
			pinnedAliases = parsePinnedAliases(trimmed)
			blocks = append(blocks, NSHBlock{Type: BlockComment, Raw: line})
			index++
			continue
		}

		// nsh tag line, followed by Host
		if isNshLine(trimmed) {
			block, ok := tryParseHostWithNsh(lines, &index)
			if ok {
				blocks = append(blocks, block)
				continue
			}
			blocks = append(blocks, NSHBlock{Type: BlockComment, Raw: line})
			index++
			continue
		}

		// Host line (no tag)
		if isHostLine(trimmed) {
			block := parseHostBlock(lines, &index, "")
			blocks = append(blocks, block)
			continue
		}

		// Blank line
		if trimmed == "" {
			blocks = append(blocks, NSHBlock{Type: BlockBlank, Raw: line})
			index++
			continue
		}

		// Comment or other line
		blocks = append(blocks, NSHBlock{Type: BlockComment, Raw: line})
		index++
	}

	return &NSHConfig{Blocks: blocks, GroupOrder: groupOrder, PinnedAliases: pinnedAliases}
}

// Serialize converts NSHConfig back to SSH config text
func Serialize(config *NSHConfig) string {
	var parts []string
	for _, block := range config.Blocks {
		switch block.Type {
		case BlockHost:
			if block.NshLine != "" {
				parts = append(parts, block.NshLine)
			}
			parts = append(parts, block.HostLine)
			if block.Host != nil {
				parts = append(parts, block.Host.RawPropertyLines...)
			}
		case BlockMatch, BlockInclude, BlockComment, BlockBlank:
			parts = append(parts, block.Raw)
		}
	}
	return strings.Join(parts, "\n")
}

// BuildNshLine constructs a "# nsh: ..." line from host metadata
func BuildNshLine(host *NSHHost) string {
	var parts []string
	if host.Group != "" && host.Group != "Uncategorized" {
		parts = append(parts, "group="+host.Group)
	}
	if host.Desc != "" {
		parts = append(parts, "desc="+host.Desc)
	}
	if host.Auth != "" {
		parts = append(parts, "auth="+host.Auth)
	}
	if host.Order > 0 {
		parts = append(parts, fmt.Sprintf("order=%d", host.Order))
	}
	if len(parts) == 0 {
		return ""
	}
	return "# nsh: " + strings.Join(parts, ", ")
}

// BuildGroupOrderLine constructs a "# nsh-groups: ..." line
func BuildGroupOrderLine(groups []string) string {
	var filtered []string
	for _, g := range groups {
		if g != "" && g != "Uncategorized" {
			filtered = append(filtered, g)
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	return "# nsh-groups: " + strings.Join(filtered, ", ")
}

// BuildPinnedLine constructs a "# nsh-pinned: ..." line
func BuildPinnedLine(aliases []string) string {
	if len(aliases) == 0 {
		return ""
	}
	return "# nsh-pinned: " + strings.Join(aliases, ", ")
}

// --- Private helpers ---

func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	var lines []string
	var current strings.Builder
	for _, ch := range content {
		if ch == '\n' {
			lines = append(lines, current.String())
			current.Reset()
		} else {
			current.WriteRune(ch)
		}
	}
	lines = append(lines, current.String())
	return lines
}

func isNshLine(trimmed string) bool {
	return len(trimmed) >= 6 && strings.HasPrefix(strings.ToLower(trimmed), "# nsh:")
}

func isNshGroupsLine(trimmed string) bool {
	return len(trimmed) >= 13 && strings.HasPrefix(strings.ToLower(trimmed), "# nsh-groups:")
}

func isNshPinnedLine(trimmed string) bool {
	return len(trimmed) >= 13 && strings.HasPrefix(strings.ToLower(trimmed), "# nsh-pinned:")
}

func parsePinnedAliases(trimmed string) []string {
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "# nsh-pinned:") {
		return nil
	}
	value := strings.TrimSpace(trimmed[len("# nsh-pinned:"):])
	parts := strings.Split(value, ",")
	var result []string
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func isHostLine(trimmed string) bool {
	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "host ") || lower == "host"
}

func isMatchLine(trimmed string) bool {
	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "match ") || lower == "match"
}

func isIncludeLine(trimmed string) bool {
	return strings.HasPrefix(strings.ToLower(trimmed), "include ")
}

func parseGroupOrder(trimmed string) []string {
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "# nsh-groups:") {
		return nil
	}
	value := strings.TrimSpace(trimmed[len("# nsh-groups:"):])
	parts := strings.Split(value, ",")
	var result []string
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

type nshMeta struct {
	group string
	desc  string
	auth  string
	order int
}

func parseNshMeta(trimmed string) nshMeta {
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "# nsh:") {
		return nshMeta{group: "Uncategorized"}
	}
	value := strings.TrimSpace(trimmed[len("# nsh:"):])

	m := nshMeta{group: "Uncategorized"}
	for _, part := range strings.Split(value, ",") {
		kv := strings.TrimSpace(part)
		eqIdx := strings.Index(kv, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(kv[:eqIdx]))
		val := strings.TrimSpace(kv[eqIdx+1:])
		switch key {
		case "group":
			m.group = val
		case "desc":
			m.desc = val
		case "auth":
			m.auth = val
		case "order":
			if n, err := strconv.Atoi(val); err == nil {
				m.order = n
			}
		}
	}
	return m
}

func tryParseHostWithNsh(lines []string, index *int) (NSHBlock, bool) {
	startIndex := *index
	nshLine := lines[*index]
	cursor := *index + 1

	if cursor < len(lines) {
		nextTrimmed := strings.TrimSpace(lines[cursor])
		if isHostLine(nextTrimmed) {
			*index = cursor
			block := parseHostBlock(lines, index, nshLine)
			return block, true
		}
	}

	*index = startIndex
	return NSHBlock{}, false
}

func parseHostBlock(lines []string, index *int, nshLine string) NSHBlock {
	hostLine := lines[*index]
	hostTrimmed := strings.TrimSpace(hostLine)

	alias := extractHostAlias(hostTrimmed)
	isWildcard := alias == "*"

	var meta nshMeta
	if nshLine != "" {
		meta = parseNshMeta(strings.TrimSpace(nshLine))
	} else {
		meta = nshMeta{group: "Uncategorized"}
	}

	*index++

	var rawPropertyLines []string
	var hostName, user, port, identityFile string

	for *index < len(lines) {
		line := lines[*index]
		trimmed := strings.TrimSpace(line)
		isIndented := len(line) > 0 && (line[0] == ' ' || line[0] == '\t')

		if !isIndented && (isHostLine(trimmed) || isMatchLine(trimmed) || isNshLine(trimmed) || isNshGroupsLine(trimmed) || isNshPinnedLine(trimmed) || isIncludeLine(trimmed)) {
			break
		}

		if !isIndented && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			break
		}

		rawPropertyLines = append(rawPropertyLines, line)

		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "hostname ") || strings.HasPrefix(lower, "hostname\t") {
			hostName = extractPropertyValue(trimmed)
		} else if strings.HasPrefix(lower, "user ") || strings.HasPrefix(lower, "user\t") {
			user = extractPropertyValue(trimmed)
		} else if strings.HasPrefix(lower, "port ") || strings.HasPrefix(lower, "port\t") {
			port = extractPropertyValue(trimmed)
		} else if strings.HasPrefix(lower, "identityfile ") || strings.HasPrefix(lower, "identityfile\t") {
			identityFile = extractPropertyValue(trimmed)
		}

		*index++
	}

	host := &NSHHost{
		Group:            meta.group,
		Desc:             meta.desc,
		Auth:             meta.auth,
		Order:            meta.order,
		Alias:            alias,
		HostName:         hostName,
		User:             user,
		Port:             port,
		IdentityFile:     identityFile,
		IsWildcard:       isWildcard,
		RawPropertyLines: rawPropertyLines,
	}

	return NSHBlock{
		Type:     BlockHost,
		Host:     host,
		NshLine:  nshLine,
		HostLine: hostLine,
	}
}

func collectMatchBlock(lines []string, index *int) string {
	var matchLines []string
	matchLines = append(matchLines, lines[*index])
	*index++

	for *index < len(lines) {
		line := lines[*index]
		trimmed := strings.TrimSpace(line)
		isIndented := len(line) > 0 && (line[0] == ' ' || line[0] == '\t')

		if !isIndented && (isHostLine(trimmed) || isMatchLine(trimmed) || isNshLine(trimmed) || isNshGroupsLine(trimmed) || isNshPinnedLine(trimmed) || isIncludeLine(trimmed)) {
			break
		}

		if !isIndented && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			break
		}

		matchLines = append(matchLines, line)
		*index++
	}

	return strings.Join(matchLines, "\n")
}

func extractHostAlias(trimmed string) string {
	parts := strings.SplitN(trimmed, " ", 2)
	if len(parts) < 2 {
		parts = strings.SplitN(trimmed, "\t", 2)
	}
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func extractPropertyValue(trimmed string) string {
	// Split on first space or tab
	for i, ch := range trimmed {
		if ch == ' ' || ch == '\t' {
			return strings.TrimSpace(trimmed[i+1:])
		}
	}
	return ""
}
