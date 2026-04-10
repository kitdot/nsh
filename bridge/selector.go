package bridge

import (
	"fmt"
	"strings"

	"github.com/kitdot/nsh/core"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	groupStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	hostNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	authPassStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	authKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("34"))

	authNoneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	filterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			MarginTop(1)

	previewBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1).
			MarginLeft(2)

	previewTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39"))

	previewKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	previewValStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)

// selectorItem is a display item (either group header or host)
type selectorItem struct {
	isGroup bool
	group   string
	host    *core.NSHHost
}

// selectorModel is the Bubble Tea model for host selection
type selectorModel struct {
	items     []selectorItem
	allItems  []selectorItem
	cursor    int
	selected  string
	filter    string
	filtering bool
	width     int
	height    int
	quitting  bool
}

// SelectHostTUI launches the Bubble Tea interactive host selector
func SelectHostTUI(hosts []core.NSHHost, groupOrder []string) string {
	filtered := filterWildcard(hosts)
	if len(filtered) == 0 {
		return ""
	}

	sorted := sortHosts(filtered, groupOrder)

	// Build display items grouped
	items := buildGroupedItems(sorted, groupOrder)

	m := selectorModel{
		items:    items,
		allItems: items,
		cursor:   findFirstHost(items),
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return ""
	}

	fm := finalModel.(selectorModel)
	return fm.selected
}

func buildGroupedItems(hosts []core.NSHHost, groupOrder []string) []selectorItem {
	// Group hosts by group name
	groupMap := map[string][]core.NSHHost{}
	var groupNames []string
	seen := map[string]bool{}

	// Determine group order
	groupRank := map[string]int{}
	for i, g := range groupOrder {
		groupRank[g] = i
	}

	for _, h := range hosts {
		if !seen[h.Group] {
			seen[h.Group] = true
			groupNames = append(groupNames, h.Group)
		}
		groupMap[h.Group] = append(groupMap[h.Group], h)
	}

	// Sort groups
	maxRank := len(groupOrder)
	for i := 1; i < len(groupNames); i++ {
		for j := i; j > 0; j-- {
			a, b := groupNames[j], groupNames[j-1]
			aRank := getRank(a, groupRank, maxRank)
			bRank := getRank(b, groupRank, maxRank)
			if aRank < bRank || (aRank == bRank && a < b) {
				groupNames[j], groupNames[j-1] = groupNames[j-1], groupNames[j]
			} else {
				break
			}
		}
	}

	var items []selectorItem
	for _, g := range groupNames {
		items = append(items, selectorItem{isGroup: true, group: g})
		for i := range groupMap[g] {
			items = append(items, selectorItem{host: &groupMap[g][i]})
		}
	}
	return items
}

func findFirstHost(items []selectorItem) int {
	for i, item := range items {
		if !item.isGroup {
			return i
		}
	}
	return 0
}

func (m selectorModel) Init() tea.Cmd {
	return nil
}

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c", msg.String() == "q":
			if m.filtering {
				m.filtering = false
				m.filter = ""
				m.applyFilter()
				return m, nil
			}
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
			m.filtering = true
			m.filter = ""
			return m, nil

		case msg.String() == "backspace":
			if m.filtering && len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.applyFilter()
			}
			return m, nil

		case msg.String() == "up", msg.String() == "k":
			if !m.filtering {
				m.moveCursorUp()
			}
			return m, nil

		case msg.String() == "down", msg.String() == "j":
			if !m.filtering {
				m.moveCursorDown()
			}
			return m, nil

		case msg.String() == "enter":
			if m.filtering {
				m.filtering = false
				return m, nil
			}
			if m.cursor >= 0 && m.cursor < len(m.items) && !m.items[m.cursor].isGroup {
				m.selected = m.items[m.cursor].host.Alias
				return m, tea.Quit
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

func (m *selectorModel) moveCursorUp() {
	for i := m.cursor - 1; i >= 0; i-- {
		if !m.items[i].isGroup {
			m.cursor = i
			return
		}
	}
}

func (m *selectorModel) moveCursorDown() {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if !m.items[i].isGroup {
			m.cursor = i
			return
		}
	}
}

func (m *selectorModel) applyFilter() {
	if m.filter == "" {
		m.items = m.allItems
		m.cursor = findFirstHost(m.items)
		return
	}

	var filtered []selectorItem
	lastGroup := ""

	for _, item := range m.allItems {
		if item.isGroup {
			lastGroup = item.group
			continue
		}
		if item.host == nil {
			continue
		}
		h := item.host
		hostText := h.Alias + " " + h.HostName + " " + h.Desc + " " + h.Group
		match := matchAllTerms(m.filter, hostText)
		if match {
			// Add group header if not already added
			if len(filtered) == 0 || !filtered[len(filtered)-1].isGroup || filtered[len(filtered)-1].group != lastGroup {
				// Check if last item is already this group header
				addGroup := true
				for _, f := range filtered {
					if f.isGroup && f.group == lastGroup {
						addGroup = false
						break
					}
				}
				if addGroup {
					filtered = append(filtered, selectorItem{isGroup: true, group: lastGroup})
				}
			}
			filtered = append(filtered, item)
		}
	}

	m.items = filtered
	m.cursor = findFirstHost(m.items)
}

