package cmd

import (
	"path/filepath"
	"testing"

	"github.com/kitdot/nsh/core"
)

type fakeKeychain struct {
	passwords map[string]string
}

func TestEditKeychainMigratePasswordOnAliasRename(t *testing.T) {
	withFakeKeychain(t, map[string]string{"old": "old-secret"}, func(kc *fakeKeychain) {
		mutations := buildEditKeychainMutations(
			"old",
			"new",
			"password",
			"password",
			"",
			"old-secret",
			true,
		)

		before := core.Parse("Host old\n    HostName old.example.com\n")
		after := core.Parse("Host new\n    HostName new.example.com\n")
		mustApplyMutations(t, before, after, mutations)

		if _, ok := kc.passwords["old"]; ok {
			t.Fatal("expected old alias password to be removed")
		}
		if got := kc.passwords["new"]; got != "old-secret" {
			t.Fatalf("expected migrated password under new alias, got %q", got)
		}
	})
}

func TestEditKeychainRenameWithNewPasswordOverridesExisting(t *testing.T) {
	withFakeKeychain(t, map[string]string{"old": "old-secret"}, func(kc *fakeKeychain) {
		mutations := buildEditKeychainMutations(
			"old",
			"new",
			"password",
			"password",
			"new-secret",
			"old-secret",
			true,
		)

		before := core.Parse("Host old\n    HostName old.example.com\n")
		after := core.Parse("Host new\n    HostName new.example.com\n")
		mustApplyMutations(t, before, after, mutations)

		if _, ok := kc.passwords["old"]; ok {
			t.Fatal("expected old alias password to be removed")
		}
		if got := kc.passwords["new"]; got != "new-secret" {
			t.Fatalf("expected new password under new alias, got %q", got)
		}
	})
}

func TestAuthKeychainSwitchAwayFromPasswordDeletesSecret(t *testing.T) {
	withFakeKeychain(t, map[string]string{"app": "secret"}, func(kc *fakeKeychain) {
		mutations := buildAuthKeychainMutations("app", "password", "key", "")

		before := core.Parse("Host app\n    HostName app.example.com\n")
		after := core.Parse("Host app\n    HostName app.example.com\n")
		mustApplyMutations(t, before, after, mutations)

		if _, ok := kc.passwords["app"]; ok {
			t.Fatal("expected password to be removed after switching away from password auth")
		}
	})
}

func mustApplyMutations(t *testing.T, before, after *core.NSHConfig, mutations []keychainMutation) {
	t.Helper()

	manager := newTestManager(t)
	if err := manager.Save(before); err != nil {
		t.Fatalf("Save before: %v", err)
	}
	if err := applyConfigAndKeychainMutations(manager, before, after, mutations); err != nil {
		t.Fatalf("applyConfigAndKeychainMutations: %v", err)
	}
}

func newTestManager(t *testing.T) *core.NSHConfigManager {
	t.Helper()

	tempDir := t.TempDir()
	manager := core.NewConfigManager(filepath.Join(tempDir, "ssh", "config"))
	if err := manager.EnsureSetup(); err != nil {
		t.Fatalf("EnsureSetup: %v", err)
	}
	return manager
}

func withFakeKeychain(t *testing.T, initial map[string]string, fn func(*fakeKeychain)) {
	t.Helper()

	origGet := keychainGetPassword
	origSet := keychainSetPassword
	origDelete := keychainDeletePasswordSafe
	t.Cleanup(func() {
		keychainGetPassword = origGet
		keychainSetPassword = origSet
		keychainDeletePasswordSafe = origDelete
	})

	kc := &fakeKeychain{passwords: cloneStringMap(initial)}

	keychainGetPassword = func(alias string) (string, bool) {
		pw, ok := kc.passwords[alias]
		return pw, ok
	}
	keychainSetPassword = func(password, alias string) error {
		kc.passwords[alias] = password
		return nil
	}
	keychainDeletePasswordSafe = func(alias string) {
		delete(kc.passwords, alias)
	}

	fn(kc)
}

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
