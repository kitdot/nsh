package cmd

import (
	"reflect"
	"testing"

	"github.com/kitdot/nsh/core"
)

func TestBuildManagedHostBlock(t *testing.T) {
	host := &core.NSHHost{
		Group:        "infra",
		Desc:         "app server",
		Auth:         "key",
		Alias:        "app",
		HostName:     "app.example.com",
		User:         "deploy",
		Port:         "2200",
		IdentityFile: "~/.ssh/nsh/id_app",
	}

	block := buildManagedHostBlock(host)

	if block.HostLine != "Host app" {
		t.Fatalf("unexpected host line: %q", block.HostLine)
	}
	if block.NshLine == "" {
		t.Fatal("expected nsh metadata line to be built")
	}

	wantRaw := []string{
		"    HostName app.example.com",
		"    User deploy",
		"    Port 2200",
		"    IdentityFile ~/.ssh/nsh/id_app",
	}
	if !reflect.DeepEqual(block.Host.RawPropertyLines, wantRaw) {
		t.Fatalf("unexpected raw property lines: %#v", block.Host.RawPropertyLines)
	}
}

func TestRewriteIdentityFileLines(t *testing.T) {
	lines := []string{
		"    HostName app.example.com",
		"    IdentityFile ~/.ssh/id_old",
		"    User deploy",
	}

	replaced := rewriteIdentityFileLines(lines, "~/.ssh/nsh/id_new")
	wantReplaced := []string{
		"    HostName app.example.com",
		"    IdentityFile ~/.ssh/nsh/id_new",
		"    User deploy",
	}
	if !reflect.DeepEqual(replaced, wantReplaced) {
		t.Fatalf("unexpected replaced lines: %#v", replaced)
	}

	added := rewriteIdentityFileLines([]string{"    HostName app.example.com"}, "~/.ssh/nsh/id_new")
	wantAdded := []string{
		"    HostName app.example.com",
		"    IdentityFile ~/.ssh/nsh/id_new",
	}
	if !reflect.DeepEqual(added, wantAdded) {
		t.Fatalf("unexpected appended identity line: %#v", added)
	}

	removed := rewriteIdentityFileLines(lines, "")
	wantRemoved := []string{
		"    HostName app.example.com",
		"    User deploy",
	}
	if !reflect.DeepEqual(removed, wantRemoved) {
		t.Fatalf("unexpected lines after identity removal: %#v", removed)
	}
}
