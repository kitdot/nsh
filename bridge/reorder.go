package bridge

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ReorderItem represents an item that can be reordered
type ReorderItem struct {
	Key         string
	Label       string
	Description string
}

type reorderModel struct {
	title    string
	items    []ReorderItem
	cursor   int
	moving   bool // whether the item at cursor is being moved
	result   []ReorderItem
	quitting bool
}

var (
	reorderTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	reorderNormalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	reorderCursorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("212"))

	reorderMovingStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("220")).
				Background(lipgloss.Color("236"))

	reorderDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	reorderNumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Width(4)

	reorderFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				MarginTop(1)
)

// Reorder launches an interactive reorder TUI.
// Returns the reordered items, or nil if cancelled.
func Reorder(title string, items []ReorderItem) []ReorderItem {
	if len(items) == 0 {
		return nil
	}

	m := reorderModel{
		title: title,
		items: make([]ReorderItem, len(items)),
	}
	copy(m.items, items)

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil
	}

	fm := finalModel.(reorderModel)
	return fm.result
}

func (m reorderModel) Init() tea.Cmd {
	return nil
}

func (m reorderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				if m.moving {
					// Swap with above
					m.items[m.cursor], m.items[m.cursor-1] = m.items[m.cursor-1], m.items[m.cursor]
				}
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				if m.moving {
					// Swap with below
					m.items[m.cursor], m.items[m.cursor+1] = m.items[m.cursor+1], m.items[m.cursor]
				}
				m.cursor++
			}

		case " ":
			// Toggle moving mode
			m.moving = !m.moving

		case "enter":
			m.result = make([]ReorderItem, len(m.items))
			copy(m.result, m.items)
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m reorderModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(reorderTitleStyle.Render("  " + m.title))
	b.WriteString("\n\n")

	for i, item := range m.items {
		num := reorderNumStyle.Render(fmt.Sprintf("%d.", i+1))

		label := item.Label
		desc := ""
		if item.Description != "" {
			desc = reorderDimStyle.Render(" " + item.Description)
		}

		if i == m.cursor {
			if m.moving {
				// Moving state — highlight with grab indicator
				line := fmt.Sprintf("  %s ≡ %s%s", num, reorderMovingStyle.Render(label), desc)
				b.WriteString(line)
			} else {
				// Cursor state
				line := fmt.Sprintf("  %s > %s%s", num, reorderCursorStyle.Render(label), desc)
				b.WriteString(line)
			}
		} else {
			line := fmt.Sprintf("  %s   %s%s", num, reorderNormalStyle.Render(label), desc)
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.moving {
		b.WriteString(reorderFooterStyle.Render("  ↑↓/jk move item • space release • enter save • esc cancel"))
	} else {
		b.WriteString(reorderFooterStyle.Render("  ↑↓/jk navigate • space grab • enter save • esc cancel"))
	}

	return b.String()
}
