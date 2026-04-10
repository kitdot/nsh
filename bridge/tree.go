package bridge

import (
	"github.com/kitdot/nsh/core"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	treeTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	treeGroupStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	treeGroupActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("212"))

	treeHostActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("212"))

	treeHostNormalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	treeDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	treeFooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	treeHostNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	treeAuthPassStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220"))

	treeAuthKeyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("34"))

	treeAuthNoneStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	treePinStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("208"))

	treePanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("245")).
			Padding(0, 1)

	treePanelTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39"))

	treePanelKeyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250"))

	treePanelValStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	treeConfirmYesStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("196"))

	treeConfirmNoStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("34"))
)

const (
	treeIconPass    = "●"
	treeIconKey     = "◆"
	treeIconDefault = "○"
)

// TreeResult holds the result of a tree browser interaction
type TreeResult struct {
	Alias         string
	Action        string // "connect", "edit", "delete", "new", "pin", or ""
	PinnedAliases []string
}

type treeGroup struct {
	name     string
	label    string
	hosts    []core.NSHHost
	expanded bool
}

type treeModel struct {
	title          string
	groups         []treeGroup
	level          int // 0 = group, 1 = host, 2 = delete confirm
	groupIdx       int
	hostIdx        int
	confirmIdx     int // 0 = No, 1 = Yes
	result         TreeResult
	quitting       bool
	filter         string
	filtering      bool
	maxAlias       int
	maxHost        int
	width          int
	height         int
	pinnedAliases  []string
	cfg            *core.NSHConfig
	viewMode       int      // 0 = groups, 1 = pinned
	pinnedCursor   int      // cursor for pinned view
	pinnedConfirm  bool     // delete confirm in pinned view
	pinnedConfIdx  int      // 0 = No, 1 = Yes
	pinnedMoving   bool     // reorder mode in pinned view
	pinnedSaveConf bool     // save confirm after reorder
	pinnedSaveIdx  int      // 0 = Cancel, 1 = Save
	pinnedOriginal []string // original order before reorder
}

// TreeBrowserPinned launches the tree browser starting in pinned view.
func TreeBrowserPinned(title string, cfg *core.NSHConfig) TreeResult {
	m := newTreeModel(title, cfg, true)
	if len(m.groups) == 0 {
		return TreeResult{}
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return TreeResult{}
	}

	fm := finalModel.(treeModel)
	fm.result.PinnedAliases = fm.pinnedAliases
	return fm.result
}

// TreeBrowser launches a grouped host browser with hotkeys.
// Returns a TreeResult with alias and action.
func TreeBrowser(title string, cfg *core.NSHConfig) TreeResult {
	m := newTreeModel(title, cfg, false)
	if len(m.groups) == 0 {
		return TreeResult{}
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return TreeResult{}
	}

	fm := finalModel.(treeModel)
	fm.result.PinnedAliases = fm.pinnedAliases
	return fm.result
}

func buildTreeGroups(cfg *core.NSHConfig) []treeGroup {
	var groups []treeGroup
	for _, g := range cfg.SortedGroups() {
		hosts := cfg.HostsInGroup(g)
		var filtered []core.NSHHost
		for _, h := range hosts {
			if !h.IsWildcard {
				filtered = append(filtered, h)
			}
		}
		if len(filtered) == 0 {
			continue
		}
		label := g
		if g == "Uncategorized" {
			label = "-"
		}
		groups = append(groups, treeGroup{
			name:  g,
			label: label,
			hosts: filtered,
		})
	}
	return groups
}

func (m treeModel) Init() tea.Cmd {
	return nil
}
