package bridge

import (
	"reflect"
	"testing"

	"github.com/kitdot/nsh/core"
)

func TestBuildTreeGroupsSkipsWildcardOnlyGroupsAndLabelsUncategorized(t *testing.T) {
	cfg := &core.NSHConfig{
		GroupOrder: []string{"infra", "wild-only", "Uncategorized"},
		Blocks: []core.NSHBlock{
			testHostBlock("app", "infra", "app.example.com", 2, false),
			testHostBlock("api", "infra", "api.example.com", 1, false),
			testHostBlock("wild", "wild-only", "*.example.com", 1, true),
			testHostBlock("misc", "Uncategorized", "misc.example.com", 1, false),
		},
	}

	groups := buildTreeGroups(cfg)
	if len(groups) != 2 {
		t.Fatalf("expected 2 visible groups, got %d", len(groups))
	}

	if groups[0].name != "infra" || groups[0].label != "infra" {
		t.Fatalf("unexpected first group: %#v", groups[0])
	}
	if groups[1].name != "Uncategorized" || groups[1].label != "-" {
		t.Fatalf("unexpected uncategorized group label: %#v", groups[1])
	}

	gotAliases := []string{groups[0].hosts[0].Alias, groups[0].hosts[1].Alias}
	wantAliases := []string{"api", "app"}
	if !reflect.DeepEqual(gotAliases, wantAliases) {
		t.Fatalf("unexpected host ordering in infra group: got %#v want %#v", gotAliases, wantAliases)
	}
}

func TestNewTreeModelInitializesExpandedGroupPinnedCopyAndViewMode(t *testing.T) {
	cfg := &core.NSHConfig{
		PinnedAliases: []string{"app"},
		Blocks: []core.NSHBlock{
			testHostBlock("app", "infra", "app.example.com", 1, false),
			testHostBlock("db-admin", "db", "db.internal.example.com", 1, false),
		},
	}

	groupsModel := newTreeModel("Hosts", cfg, false)
	if groupsModel.level != 1 {
		t.Fatalf("expected groups browser to start at host level, got %d", groupsModel.level)
	}
	if groupsModel.viewMode != 0 {
		t.Fatalf("expected groups browser view mode, got %d", groupsModel.viewMode)
	}
	if len(groupsModel.groups) == 0 || !groupsModel.groups[0].expanded {
		t.Fatal("expected first group to start expanded")
	}
	if groupsModel.maxAlias != len("db-admin") {
		t.Fatalf("unexpected max alias width: %d", groupsModel.maxAlias)
	}
	if groupsModel.maxHost != len("db.internal.example.com") {
		t.Fatalf("unexpected max host width: %d", groupsModel.maxHost)
	}

	groupsModel.pinnedAliases[0] = "changed"
	if cfg.PinnedAliases[0] != "app" {
		t.Fatalf("expected pinned aliases to be copied, config mutated to %#v", cfg.PinnedAliases)
	}

	pinnedModel := newTreeModel("Pinned", cfg, true)
	if pinnedModel.viewMode != 1 {
		t.Fatalf("expected pinned browser view mode, got %d", pinnedModel.viewMode)
	}
}

func TestStartPinnedReorderFiltersMissingAliasesAndEntersMoveMode(t *testing.T) {
	cfg := &core.NSHConfig{
		Blocks: []core.NSHBlock{
			testHostBlock("app", "infra", "app.example.com", 1, false),
			testHostBlock("db", "infra", "db.example.com", 2, false),
		},
	}

	m := treeModel{
		cfg:           cfg,
		pinnedAliases: []string{"ghost", "app", "db"},
		pinnedCursor:  99,
		filter:        "app",
		filtering:     true,
	}

	m.startPinnedReorder()

	wantAliases := []string{"app", "db"}
	if !reflect.DeepEqual(m.pinnedAliases, wantAliases) {
		t.Fatalf("unexpected valid pinned aliases: got %#v want %#v", m.pinnedAliases, wantAliases)
	}
	if !reflect.DeepEqual(m.pinnedOriginal, wantAliases) {
		t.Fatalf("unexpected original pinned snapshot: got %#v want %#v", m.pinnedOriginal, wantAliases)
	}
	if !m.pinnedMoving {
		t.Fatal("expected pinned reorder mode to start when multiple valid aliases remain")
	}
	if m.filtering || m.filter != "" {
		t.Fatalf("expected reorder to clear filter state, got filtering=%v filter=%q", m.filtering, m.filter)
	}
	if m.pinnedCursor != 1 {
		t.Fatalf("expected pinned cursor to clamp to last valid host, got %d", m.pinnedCursor)
	}
}

func TestResetGroupsFilterClearsFilterAndCollapsesGroups(t *testing.T) {
	cfg := &core.NSHConfig{
		Blocks: []core.NSHBlock{
			testHostBlock("app", "infra", "app.example.com", 1, false),
			testHostBlock("db", "db", "db.example.com", 1, false),
		},
	}

	m := newTreeModel("Hosts", cfg, false)
	for i := range m.groups {
		m.groups[i].expanded = true
	}
	m.filtering = true
	m.filter = "db"
	m.level = 1
	m.groupIdx = 1
	m.hostIdx = 3

	m.resetGroupsFilter()

	if m.filtering || m.filter != "" {
		t.Fatalf("expected filter state to reset, got filtering=%v filter=%q", m.filtering, m.filter)
	}
	if m.level != 0 || m.groupIdx != 0 || m.hostIdx != 0 {
		t.Fatalf("expected cursor to reset to top-level origin, got level=%d groupIdx=%d hostIdx=%d", m.level, m.groupIdx, m.hostIdx)
	}
	for i, group := range m.groups {
		if group.expanded {
			t.Fatalf("expected group %d to be collapsed after reset", i)
		}
	}
}

func testHostBlock(alias, group, hostName string, order int, wildcard bool) core.NSHBlock {
	return core.NSHBlock{
		Type: core.BlockHost,
		Host: &core.NSHHost{
			Alias:      alias,
			Group:      group,
			HostName:   hostName,
			Order:      order,
			IsWildcard: wildcard,
		},
		HostLine: "Host " + alias,
	}
}
