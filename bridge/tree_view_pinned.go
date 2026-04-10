package bridge

import (
	"strings"

	"github.com/kitdot/nsh/core"

	"github.com/charmbracelet/lipgloss"
)

func (m treeModel) renderPinnedView() string {
	var lines []string

	groupsTab := treeDimStyle.Render(" Groups ")
	pinnedTab := treePinStyle.Render("[Pinned]")
	lines = append(lines, "  "+groupsTab+"  "+pinnedTab)

	lines = append(lines, treeTitleStyle.Render("  ★ Pinned"))

	if m.filtering {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("  / "+m.filter)+"█")
	}

	lines = append(lines, "")

	fph := m.filteredPinnedHosts()
	if len(fph) == 0 {
		if m.filter != "" {
			lines = append(lines, treeDimStyle.Render("  No matches"))
		} else {
			lines = append(lines, treeDimStyle.Render("  No pinned hosts. Press tab to switch back."))
		}
	} else {
		for i, host := range fph {
			isCursor := i == m.pinnedCursor
			if m.pinnedMoving && isCursor {
				lines = append(lines, m.renderMovingHostLine(host))
			} else {
				lines = append(lines, m.renderHostLine(host, isCursor))
			}
		}
	}

	listContent := strings.Join(lines, "\n")
	footerRendered := m.renderPinnedFooter(fph)

	var curHost *core.NSHHost
	if len(fph) > 0 && m.pinnedCursor < len(fph) {
		curHost = &fph[m.pinnedCursor]
	}

	return m.renderTreeBody(listContent, curHost, footerRendered)
}

func (m treeModel) renderPinnedFooter(fph []core.NSHHost) string {
	var footer string
	switch {
	case m.pinnedConfirm:
		footer = m.renderPinnedDeleteConfirmBar(fph) + "\n" + treeFooterStyle.Render("  tab switch • enter confirm • esc back")
	case m.pinnedSaveConf:
		footer = m.renderPinnedSaveConfirmBar() + "\n" + treeFooterStyle.Render("  tab switch • enter confirm • esc cancel")
	case m.pinnedMoving:
		footer = "\n" + treeFooterStyle.Render("  ↑↓ move item • space/enter save • esc cancel")
	case m.filtering:
		footer = "\n" + treeFooterStyle.Render("  type to filter • ↑↓ navigate • enter connect • esc clear")
	default:
		footer = "\n" + treeFooterStyle.Render("  ↑↓ navigate • enter connect • / filter • esc quit\n"+
			"  e edit • d delete • p unpin • space reorder")
	}
	return footer
}

func (m treeModel) renderPinnedDeleteConfirmBar(fph []core.NSHHost) string {
	hostLabel := ""
	if m.pinnedCursor < len(fph) {
		hostLabel = fph[m.pinnedCursor].Alias
	}
	noLabel := treeConfirmNoStyle.Render("[No]")
	yesLabel := treeDimStyle.Render(" Yes, delete ")
	if m.pinnedConfIdx == 1 {
		noLabel = treeDimStyle.Render(" No ")
		yesLabel = treeConfirmYesStyle.Render("[Yes, delete]")
	}
	return "\n  " + treeConfirmYesStyle.Render("Delete "+hostLabel+"? ") + " " + noLabel + "  " + yesLabel
}

func (m treeModel) renderPinnedSaveConfirmBar() string {
	cancelLabel := treeConfirmNoStyle.Render("[Cancel]")
	saveLabel := treeDimStyle.Render(" Save ")
	if m.pinnedSaveIdx == 1 {
		cancelLabel = treeDimStyle.Render(" Cancel ")
		saveLabel = treeConfirmNoStyle.Render("[Save]")
	}
	return "\n  " + treePanelTitleStyle.Render("Save new pinned order?") + " " + cancelLabel + "  " + saveLabel
}

func (m treeModel) renderTreeBody(listContent string, curHost *core.NSHHost, footerRendered string) string {
	var body string
	if m.width > 70 && curHost != nil {
		panelWidth := 35
		if panelWidth > m.width/3 {
			panelWidth = m.width / 3
		}
		listWidth := m.width - panelWidth - 4

		previewContent := m.renderPreviewPanel(curHost, panelWidth-4)
		previewRendered := treePanelStyle.Width(panelWidth).Render(previewContent)

		body = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(listWidth).Render(listContent),
			previewRendered,
		)
	} else {
		body = listContent
	}

	return m.pinFooterToBottom(body, footerRendered)
}
