package bridge

import (
	"fmt"
	"strings"

	"github.com/kitdot/nsh/core"

	"github.com/charmbracelet/lipgloss"
)

func (m treeModel) View() string {
	if m.quitting {
		return ""
	}

	if m.viewMode == 1 {
		return m.renderPinnedView()
	}
	return m.renderGroupsView()
}

func (m treeModel) renderPreviewPanel(host *core.NSHHost, width int) string {
	var b strings.Builder

	b.WriteString(treePanelTitleStyle.Render(host.Alias))
	b.WriteString("\n\n")

	addField := func(key, val string) {
		if val != "" {
			b.WriteString(treePanelKeyStyle.Render(pad(key+":", 15)))
			b.WriteString(treePanelValStyle.Render(val))
			b.WriteString("\n")
		}
	}

	addField("HostName", host.HostName)
	addField("User", host.User)
	if host.Port != "" && host.Port != "22" {
		addField("Port", host.Port)
	}
	addField("IdentityFile", host.IdentityFile)
	if host.Auth != "" {
		addField("Auth", host.Auth)
	}
	group := host.Group
	if group == "Uncategorized" {
		group = "-"
	}
	addField("Group", group)
	if host.Order > 0 {
		addField("Order", fmt.Sprintf("%d", host.Order))
	}
	addField("Description", host.Desc)

	return b.String()
}

func (m treeModel) renderHostLine(host core.NSHHost, isCursor bool) string {
	cursor := "  "
	if isCursor {
		cursor = "> "
	}

	pin := "  "
	if m.isPinned(host.Alias) {
		pin = treePinStyle.Render("⦿ ")
	}

	var icon string
	switch host.Auth {
	case "password":
		icon = treeAuthPassStyle.Render(treeIconPass)
	case "key":
		icon = treeAuthKeyStyle.Render(treeIconKey)
	default:
		icon = treeAuthNoneStyle.Render(treeIconDefault)
	}

	paddedAlias := host.Alias + strings.Repeat(" ", m.maxAlias-len(host.Alias))
	paddedHost := host.HostName + strings.Repeat(" ", m.maxHost-len(host.HostName))

	if isCursor {
		return fmt.Sprintf("  %s%s%s %s    %s",
			cursor,
			pin,
			icon,
			treeHostActiveStyle.Render(paddedAlias),
			treeHostNameStyle.Render(paddedHost),
		)
	}
	return fmt.Sprintf("  %s%s%s %s    %s",
		cursor,
		pin,
		icon,
		treeHostNormalStyle.Render(paddedAlias),
		treeHostNameStyle.Render(paddedHost),
	)
}

func (m treeModel) renderMovingHostLine(host core.NSHHost) string {
	pin := "  "
	if m.isPinned(host.Alias) {
		pin = treePinStyle.Render("⦿ ")
	}

	var icon string
	switch host.Auth {
	case "password":
		icon = treeAuthPassStyle.Render(treeIconPass)
	case "key":
		icon = treeAuthKeyStyle.Render(treeIconKey)
	default:
		icon = treeAuthNoneStyle.Render(treeIconDefault)
	}

	paddedAlias := host.Alias + strings.Repeat(" ", m.maxAlias-len(host.Alias))
	paddedHost := host.HostName + strings.Repeat(" ", m.maxHost-len(host.HostName))

	movingStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Background(lipgloss.Color("236"))

	return fmt.Sprintf("  ≡ %s%s %s    %s",
		pin,
		icon,
		movingStyle.Render(paddedAlias),
		treeHostNameStyle.Render(paddedHost),
	)
}

func (m treeModel) pinFooterToBottom(body, footer string) string {
	footerHeight := lipgloss.Height(footer)
	maxBody := m.height - footerHeight
	if maxBody < 3 {
		maxBody = 3
	}

	bodyLines := strings.Split(body, "\n")

	if len(bodyLines) <= maxBody {
		gap := maxBody - len(bodyLines)
		return body + strings.Repeat("\n", gap) + footer
	}

	cursorLine := 0
	for i, line := range bodyLines {
		stripped := stripAnsi(line)
		if strings.Contains(stripped, "> ") {
			cursorLine = i
			break
		}
	}

	margin := maxBody / 4
	if margin < 2 {
		margin = 2
	}
	offset := cursorLine - margin
	if offset > len(bodyLines)-maxBody {
		offset = len(bodyLines) - maxBody
	}
	if offset < 0 {
		offset = 0
	}

	end := offset + maxBody
	if end > len(bodyLines) {
		end = len(bodyLines)
	}
	visible := bodyLines[offset:end]

	return strings.Join(visible, "\n") + footer
}
