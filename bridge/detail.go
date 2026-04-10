package bridge

import (
	"fmt"
	"strings"

	"github.com/kitdot/nsh/core"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("252"))

	detailKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	detailValStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	detailLineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	detailFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))
)

type detailModel struct {
	host *core.NSHHost
	done bool
}

// ShowDetail displays host details in a clean alt-screen view.
// Blocks until user presses any key.
func ShowDetail(host *core.NSHHost) {
	m := detailModel{host: host}
	p := tea.NewProgram(m, tea.WithAltScreen())
	p.Run()
}

func (m detailModel) Init() tea.Cmd {
	return nil
}

func (m detailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m detailModel) View() string {
	if m.done {
		return ""
	}

	h := m.host
	var b strings.Builder

	line := detailLineStyle.Render("  ─────────────────────────────")

	b.WriteString("\n")
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("  Host: %s", h.Alias)))
	b.WriteString("\n")
	b.WriteString(line)
	b.WriteString("\n\n")

	addField := func(key, val string) {
		if val != "" {
			b.WriteString(fmt.Sprintf("  %s %s\n",
				detailKeyStyle.Render(pad(key+":", 16)),
				detailValStyle.Render(val),
			))
		}
	}

	addField("HostName", h.HostName)
	addField("User", h.User)
	if h.Port != "" {
		addField("Port", h.Port)
	}
	addField("IdentityFile", h.IdentityFile)
	if h.Auth != "" {
		addField("Auth", h.Auth)
	} else {
		addField("Auth", "none")
	}

	group := h.Group
	if group == "Uncategorized" {
		group = "-"
	}
	addField("Group", group)

	if h.Order > 0 {
		addField("Order", fmt.Sprintf("%d", h.Order))
	}
	addField("Description", h.Desc)

	if h.IsWildcard {
		b.WriteString("\n")
		b.WriteString(detailKeyStyle.Render("  [global-default]"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(line)
	b.WriteString("\n\n")
	b.WriteString(detailFooterStyle.Render("  press any key to go back"))

	return b.String()
}
