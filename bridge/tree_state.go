package bridge

import (
	"strings"

	"github.com/kitdot/nsh/core"
)

func (m *treeModel) togglePin(alias string) {
	found := false
	for i, a := range m.pinnedAliases {
		if a == alias {
			m.pinnedAliases = append(m.pinnedAliases[:i], m.pinnedAliases[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		m.pinnedAliases = append(m.pinnedAliases, alias)
	}
}

func (m *treeModel) pinnedHosts() []core.NSHHost {
	var result []core.NSHHost
	for _, alias := range m.pinnedAliases {
		if h := m.cfg.HostByAlias(alias); h != nil {
			result = append(result, *h)
		}
	}
	return result
}

func (m *treeModel) filteredPinnedHosts() []core.NSHHost {
	ph := m.pinnedHosts()
	if m.filter == "" {
		return ph
	}
	var result []core.NSHHost
	for _, h := range ph {
		text := h.Alias + " " + h.HostName + " " + h.Desc + " " + h.Group
		if matchAllTerms(m.filter, text) {
			result = append(result, h)
		}
	}
	return result
}

func (m *treeModel) isPinned(alias string) bool {
	for _, a := range m.pinnedAliases {
		if a == alias {
			return true
		}
	}
	return false
}

func (m *treeModel) expandCurrent() {
	vg := m.visibleGroups()
	if m.groupIdx < len(vg) {
		m.groups[vg[m.groupIdx]].expanded = true
		m.level = 1
		m.hostIdx = 0
	}
}

func (m *treeModel) collapseAll() {
	for i := range m.groups {
		m.groups[i].expanded = false
	}
}

func (m *treeModel) visibleGroups() []int {
	if m.filter == "" {
		var indices []int
		for i := range m.groups {
			indices = append(indices, i)
		}
		return indices
	}
	var indices []int
	for i, g := range m.groups {
		groupText := g.name + " " + g.label
		if matchAllTerms(m.filter, groupText) {
			indices = append(indices, i)
			continue
		}
		for _, h := range g.hosts {
			hostText := h.Alias + " " + h.HostName + " " + h.Desc + " " + h.Group
			if matchAllTerms(m.filter, hostText) {
				indices = append(indices, i)
				break
			}
		}
	}
	return indices
}

func (m *treeModel) visibleHosts(groupIdx int) []core.NSHHost {
	if groupIdx >= len(m.groups) {
		return nil
	}
	if m.filter == "" {
		return m.groups[groupIdx].hosts
	}
	g := m.groups[groupIdx]
	groupText := g.name + " " + g.label
	if matchAllTerms(m.filter, groupText) {
		return g.hosts
	}
	var result []core.NSHHost
	for _, h := range g.hosts {
		hostText := h.Alias + " " + h.HostName + " " + h.Desc + " " + h.Group
		if matchAllTerms(m.filter, hostText) {
			result = append(result, h)
		}
	}
	return result
}

func (m *treeModel) applyFilterExpand() {
	if m.filter == "" {
		m.collapseAll()
		m.level = 0
		m.groupIdx = 0
		m.hostIdx = 0
		return
	}
	vg := m.visibleGroups()
	for i := range m.groups {
		m.groups[i].expanded = false
	}
	for _, gi := range vg {
		m.groups[gi].expanded = true
	}
	m.level = 1
	m.groupIdx = 0
	m.hostIdx = 0
}

type flatHost struct {
	groupVisIdx int
	hostIdx     int
	host        core.NSHHost
}

func (m *treeModel) flatVisibleHosts() []flatHost {
	var result []flatHost
	vg := m.visibleGroups()
	for vi, gi := range vg {
		hosts := m.visibleHosts(gi)
		for hi, h := range hosts {
			result = append(result, flatHost{groupVisIdx: vi, hostIdx: hi, host: h})
		}
	}
	return result
}

func (m *treeModel) flatCursorIndex() int {
	flat := m.flatVisibleHosts()
	for idx, fh := range flat {
		if fh.groupVisIdx == m.groupIdx && fh.hostIdx == m.hostIdx {
			return idx
		}
	}
	return 0
}

func (m *treeModel) setFlatCursor(flatIdx int) {
	flat := m.flatVisibleHosts()
	if flatIdx >= 0 && flatIdx < len(flat) {
		m.groupIdx = flat[flatIdx].groupVisIdx
		m.hostIdx = flat[flatIdx].hostIdx
	}
}

func (m *treeModel) moveUp() {
	if m.level == 0 {
		if m.groupIdx > 0 {
			m.groupIdx--
		}
	} else if m.filtering {
		fi := m.flatCursorIndex()
		if fi > 0 {
			m.setFlatCursor(fi - 1)
		}
	} else {
		if m.hostIdx > 0 {
			m.hostIdx--
		} else {
			vg := m.visibleGroups()
			curVi := -1
			for vi, gi := range vg {
				if gi == m.findActualGroupIdx() {
					curVi = vi
					break
				}
			}
			if curVi > 0 {
				prevGi := vg[curVi-1]
				m.groups[prevGi].expanded = true
				hosts := m.visibleHosts(prevGi)
				m.groupIdx = curVi - 1
				m.hostIdx = len(hosts) - 1
			}
		}
	}
}

func (m *treeModel) moveDown() {
	if m.level == 0 {
		vg := m.visibleGroups()
		if m.groupIdx < len(vg)-1 {
			m.groupIdx++
		}
	} else if m.filtering {
		flat := m.flatVisibleHosts()
		fi := m.flatCursorIndex()
		if fi < len(flat)-1 {
			m.setFlatCursor(fi + 1)
		}
	} else {
		vg := m.visibleGroups()
		if m.groupIdx < len(vg) {
			hosts := m.visibleHosts(vg[m.groupIdx])
			if m.hostIdx < len(hosts)-1 {
				m.hostIdx++
			} else if m.groupIdx < len(vg)-1 {
				nextGi := vg[m.groupIdx+1]
				m.groups[nextGi].expanded = true
				m.groupIdx = m.groupIdx + 1
				m.hostIdx = 0
			}
		}
	}
}

func (m *treeModel) findActualGroupIdx() int {
	vg := m.visibleGroups()
	if m.groupIdx < len(vg) {
		return vg[m.groupIdx]
	}
	return -1
}

func (m *treeModel) getCurrentHost() *core.NSHHost {
	if m.level == 0 {
		return nil
	}
	vg := m.visibleGroups()
	if m.groupIdx >= len(vg) {
		return nil
	}
	hosts := m.visibleHosts(vg[m.groupIdx])
	if m.hostIdx >= len(hosts) {
		return nil
	}
	h := hosts[m.hostIdx]
	return &h
}

func stripAnsi(s string) string {
	var out strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}
