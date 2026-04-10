package bridge

import tea "github.com/charmbracelet/bubbletea"

func (m *treeModel) handlePinnedKey(key string) (tea.Cmd, bool) {
	if m.pinnedConfirm {
		return m.handlePinnedDeleteConfirmKey(key), true
	}
	if m.pinnedSaveConf {
		return m.handlePinnedSaveConfirmKey(key), true
	}
	if m.pinnedMoving {
		return m.handlePinnedReorderKey(key), true
	}

	switch key {
	case "ctrl+c":
		m.quitting = true
		return tea.Quit, true
	case "esc":
		if m.filtering {
			m.filtering = false
			m.filter = ""
			m.pinnedCursor = 0
			return nil, true
		}
		m.quitting = true
		return tea.Quit, true
	case "q":
		if m.filtering {
			m.filter += "q"
			m.pinnedCursor = 0
			return nil, true
		}
		m.quitting = true
		return tea.Quit, true
	case "tab":
		m.viewMode = 0
		m.filtering = false
		m.filter = ""
		return nil, true
	case "/":
		if !m.filtering {
			m.filtering = true
			m.filter = ""
		} else {
			m.filter += "/"
		}
		m.pinnedCursor = 0
		return nil, true
	case "backspace":
		if m.filtering && len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.pinnedCursor = 0
		}
		return nil, true
	case "up", "k":
		if m.filtering && key == "k" {
			m.filter += "k"
			m.pinnedCursor = 0
			return nil, true
		}
		fph := m.filteredPinnedHosts()
		if m.pinnedCursor > 0 && m.pinnedCursor < len(fph) {
			m.pinnedCursor--
		}
		return nil, true
	case "down", "j":
		if m.filtering && key == "j" {
			m.filter += "j"
			m.pinnedCursor = 0
			return nil, true
		}
		fph := m.filteredPinnedHosts()
		if m.pinnedCursor < len(fph)-1 {
			m.pinnedCursor++
		}
		return nil, true
	case "enter":
		fph := m.filteredPinnedHosts()
		if m.pinnedCursor < len(fph) {
			m.result = TreeResult{Alias: fph[m.pinnedCursor].Alias, Action: "connect"}
			return tea.Quit, true
		}
		return nil, true
	case " ":
		if !m.filtering {
			m.startPinnedReorder()
		}
		return nil, true
	case "p":
		if m.filtering {
			m.filter += "p"
			m.pinnedCursor = 0
			return nil, true
		}
		fph := m.filteredPinnedHosts()
		if m.pinnedCursor < len(fph) {
			m.togglePin(fph[m.pinnedCursor].Alias)
			fph = m.filteredPinnedHosts()
			if m.pinnedCursor >= len(fph) && m.pinnedCursor > 0 {
				m.pinnedCursor--
			}
		}
		return nil, true
	case "e":
		if m.filtering {
			m.filter += "e"
			m.pinnedCursor = 0
			return nil, true
		}
		fph := m.filteredPinnedHosts()
		if m.pinnedCursor < len(fph) {
			m.result = TreeResult{Alias: fph[m.pinnedCursor].Alias, Action: "edit"}
			return tea.Quit, true
		}
		return nil, true
	case "d":
		if m.filtering {
			m.filter += "d"
			m.pinnedCursor = 0
			return nil, true
		}
		fph := m.filteredPinnedHosts()
		if m.pinnedCursor < len(fph) {
			m.pinnedConfirm = true
			m.pinnedConfIdx = 0
		}
		return nil, true
	default:
		if m.filtering {
			m.filter += key
			m.pinnedCursor = 0
		}
		return nil, true
	}
}

func (m *treeModel) handlePinnedDeleteConfirmKey(key string) tea.Cmd {
	switch key {
	case "tab", "right", "left":
		if m.pinnedConfIdx == 0 {
			m.pinnedConfIdx = 1
		} else {
			m.pinnedConfIdx = 0
		}
	case "enter":
		if m.pinnedConfIdx == 1 {
			fph := m.filteredPinnedHosts()
			if m.pinnedCursor < len(fph) {
				m.result = TreeResult{Alias: fph[m.pinnedCursor].Alias, Action: "delete"}
				return tea.Quit
			}
		}
		m.pinnedConfirm = false
	case "esc":
		m.pinnedConfirm = false
	case "ctrl+c", "q":
		m.quitting = true
		return tea.Quit
	}
	return nil
}

func (m *treeModel) handlePinnedSaveConfirmKey(key string) tea.Cmd {
	switch key {
	case "tab", "right", "left":
		if m.pinnedSaveIdx == 0 {
			m.pinnedSaveIdx = 1
		} else {
			m.pinnedSaveIdx = 0
		}
	case "enter":
		if m.pinnedSaveIdx == 1 {
			m.pinnedSaveConf = false
		} else {
			m.restorePinnedOrder()
			m.pinnedSaveConf = false
		}
	case "esc":
		m.restorePinnedOrder()
		m.pinnedSaveConf = false
	case "ctrl+c", "q":
		m.quitting = true
		return tea.Quit
	}
	return nil
}

func (m *treeModel) handlePinnedReorderKey(key string) tea.Cmd {
	switch key {
	case "up", "k":
		if m.pinnedCursor > 0 {
			newAliases := make([]string, len(m.pinnedAliases))
			copy(newAliases, m.pinnedAliases)
			newAliases[m.pinnedCursor], newAliases[m.pinnedCursor-1] =
				newAliases[m.pinnedCursor-1], newAliases[m.pinnedCursor]
			m.pinnedAliases = newAliases
			m.pinnedCursor--
		}
	case "down", "j":
		if m.pinnedCursor < len(m.pinnedAliases)-1 {
			newAliases := make([]string, len(m.pinnedAliases))
			copy(newAliases, m.pinnedAliases)
			newAliases[m.pinnedCursor], newAliases[m.pinnedCursor+1] =
				newAliases[m.pinnedCursor+1], newAliases[m.pinnedCursor]
			m.pinnedAliases = newAliases
			m.pinnedCursor++
		}
	case " ", "enter":
		m.pinnedMoving = false
		m.pinnedSaveConf = true
		m.pinnedSaveIdx = 0
	case "esc":
		m.pinnedMoving = false
		m.restorePinnedOrder()
	case "ctrl+c":
		m.quitting = true
		return tea.Quit
	}
	return nil
}

func (m *treeModel) startPinnedReorder() {
	var valid []string
	for _, a := range m.pinnedAliases {
		if m.cfg.HostByAlias(a) != nil {
			valid = append(valid, a)
		}
	}
	m.pinnedAliases = valid
	if len(m.pinnedAliases) <= 1 {
		return
	}

	orig := make([]string, len(m.pinnedAliases))
	copy(orig, m.pinnedAliases)
	m.pinnedOriginal = orig
	m.pinnedMoving = true
	m.filter = ""
	m.filtering = false
	if m.pinnedCursor >= len(m.pinnedAliases) {
		m.pinnedCursor = len(m.pinnedAliases) - 1
	}
}

func (m *treeModel) restorePinnedOrder() {
	m.pinnedAliases = m.pinnedOriginal
	m.pinnedOriginal = nil
}
