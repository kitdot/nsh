package cmd

import (
	"fmt"

	"github.com/kitdot/nsh/core"
)

var (
	keychainGetPassword        = core.KeychainGetPassword
	keychainSetPassword        = core.KeychainSetPassword
	keychainDeletePasswordSafe = core.KeychainDeletePasswordSilent
)

type keychainMutation struct {
	name     string
	apply    func() error
	rollback func()
}

func applyConfigAndKeychainMutations(
	manager *core.NSHConfigManager,
	before *core.NSHConfig,
	after *core.NSHConfig,
	mutations []keychainMutation,
) error {
	if err := manager.Save(after); err != nil {
		return err
	}

	applied := make([]keychainMutation, 0, len(mutations))
	for _, mutation := range mutations {
		if mutation.apply == nil {
			continue
		}
		if err := mutation.apply(); err != nil {
			rollbackKeychainMutations(applied)
			if rollbackErr := manager.Save(before); rollbackErr != nil {
				if mutation.name == "" {
					return fmt.Errorf("keychain update failed: %w; config rollback failed: %v", err, rollbackErr)
				}
				return fmt.Errorf("%s: %w; config rollback failed: %v", mutation.name, err, rollbackErr)
			}
			if mutation.name == "" {
				return fmt.Errorf("keychain update failed: %w; config changes rolled back", err)
			}
			return fmt.Errorf("%s: %w; config changes rolled back", mutation.name, err)
		}
		applied = append(applied, mutation)
	}

	return nil
}

func rollbackKeychainMutations(applied []keychainMutation) {
	for i := len(applied) - 1; i >= 0; i-- {
		if applied[i].rollback != nil {
			applied[i].rollback()
		}
	}
}

func setPasswordMutation(alias, password string) keychainMutation {
	previousPassword, hadPrevious := keychainGetPassword(alias)
	return keychainMutation{
		name: fmt.Sprintf("failed to set password for '%s'", alias),
		apply: func() error {
			return keychainSetPassword(password, alias)
		},
		rollback: func() {
			if hadPrevious {
				_ = keychainSetPassword(previousPassword, alias)
				return
			}
			keychainDeletePasswordSafe(alias)
		},
	}
}

func deletePasswordMutation(alias string) keychainMutation {
	previousPassword, hadPrevious := keychainGetPassword(alias)
	return keychainMutation{
		apply: func() error {
			keychainDeletePasswordSafe(alias)
			return nil
		},
		rollback: func() {
			if hadPrevious {
				_ = keychainSetPassword(previousPassword, alias)
			}
		},
	}
}
