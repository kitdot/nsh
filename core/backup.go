package core

import (
	"fmt"
	"io"
	"os"
)

// Backup creates rotating backups (.nsh.bak, .nsh.bak.1, .nsh.bak.2)
func Backup(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	bak0 := filePath + ".nsh.bak"
	bak1 := filePath + ".nsh.bak.1"
	bak2 := filePath + ".nsh.bak.2"

	// Rotate: bak.1 → bak.2, bak → bak.1
	os.Remove(bak2)

	if _, err := os.Stat(bak1); err == nil {
		os.Rename(bak1, bak2)
	}

	if _, err := os.Stat(bak0); err == nil {
		os.Rename(bak0, bak1)
	}

	// Copy current file → bak
	return copyFile(filePath, bak0)
}

// ListBackups returns existing backup paths
func ListBackups(filePath string) []string {
	suffixes := []string{".nsh.bak", ".nsh.bak.1", ".nsh.bak.2"}
	var result []string
	for _, s := range suffixes {
		p := filePath + s
		if _, err := os.Stat(p); err == nil {
			result = append(result, p)
		}
	}
	return result
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}
	return nil
}
