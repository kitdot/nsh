package bridge

import tea "github.com/charmbracelet/bubbletea"

func (m *treeModel) handleGroupsKey(key string) (tea.Cmd, bool) {
	switch {
	case key == "ctrl+c":
		m.quitting = true
		return tea.Quit, true

	case key == "tab":
		if !m.filtering {
			m.viewMode = 1
			m.pinnedCursor = 0
			return nil, true
		}

	case key == "q":
		if m.filtering {
			m.filter += "q"
			m.applyFilterExpand()
			return nil, true
		}
		m.quitting = true
		return tea.Quit, true

	case key == "esc":
		if m.filtering {
			m.resetGroupsFilter()
			return nil, true
		}
		if m.level == 1 {
			vg := m.visibleGroups()
			if m.groupIdx < len(vg) {
				m.groups[vg[m.groupIdx]].expanded = false
			}
			m.level = 0
			m.hostIdx = 0
			return nil, true
		}
		m.quitting = true
		return tea.Quit, true

	case key == "/":
		if !m.filtering {
			m.filtering = true
			m.filter = ""
			return nil, true
		}
		m.filter += "/"
		m.applyFilterExpand()
		return nil, true

	case key == "backspace":
		if m.filtering && len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilterExpand()
		}
		return nil, true

	case key == "e":
		if m.filtering {
			m.filter += "e"
			m.applyFilterExpand()
			return nil, true
		}
		if m.level == 1 {
			curHost := m.getCurrentHost()
			if curHost != nil {
				m.result = TreeResult{Alias: curHost.Alias, Action: "edit"}
				return tea.Quit, true
			}
		}
		return nil, true

	case key == "d":
		if m.filtering {
			m.filter += "d"
			m.applyFilterExpand()
			return nil, true
		}
		if m.level == 1 {
			m.level = 2
			m.confirmIdx = 0
		}
		return nil, true

	case key == "n":
		if m.filtering {
			m.filter += "n"
			m.applyFilterExpand()
			return nil, true
		}
		m.result = TreeResult{Action: "new"}
		return tea.Quit, true

	case key == "p":
		if m.filtering {
			m.filter += "p"
			m.applyFilterExpand()
			return nil, true
		}
		if m.level == 1 {
			curHost := m.getCurrentHost()
			if curHost != nil {
				m.togglePin(curHost.Alias)
			}
		}
		return nil, true

	case key == "up" || key == "k":
		if m.filtering && key == "k" {
			m.filter += "k"
			m.applyFilterExpand()
			return nil, true
		}
		m.moveUp()
		return nil, true

	case key == "down" || key == "j":
		if m.filtering && key == "j" {
			m.filter += "j"
			m.applyFilterExpand()
			return nil, true
		}
		m.moveDown()
		return nil, true

	case key == "left" || key == "h":
		if m.filtering && key == "h" {
			m.filter += "h"
			m.applyFilterExpand()
			return nil, true
		}
		if m.level == 1 && !m.filtering {
			vg := m.visibleGroups()
			if m.groupIdx < len(vg) {
				m.groups[vg[m.groupIdx]].expanded = false
			}
			m.level = 0
			m.hostIdx = 0
		}
		return nil, true

	case key == "right" || key == "l":
		if m.filtering && key == "l" {
			m.filter += "l"
			m.applyFilterExpand()
			return nil, true
		}
		if m.level == 0 && !m.filtering {
			m.expandCurrent()
		}
		return nil, true

	case key == "enter":
		if m.filtering && m.level == 1 {
			curHost := m.getCurrentHost()
			if curHost != nil {
				m.result = TreeResult{Alias: curHost.Alias, Action: "connect"}
				return tea.Quit, true
			}
			return nil, true
		}
		if m.filtering {
			m.filtering = false
			return nil, true
		}
		if m.level == 0 {
			m.expandCurrent()
		} else if m.level == 1 {
			curHost := m.getCurrentHost()
			if curHost != nil {
				m.result = TreeResult{Alias: curHost.Alias, Action: "connect"}
				return tea.Quit, true
			}
		}
		return nil, true

	default:
		if m.filtering {
			m.filter += key
			m.applyFilterExpand()
		}
		return nil, true
	}

	return nil, false
}

func (m *treeModel) resetGroupsFilter() {
	m.filtering = false
	m.filter = ""
	m.collapseAll()
	m.level = 0
	m.groupIdx = 0
	m.hostIdx = 0
}
