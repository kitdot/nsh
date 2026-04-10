package bridge

import tea "github.com/charmbracelet/bubbletea"

func (m treeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.level == 2 {
			if cmd, handled := m.handleDeleteConfirmKey(msg.String()); handled {
				return m, cmd
			}
		}

		if m.viewMode == 1 {
			if cmd, handled := m.handlePinnedKey(msg.String()); handled {
				return m, cmd
			}
		}

		if cmd, handled := m.handleGroupsKey(msg.String()); handled {
			return m, cmd
		}
	}
	return m, nil
}

func (m *treeModel) handleDeleteConfirmKey(key string) (tea.Cmd, bool) {
	switch key {
	case "tab", "right", "left":
		if m.confirmIdx == 0 {
			m.confirmIdx = 1
		} else {
			m.confirmIdx = 0
		}
	case "enter":
		if m.confirmIdx == 1 {
			curHost := m.getCurrentHost()
			if curHost != nil {
				m.result = TreeResult{Alias: curHost.Alias, Action: "delete"}
				return tea.Quit, true
			}
		}
		m.level = 1
	case "esc":
		m.level = 1
	case "ctrl+c", "q":
		m.quitting = true
		return tea.Quit, true
	}
	return nil, true
}