func (m selectorModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("  SSH Host Selector"))
	b.WriteString("\n")

	// Filter bar
	if m.filtering {
		b.WriteString(filterStyle.Render("  / " + m.filter + "█"))
		b.WriteString("\n")
	}

	// Calculate available height for list
	listHeight := m.height - 6 // title + filter + footer
	if listHeight < 5 {
		listHeight = 5
	}

	// Determine preview width
	previewWidth := 0
	showPreview := m.width > 80
	listWidth := m.width
	if showPreview {
		previewWidth = 35
		if previewWidth > m.width/3 {
			previewWidth = m.width / 3
		}
		listWidth = m.width - previewWidth - 4
	}

	// Compute visible window
	start, end := computeWindow(m.cursor, len(m.items), listHeight)

	// Render list
	var listLines []string
	for i := start; i < end; i++ {
		item := m.items[i]
		if item.isGroup {
			listLines = append(listLines, renderGroupHeader(item.group))
		} else if item.host != nil {
			isCursor := i == m.cursor
			listLines = append(listLines, renderHostLine(item.host, isCursor))
		}
	}

	if showPreview {
		// Render preview panel
		var previewContent string
		if m.cursor >= 0 && m.cursor < len(m.items) && !m.items[m.cursor].isGroup && m.items[m.cursor].host != nil {
			previewContent = renderPreview(m.items[m.cursor].host, previewWidth-4)
		}

		// Side by side layout
		listStr := strings.Join(listLines, "\n")

		previewRendered := previewBoxStyle.Width(previewWidth).Render(previewContent)

		joined := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(listWidth).Render(listStr),
			previewRendered,
		)
		b.WriteString(joined)
	} else {
		b.WriteString(strings.Join(listLines, "\n"))
	}

	b.WriteString("\n")

	// Footer
	footer := "  ↑↓/jk navigate • enter select • / filter • esc quit"
	b.WriteString(footerStyle.Render(footer))

	return b.String()
}

func computeWindow(cursor, total, height int) (int, int) {
	if total <= height {
		return 0, total
	}

	half := height / 2
	start := cursor - half
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > total {
		end = total
		start = end - height
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

func renderGroupHeader(group string) string {
	if group == "Uncategorized" {
		return groupStyle.Render("  -")
	}
	return groupStyle.Render(fmt.Sprintf("  - %s", group))
}

func renderHostLine(host *core.NSHHost, isCursor bool) string {
	cursor := "  "
	if isCursor {
		cursor = "> "
	}

	var icon string
	switch host.Auth {
	case "password":
		icon = authPassStyle.Render(iconPass)
	case "key":
		icon = authKeyStyle.Render(iconKey)
	default:
		icon = authNoneStyle.Render(iconDefault)
	}

	alias := host.Alias
	hostName := host.HostName
	desc := ""
	if host.Desc != "" {
		desc = dimStyle.Render(" - " + host.Desc)
	}

	if isCursor {
		return fmt.Sprintf("  %s %s  %s  %s%s",
			selectedStyle.Render(cursor),
			icon,
			selectedStyle.Render(pad(alias, 20)),
			hostNameStyle.Render(pad(hostName, 18)),
			desc,
		)
	}

	return fmt.Sprintf("  %s %s  %s  %s%s",
		normalStyle.Render(cursor),
		icon,
		normalStyle.Render(pad(alias, 20)),
		dimStyle.Render(pad(hostName, 18)),
		desc,
	)
}

func renderPreview(host *core.NSHHost, width int) string {
	var b strings.Builder

	b.WriteString(previewTitleStyle.Render(host.Alias))
	b.WriteString("\n\n")

	addField := func(key, val string) {
		if val != "" {
			b.WriteString(previewKeyStyle.Render(pad(key+":", 15)))
			b.WriteString(previewValStyle.Render(val))
			b.WriteString("\n")
		}
	}

	addField("HostName", host.HostName)
	addField("User", host.User)
	if host.Port != "" && host.Port != "22" {
		addField("Port", host.Port)
	}
	addField("IdentityFile", host.IdentityFile)
	if host.Auth != "" {
		addField("Auth", host.Auth)
	}
	addField("Group", host.Group)
	if host.Order > 0 {
		addField("Order", fmt.Sprintf("%d", host.Order))
	}
	addField("Description", host.Desc)

	if host.IsWildcard {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("[global-default]"))
	}

	return b.String()
}
