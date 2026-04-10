package cmd

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/kitdot/nsh/core"
)

func TestApplyConfigAndKeychainMutationsRollbackOnFailure(t *testing.T) {
	tempDir := t.TempDir()
	manager := core.NewConfigManager(filepath.Join(tempDir, "ssh", "config"))
	if err := manager.EnsureSetup(); err != nil {
		t.Fatalf("EnsureSetup: %v", err)
	}

	before := core.Parse("Host old\n    HostName old.example.com\n")
	after := core.Parse("Host new\n    HostName new.example.com\n")
	if err := manager.Save(before); err != nil {
		t.Fatalf("Save before: %v", err)
	}

	firstRolledBack := false
	secondRolledBack := false
	err := applyConfigAndKeychainMutations(
		manager,
		before,
		after,
		[]keychainMutation{
			{
				apply: func() error { return nil },
				rollback: func() {
					firstRolledBack = true
				},
			},
			{
				apply: func() error { return errors.New("boom") },
				rollback: func() {
					secondRolledBack = true
				},
			},
		},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !firstRolledBack {
		t.Fatal("expected already-applied mutation to be rolled back")
	}
	if secondRolledBack {
		t.Fatal("did not expect failing mutation rollback to run before apply succeeds")
	}

	loaded, loadErr := manager.Load()
	if loadErr != nil {
		t.Fatalf("Load after rollback: %v", loadErr)
	}
	if loaded.HostByAlias("old") == nil {
		t.Fatal("expected original config to be restored")
	}
	if loaded.HostByAlias("new") != nil {
		t.Fatal("did not expect new config to remain after rollback")
	}
}

func TestApplyConfigAndKeychainMutationsSuccess(t *testing.T) {
	tempDir := t.TempDir()
	manager := core.NewConfigManager(filepath.Join(tempDir, "ssh", "config"))
	if err := manager.EnsureSetup(); err != nil {
		t.Fatalf("EnsureSetup: %v", err)
	}

	before := core.Parse("Host old\n    HostName old.example.com\n")
	after := core.Parse("Host new\n    HostName new.example.com\n")
	if err := manager.Save(before); err != nil {
		t.Fatalf("Save before: %v", err)
	}

	applied := false
	rolledBack := false
	err := applyConfigAndKeychainMutations(
		manager,
		before,
		after,
		[]keychainMutation{
			{
				apply: func() error {
					applied = true
					return nil
				},
				rollback: func() {
					rolledBack = true
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("applyConfigAndKeychainMutations: %v", err)
	}
	if !applied {
		t.Fatal("expected mutation to run")
	}
	if rolledBack {
		t.Fatal("did not expect rollback on success")
	}

	loaded, loadErr := manager.Load()
	if loadErr != nil {
		t.Fatalf("Load after success: %v", loadErr)
	}
	if loaded.HostByAlias("new") == nil {
		t.Fatal("expected updated config to be saved")
	}
}
