package bridge

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MultiPickerOption represents a selectable option in a multi-select picker
type MultiPickerOption struct {
	Value       string
	Label       string
	Description string
}

// MultiPickResult holds the result of a multi-select picker
type MultiPickResult struct {
	Selected []string // selected values
	All      bool     // true if cancelled or none selected
}

type multiPickerModel struct {
	title    string
	options  []MultiPickerOption
	cursor   int
	checked  map[int]bool
	quitting bool
	done     bool
}

var (
	mpCheckStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)
)

// MultiPick launches a multi-select picker. Returns selected values.
// If user cancels or selects nothing, returns nil.
func MultiPick(title string, options []MultiPickerOption) []string {
	if len(options) == 0 {
		return nil
	}

	m := multiPickerModel{
		title:   title,
		options: options,
		checked: make(map[int]bool),
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil
	}

	fm := finalModel.(multiPickerModel)
	if fm.quitting || !fm.done {
		return nil
	}

	var selected []string
	for i, opt := range fm.options {
		if fm.checked[i] {
			selected = append(selected, opt.Value)
		}
	}
	return selected
}

func (m multiPickerModel) Init() tea.Cmd {
	return nil
}

func (m multiPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}

		case " ":
			m.checked[m.cursor] = !m.checked[m.cursor]

		case "a":
			// Toggle all
			allChecked := len(m.checked) == len(m.options)
			if allChecked {
				// Check if truly all checked
				for i := range m.options {
					if !m.checked[i] {
						allChecked = false
						break
					}
				}
			}
			if allChecked {
				m.checked = make(map[int]bool)
			} else {
				for i := range m.options {
					m.checked[i] = true
				}
			}

		case "enter":
			// Must have at least one selected
			hasSelection := false
			for _, v := range m.checked {
				if v {
					hasSelection = true
					break
				}
			}
			if hasSelection {
				m.done = true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m multiPickerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(pickerTitleStyle.Render("  " + m.title))
	b.WriteString("\n\n")

	selectedCount := 0
	for _, v := range m.checked {
		if v {
			selectedCount++
		}
	}

	for i, opt := range m.options {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		check := "[ ]"
		if m.checked[i] {
			check = mpCheckStyle.Render("[✓]")
		}

		label := opt.Label
		if label == "" {
			label = opt.Value
		}

		desc := ""
		if opt.Description != "" {
			desc = pickerDescStyle.Render(" — " + opt.Description)
		}

		if i == m.cursor {
			line := fmt.Sprintf("  %s%s %s%s", cursor, check, pickerActiveStyle.Render(label), desc)
			b.WriteString(line)
		} else {
			line := fmt.Sprintf("  %s%s %s%s", cursor, check, pickerNormalStyle.Render(label), desc)
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	footer := fmt.Sprintf("  ↑↓/jk navigate • space toggle • a toggle all • enter confirm (%d selected) • esc cancel", selectedCount)
	b.WriteString(pickerFooterStyle.Render(footer))

	return b.String()
}
