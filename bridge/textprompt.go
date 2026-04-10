package bridge

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TextPromptResult holds the result of a text prompt
type TextPromptResult struct {
	Value     string
	Cancelled bool
}

var (
	tpLabelStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	tpDefaultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	tpFooterStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

type textPromptModel struct {
	label        string
	defaultValue string
	textInput    textinput.Model
	result       TextPromptResult
}

// PromptText launches a Bubble Tea text input with Esc to cancel.
// Returns the entered value and whether it was cancelled.
// Empty input returns defaultValue.
func PromptText(label, defaultValue string) TextPromptResult {
	ti := textinput.New()
	ti.Placeholder = defaultValue
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	m := textPromptModel{
		label:        label,
		defaultValue: defaultValue,
		textInput:    ti,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return TextPromptResult{Cancelled: true}
	}

	fm := finalModel.(textPromptModel)
	return fm.result
}

func (m textPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.result = TextPromptResult{Cancelled: true}
			return m, tea.Quit
		case "enter":
			value := strings.TrimSpace(m.textInput.Value())
			if value == "" {
				value = m.defaultValue
			}
			m.result = TextPromptResult{Value: value}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m textPromptModel) View() string {
	var b strings.Builder

	defaultHint := ""
	if m.defaultValue != "" {
		defaultHint = tpDefaultStyle.Render(fmt.Sprintf(" [%s]", m.defaultValue))
	}

	b.WriteString(tpLabelStyle.Render("  "+m.label) + defaultHint)
	b.WriteString("\n")
	b.WriteString("  " + m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(tpFooterStyle.Render("  enter confirm • esc cancel"))

	return b.String()
}
