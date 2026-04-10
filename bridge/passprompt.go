package bridge

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type passPromptModel struct {
	label     string
	hasOld    bool // whether there's an existing password (Enter to keep)
	textInput textinput.Model
	result    TextPromptResult
}

// PromptPassword launches a Bubble Tea password input (masked).
// If hasExisting is true, empty Enter keeps the old password (returns empty Value, not cancelled).
func PromptPassword(label string, hasExisting bool) TextPromptResult {
	ti := textinput.New()
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50
	if hasExisting {
		ti.Placeholder = "Enter to keep current"
	}

	m := passPromptModel{
		label:     label,
		hasOld:    hasExisting,
		textInput: ti,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return TextPromptResult{Cancelled: true}
	}

	fm := finalModel.(passPromptModel)
	return fm.result
}

func (m passPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m passPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.result = TextPromptResult{Cancelled: true}
			return m, tea.Quit
		case "enter":
			value := strings.TrimSpace(m.textInput.Value())
			m.result = TextPromptResult{Value: value}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m passPromptModel) View() string {
	var b strings.Builder

	hint := ""
	if m.hasOld {
		hint = tpDefaultStyle.Render(" (Enter to keep current)")
	}

	b.WriteString(tpLabelStyle.Render("  "+m.label) + hint)
	b.WriteString("\n")
	b.WriteString("  " + m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(tpFooterStyle.Render("  enter confirm • esc cancel"))

	return b.String()
}
