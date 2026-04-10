package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	pathPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	pathHintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

type pathInputModel struct {
	textInput textinput.Model
	label     string
	result    string
	quitting  bool
	hint      string
}

// PromptPath launches a Bubble Tea text input with Tab file path completion.
// Returns the entered path or "" if cancelled.
func PromptPath(label string) string {
	ti := textinput.New()
	ti.Placeholder = "~/path/to/file"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	m := pathInputModel{
		textInput: ti,
		label:     label,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return ""
	}

	fm := finalModel.(pathInputModel)
	return fm.result
}

func (m pathInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m pathInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			m.result = strings.TrimSpace(m.textInput.Value())
			return m, tea.Quit

		case "tab":
			m.completePath()
			m.hint = ""
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	// Update hint with possible completions
	m.hint = m.getHint()

	return m, cmd
}

func (m pathInputModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(pathPromptStyle.Render("  "+m.label) + "\n")
	b.WriteString("  " + m.textInput.View())

	if m.hint != "" {
		b.WriteString("\n" + pathHintStyle.Render("  "+m.hint))
	}

	b.WriteString("\n\n" + pathHintStyle.Render("  tab complete • enter confirm • esc cancel"))

	return b.String()
}

func (m *pathInputModel) completePath() {
	current := m.textInput.Value()
	if current == "" {
		return
	}

	expanded := expandHome(current)
	matches := globPath(expanded)

	if len(matches) == 0 {
		return
	}

	if len(matches) == 1 {
		result := unexpandHome(matches[0])
		// If it's a directory, add trailing /
		if info, err := os.Stat(matches[0]); err == nil && info.IsDir() {
			if !strings.HasSuffix(result, "/") {
				result += "/"
			}
		}
		m.textInput.SetValue(result)
		m.textInput.CursorEnd()
		return
	}

	// Multiple matches — find common prefix
	common := longestCommonPrefix(matches)
	if common != "" && len(common) > len(expanded) {
		result := unexpandHome(common)
		m.textInput.SetValue(result)
		m.textInput.CursorEnd()
	}
}

func (m *pathInputModel) getHint() string {
	current := m.textInput.Value()
	if current == "" {
		return ""
	}

	expanded := expandHome(current)
	matches := globPath(expanded)

	if len(matches) <= 1 {
		return ""
	}

	// Show up to 5 matches
	var names []string
	for i, match := range matches {
		if i >= 5 {
			names = append(names, fmt.Sprintf("... +%d more", len(matches)-5))
			break
		}
		name := filepath.Base(match)
		if info, err := os.Stat(match); err == nil && info.IsDir() {
			name += "/"
		}
		names = append(names, name)
	}

	return strings.Join(names, "  ")
}

func globPath(path string) []string {
	// If path ends with /, list contents of directory
	if strings.HasSuffix(path, "/") {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		var results []string
		for _, e := range entries {
			if !strings.HasPrefix(e.Name(), ".") {
				results = append(results, filepath.Join(path, e.Name()))
			}
		}
		return results
	}

	// Otherwise glob with wildcard
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var results []string
	lowerBase := strings.ToLower(base)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") && !strings.HasPrefix(base, ".") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(e.Name()), lowerBase) {
			results = append(results, filepath.Join(dir, e.Name()))
		}
	}
	return results
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

func unexpandHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for len(prefix) > 0 && !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}
