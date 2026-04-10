//go:build !darwin

package core

import "fmt"

// AuthenticateTouchID is not available on non-macOS platforms
func AuthenticateTouchID(reason string) error {
	return fmt.Errorf("Touch ID is only available on macOS")
}
