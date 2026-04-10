package core

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseSerializeRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		content string
		check   func(t *testing.T, cfg *NSHConfig)
	}{
		{
			name: "mixed config with nsh metadata",
			content: strings.Join([]string{
				"# global comment",
				"# nsh-groups: infra, db",
				"# nsh-pinned: api, worker",
				"",
				"# nsh: group=infra, desc=API server, auth=key, order=2",
				"Host api",
				"    HostName api.example.com",
				"    User deploy",
				"    Port 2222",
				"    IdentityFile ~/.ssh/id_api",
				"    ServerAliveInterval 30",
				"",
				"Match host bastion",
				"    User jump",
				"",
				"Include ~/.ssh/common.conf",
				"",
				"# nsh: group=infra, desc=wildcard",
				"Host *",
				"    ForwardAgent yes",
				"",
				"# nsh: desc=Worker, auth=password, order=3",
				"Host worker",
				"    HostName worker.example.com",
				"    User ops",
				"",
			}, "\n"),
			check: func(t *testing.T, cfg *NSHConfig) {
				t.Helper()

				if !reflect.DeepEqual(cfg.GroupOrder, []string{"infra", "db"}) {
					t.Fatalf("unexpected group order: %#v", cfg.GroupOrder)
				}
				if !reflect.DeepEqual(cfg.PinnedAliases, []string{"api", "worker"}) {
					t.Fatalf("unexpected pinned aliases: %#v", cfg.PinnedAliases)
				}

				api := cfg.HostByAlias("api")
				if api == nil {
					t.Fatal("expected api host to be parsed")
				}
				if api.Group != "infra" || api.Desc != "API server" || api.Auth != "key" || api.Order != 2 {
					t.Fatalf("unexpected api metadata: %#v", api)
				}
				if api.IdentityFile != "~/.ssh/id_api" {
					t.Fatalf("unexpected api identity file: %q", api.IdentityFile)
				}

				worker := cfg.HostByAlias("worker")
				if worker == nil {
					t.Fatal("expected worker host to be parsed")
				}
				if worker.Group != "Uncategorized" || worker.Auth != "password" || worker.Order != 3 {
					t.Fatalf("unexpected worker metadata: %#v", worker)
				}

				wildcard := cfg.HostByAlias("*")
				if wildcard == nil || !wildcard.IsWildcard {
					t.Fatalf("expected wildcard host to be preserved, got %#v", wildcard)
				}
			},
		},
		{
			name: "nsh metadata comment without following host stays comment",
			content: strings.Join([]string{
				"# nsh: group=infra, desc=lonely",
				"# plain comment",
				"Host plain",
				"    HostName plain.example.com",
				"",
			}, "\n"),
			check: func(t *testing.T, cfg *NSHConfig) {
				t.Helper()

				host := cfg.HostByAlias("plain")
				if host == nil {
					t.Fatal("expected plain host to be parsed")
				}
				if host.Group != "Uncategorized" {
					t.Fatalf("expected untagged host to default to Uncategorized, got %q", host.Group)
				}
				if len(cfg.Blocks) < 2 || cfg.Blocks[0].Type != BlockComment {
					t.Fatalf("expected leading nsh metadata line to remain a comment block, got %#v", cfg.Blocks)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Parse(tt.content)
			if got := Serialize(cfg); got != tt.content {
				t.Fatalf("round-trip mismatch\nwant:\n%s\n\ngot:\n%s", tt.content, got)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}
