package core

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildExportDataUsesKeyIDsForSameNamedKeys(t *testing.T) {
	tempDir := t.TempDir()
	keyOne := filepath.Join(tempDir, "first", "id_test")
	keyTwo := filepath.Join(tempDir, "second", "id_test")

	if err := os.MkdirAll(filepath.Dir(keyOne), 0755); err != nil {
		t.Fatalf("mkdir first: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyTwo), 0755); err != nil {
		t.Fatalf("mkdir second: %v", err)
	}
	if err := os.WriteFile(keyOne, []byte("first-key"), 0600); err != nil {
		t.Fatalf("write first key: %v", err)
	}
	if err := os.WriteFile(keyTwo, []byte("second-key"), 0600); err != nil {
		t.Fatalf("write second key: %v", err)
	}

	cfg := &NSHConfig{
		Blocks: []NSHBlock{
			{
				Type: BlockHost,
				Host: &NSHHost{
					Alias:        "alpha",
					HostName:     "alpha.example.com",
					IdentityFile: keyOne,
				},
				HostLine: "Host alpha",
			},
			{
				Type: BlockHost,
				Host: &NSHHost{
					Alias:        "beta",
					HostName:     "beta.example.com",
					IdentityFile: keyTwo,
				},
				HostLine: "Host beta",
			},
		},
	}

	data, err := BuildExportData(cfg, true, nil)
	if err != nil {
		t.Fatalf("BuildExportData: %v", err)
	}
	if data.Secrets == nil {
		t.Fatal("expected secrets in full export")
	}
	if len(data.Secrets.KeyFiles) != 2 {
		t.Fatalf("expected 2 exported key payloads, got %d", len(data.Secrets.KeyFiles))
	}
	if data.Hosts[0].IdentityKeyID == "" || data.Hosts[1].IdentityKeyID == "" {
		t.Fatal("expected exported hosts to include identity_key_id")
	}
	if data.Hosts[0].IdentityKeyID == data.Hosts[1].IdentityKeyID {
		t.Fatal("expected same-named keys with different content to get different key ids")
	}

	firstPayload := data.Secrets.KeyFiles[data.Hosts[0].IdentityKeyID]
	secondPayload := data.Secrets.KeyFiles[data.Hosts[1].IdentityKeyID]
	if firstPayload.FileName != "id_test" || secondPayload.FileName != "id_test" {
		t.Fatalf("expected original file names to be preserved, got %q and %q", firstPayload.FileName, secondPayload.FileName)
	}

	firstContent, err := base64.StdEncoding.DecodeString(firstPayload.Content)
	if err != nil {
		t.Fatalf("decode first payload: %v", err)
	}
	secondContent, err := base64.StdEncoding.DecodeString(secondPayload.Content)
	if err != nil {
		t.Fatalf("decode second payload: %v", err)
	}
	if string(firstContent) != "first-key" || string(secondContent) != "second-key" {
		t.Fatalf("unexpected key payloads: %q / %q", string(firstContent), string(secondContent))
	}
}

func TestImportHostsRenameAvoidsBatchAliasCollisions(t *testing.T) {
	cfg := &NSHConfig{
		Blocks: []NSHBlock{
			{
				Type: BlockHost,
				Host: &NSHHost{
					Alias:    "web",
					HostName: "existing.example.com",
				},
				HostLine: "Host web",
			},
		},
	}

	data := &ExportData{
		Hosts: []ExportHost{
			{Alias: "web", HostName: "import-one.example.com"},
			{Alias: "web", HostName: "import-two.example.com"},
		},
	}

	newConfig, _ := ImportHosts(cfg, data, map[string]string{"web": "rename"})

	if newConfig.HostByAlias("web") == nil {
		t.Fatal("expected existing alias to remain")
	}
	if newConfig.HostByAlias("web_2") == nil {
		t.Fatal("expected first imported host to be renamed to web_2")
	}
	if newConfig.HostByAlias("web_3") == nil {
		t.Fatal("expected second imported host to avoid colliding with web_2")
	}
}
