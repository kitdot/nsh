package bridge

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m treeModel) renderGroupsView() string {
	var treeLines []string

	groupsTab := treePinStyle.Render("[Groups]")
	pinnedTab := treeDimStyle.Render(" Pinned ")
	treeLines = append(treeLines, "  "+groupsTab+"  "+pinnedTab)

	treeLines = append(treeLines, treeTitleStyle.Render("  "+m.title))

	if m.filtering {
		treeLines = append(treeLines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("  / "+m.filter)+"█")
	}

	treeLines = append(treeLines, "")

	vg := m.visibleGroups()
	if len(vg) == 0 {
		treeLines = append(treeLines, treeDimStyle.Render("  No matches"))
	}

	for vi, gi := range vg {
		g := m.groups[gi]
		if vi > 0 {
			treeLines = append(treeLines, "")
		}

		isGroupCursor := m.level == 0 && vi == m.groupIdx
		if g.expanded {
			treeLines = append(treeLines, treeGroupStyle.Render("▼ "+g.label))

			hosts := m.visibleHosts(gi)
			for hi, host := range hosts {
				isHostCursor := m.level >= 1 && vi == m.groupIdx && hi == m.hostIdx
				treeLines = append(treeLines, m.renderHostLine(host, isHostCursor))
			}
			continue
		}

		cursor := "  "
		if isGroupCursor {
			cursor = "> "
		}
		countStr := treeDimStyle.Render(fmt.Sprintf("(%d hosts)", len(m.visibleHosts(gi))))

		if isGroupCursor {
			treeLines = append(treeLines, fmt.Sprintf("%s%s  %s",
				cursor,
				treeGroupActiveStyle.Render("- "+g.label),
				countStr,
			))
		} else {
			treeLines = append(treeLines, fmt.Sprintf("%s%s  %s",
				cursor,
				treeGroupStyle.Render("- "+g.label),
				countStr,
			))
		}
	}

	treeContent := strings.Join(treeLines, "\n")
	footerRendered := "\n" + m.renderGroupsConfirmBar() + treeFooterStyle.Render(m.groupsFooter())
	return m.renderTreeBody(treeContent, m.getCurrentHost(), footerRendered)
}

func (m treeModel) renderGroupsConfirmBar() string {
	if m.level != 2 {
		return ""
	}

	curHost := m.getCurrentHost()
	hostLabel := ""
	if curHost != nil {
		hostLabel = curHost.Alias
	}

	noLabel := treeConfirmNoStyle.Render("[No]")
	yesLabel := treeDimStyle.Render(" Yes, delete ")
	if m.confirmIdx == 1 {
		noLabel = treeDimStyle.Render(" No ")
		yesLabel = treeConfirmYesStyle.Render("[Yes, delete]")
	}

	return "\n  " + treeConfirmYesStyle.Render("Delete "+hostLabel+"? ") + " " + noLabel + "  " + yesLabel + "\n"
}

func (m treeModel) groupsFooter() string {
	if m.level == 2 {
		return "  tab switch • enter confirm • esc back"
	}
	if m.filtering {
		return "  type to filter • ↑↓ navigate • enter connect • esc clear"
	}
	if m.level == 1 {
		return "  ↑↓ navigate • enter connect • ← collapse • / filter • esc back\n" +
			"  n new • e edit • d delete • p pin"
	}
	return "  ↑↓ navigate • enter/→ expand • / filter • esc quit\n" +
		"  n new"
}
