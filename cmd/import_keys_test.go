package cmd

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kitdot/nsh/core"
)

func TestRestoreImportedKeyFilesSeparatesSameNamedKeys(t *testing.T) {
	keyDir := t.TempDir()
	first := []byte("first-private-key")
	second := []byte("second-private-key")

	data := &core.ExportData{
		Secrets: &core.Secrets{
			KeyFiles: map[string]core.KeyExport{
				"key-one": {
					FileName: "id_test",
					Content:  base64.StdEncoding.EncodeToString(first),
				},
				"key-two": {
					FileName: "id_test",
					Content:  base64.StdEncoding.EncodeToString(second),
				},
			},
		},
	}

	keyRefs, _, err := restoreImportedKeyFiles(data, keyDir)
	if err != nil {
		t.Fatalf("restoreImportedKeyFiles: %v", err)
	}
	if len(keyRefs) != 2 {
		t.Fatalf("expected 2 restored key refs, got %d", len(keyRefs))
	}
	if keyRefs["key-one"] == keyRefs["key-two"] {
		t.Fatalf("expected different key refs for different contents with same filename, got %q", keyRefs["key-one"])
	}

	firstPath := filepath.Join(keyDir, filepath.Base(keyRefs["key-one"]))
	secondPath := filepath.Join(keyDir, filepath.Base(keyRefs["key-two"]))

	firstContent, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("read first restored key: %v", err)
	}
	secondContent, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatalf("read second restored key: %v", err)
	}
	if string(firstContent) != string(first) || string(secondContent) != string(second) {
		t.Fatalf("unexpected restored key contents: %q / %q", string(firstContent), string(secondContent))
	}
}

func TestNewConfigWithMetadataUpdatesPinnedLineOnRename(t *testing.T) {
	cfg := core.Parse("# nsh-pinned: old\n\nHost old\n    HostName old.example.com\n")

	updatedPinned, changed := replacePinnedAlias(cfg.PinnedAliases, "old", "new")
	if !changed {
		t.Fatal("expected pinned aliases to change")
	}

	newConfig := newConfigWithMetadata(cfg.Blocks, cfg.GroupOrder, updatedPinned)
	if len(newConfig.PinnedAliases) != 1 || newConfig.PinnedAliases[0] != "new" {
		t.Fatalf("unexpected pinned aliases: %#v", newConfig.PinnedAliases)
	}

	if !strings.Contains(core.Serialize(newConfig), "# nsh-pinned: new") {
		t.Fatalf("expected pinned line to be rewritten, got %q", core.Serialize(newConfig))
	}
}

func TestDeleteHostWithOptionsRemovesPinnedAlias(t *testing.T) {
	tempDir := t.TempDir()
	manager := core.NewConfigManager(filepath.Join(tempDir, "ssh", "config"))
	if err := manager.EnsureSetup(); err != nil {
		t.Fatalf("EnsureSetup: %v", err)
	}

	cfg := core.Parse("# nsh-pinned: web\n\nHost web\n    HostName web.example.com\n")
	if err := manager.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := deleteHostWithOptions("web", loaded, manager, true); err != nil {
		t.Fatalf("deleteHostWithOptions: %v", err)
	}

	updated, err := manager.Load()
	if err != nil {
		t.Fatalf("Load updated: %v", err)
	}
	if updated.HostByAlias("web") != nil {
		t.Fatal("expected host to be deleted")
	}
	if len(updated.PinnedAliases) != 0 {
		t.Fatalf("expected pinned aliases to be cleared, got %#v", updated.PinnedAliases)
	}
	if strings.Contains(core.Serialize(updated), "# nsh-pinned:") {
		t.Fatalf("expected pinned line to be removed, got %q", core.Serialize(updated))
	}
}
