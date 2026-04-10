package bridge

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PickerOption represents a selectable option
type PickerOption struct {
	Value       string
	Label       string
	Description string
	IsCurrent   bool
	IsSeparator bool // renders as a divider line, not selectable
}

// pickerModel is a generic Bubble Tea list picker with filter support
type pickerModel struct {
	title      string
	allOptions []PickerOption
	filtered   []int // indices into allOptions
	cursor     int   // index into filtered
	selected   string
	filter     string
	filtering  bool
	quitting   bool
}

var (
	pickerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	pickerActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("212"))

	pickerNormalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	pickerDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	pickerCurrentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("34"))

	pickerFilterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	pickerFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				MarginTop(1)
)

// Pick launches a Bubble Tea interactive picker, returns the selected Value or ""
func Pick(title string, options []PickerOption) string {
	// Build initial filtered list (all items)
	var filtered []int
	for i := range options {
		filtered = append(filtered, i)
	}

	// Default cursor to current item
	cursor := 0
	for fi, oi := range filtered {
		if !options[oi].IsSeparator && options[oi].IsCurrent {
			cursor = fi
			break
		}
	}
	// If no current found, find first non-separator
	if cursor == 0 && (len(filtered) == 0 || options[filtered[0]].IsSeparator) {
		for fi, oi := range filtered {
			if !options[oi].IsSeparator {
				cursor = fi
				break
			}
		}
	}

	m := pickerModel{
		title:      title,
		allOptions: options,
		filtered:   filtered,
		cursor:     cursor,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return ""
	}

	fm := finalModel.(pickerModel)
	return fm.selected
}

func (m pickerModel) Init() tea.Cmd {
	return nil
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case msg.String() == "esc":
			if m.filtering {
				m.filtering = false
				m.filter = ""
				m.applyFilter()
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case msg.String() == "/":
			if !m.filtering {
				m.filtering = true
				m.filter = ""
				return m, nil
			}
			// In filter mode, '/' is just a character
			m.filter += "/"
			m.applyFilter()
			return m, nil

		case msg.String() == "backspace":
			if m.filtering && len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.applyFilter()
			}
			return m, nil

		case msg.String() == "up" || msg.String() == "k":
			if !m.filtering || msg.String() == "up" {
				m.moveCursorUp()
			} else {
				m.filter += msg.String()
				m.applyFilter()
			}
			return m, nil

		case msg.String() == "down" || msg.String() == "j":
			if !m.filtering || msg.String() == "down" {
				m.moveCursorDown()
			} else {
				m.filter += msg.String()
				m.applyFilter()
			}
			return m, nil

		case msg.String() == "enter":
			if m.filtering {
				m.filtering = false
				return m, nil
			}
			if m.cursor >= 0 && m.cursor < len(m.filtered) {
				opt := m.allOptions[m.filtered[m.cursor]]
				if !opt.IsSeparator {
					m.selected = opt.Value
					return m, tea.Quit
				}
			}
			return m, nil

		default:
			if m.filtering {
				m.filter += msg.String()
				m.applyFilter()
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *pickerModel) moveCursorUp() {
	for i := m.cursor - 1; i >= 0; i-- {
		if !m.allOptions[m.filtered[i]].IsSeparator {
			m.cursor = i
			return
		}
	}
}

func (m *pickerModel) moveCursorDown() {
	for i := m.cursor + 1; i < len(m.filtered); i++ {
		if !m.allOptions[m.filtered[i]].IsSeparator {
			m.cursor = i
			return
		}
	}
}

func (m *pickerModel) applyFilter() {
	if m.filter == "" {
		m.filtered = nil
		for i := range m.allOptions {
			m.filtered = append(m.filtered, i)
		}
		m.cursor = 0
		for fi, oi := range m.filtered {
			if !m.allOptions[oi].IsSeparator {
				m.cursor = fi
				break
			}
		}
		return
	}

	m.filtered = nil
	for i, opt := range m.allOptions {
		if opt.IsSeparator {
			continue
		}
		combined := strings.ToLower(opt.Label + " " + opt.Description + " " + opt.Value)
		if matchAllTerms(m.filter, combined) {
			m.filtered = append(m.filtered, i)
		}
	}
	m.cursor = 0
}

func (m pickerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(pickerTitleStyle.Render("  " + m.title))
	b.WriteString("\n")

	// Filter bar
	if m.filtering {
		b.WriteString(pickerFilterStyle.Render("  / "+m.filter) + "█")
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if len(m.filtered) == 0 {
		b.WriteString(pickerDescStyle.Render("  No matches"))
		b.WriteString("\n")
	}

	// Check if we have group headers (labeled separators) for indentation
	hasGroups := false
	for _, opt := range m.allOptions {
		if opt.IsSeparator && opt.Label != "" {
			hasGroups = true
			break
		}
	}
	indent := ""
	if hasGroups {
		indent = "  "
	}

	pickerGroupStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))

	prevWasSeparator := false
	for fi, oi := range m.filtered {
		opt := m.allOptions[oi]

		if opt.IsSeparator {
			// Add blank line between groups (not before first)
			if fi > 0 {
				b.WriteString("\n")
			}
			if opt.Label != "" {
				b.WriteString(pickerGroupStyle.Render("  - " + opt.Label))
			} else if opt.Value == "__divider__" {
				b.WriteString(pickerDescStyle.Render("  ──────────────"))
			} else {
				b.WriteString(pickerGroupStyle.Render("  -"))
			}
			b.WriteString("\n")
			prevWasSeparator = true
			continue
		}
		_ = prevWasSeparator
		prevWasSeparator = false

		cursor := "  "
		if fi == m.cursor {
			cursor = "> "
		}

		current := ""
		if opt.IsCurrent {
			current = pickerCurrentStyle.Render(" (current)")
		}

		label := opt.Label
		if label == "" {
			label = opt.Value
		}

		desc := ""
		if opt.Description != "" {
			desc = pickerDescStyle.Render(" — " + opt.Description)
		}

		if fi == m.cursor {
			line := fmt.Sprintf("  %s%s%s%s%s", indent, cursor, pickerActiveStyle.Render(label), current, desc)
			b.WriteString(line)
		} else {
			line := fmt.Sprintf("  %s%s%s%s%s", indent, cursor, pickerNormalStyle.Render(label), current, desc)
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.filtering {
		b.WriteString(pickerFooterStyle.Render("  type to filter • ↑↓ navigate • enter confirm • esc clear"))
	} else {
		footer := "  ↑↓/jk navigate • enter select • / filter • esc cancel"
		if len(m.allOptions) <= 5 {
			footer = "  ↑↓/jk navigate • enter select • esc cancel"
		}
		b.WriteString(pickerFooterStyle.Render(footer))
	}

	return b.String()
}

// matchAllTerms checks if all space-separated terms in filter are found in text
func matchAllTerms(filter, text string) bool {
	terms := strings.Fields(strings.ToLower(filter))
	lower := strings.ToLower(text)
	for _, term := range terms {
		if !strings.Contains(lower, term) {
			return false
		}
	}
	return true
}
