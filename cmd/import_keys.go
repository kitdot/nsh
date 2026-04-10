package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kitdot/nsh/core"
)

type importedKeyFile struct {
	FileName       string
	EncodedContent string
}

func restoreImportedKeyFiles(data *core.ExportData, keyDir string) (map[string]string, []string, error) {
	if data.Secrets == nil {
		return nil, nil, nil
	}

	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, nil, err
	}

	keyFiles := collectImportedKeyFiles(data)
	if len(keyFiles) == 0 {
		return nil, nil, nil
	}

	ids := make([]string, 0, len(keyFiles))
	for id := range keyFiles {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	keyRefs := make(map[string]string)
	var logs []string
	for _, id := range ids {
		payload := keyFiles[id]
		safeName, ok := safeImportedKeyName(payload.FileName)
		if !ok {
			logs = append(logs, fmt.Sprintf("Warning: skipping unsafe key name: %q", payload.FileName))
			continue
		}

		content, err := base64.StdEncoding.DecodeString(payload.EncodedContent)
		if err != nil {
			logs = append(logs, fmt.Sprintf("Warning: failed to decode key '%s': %v", payload.FileName, err))
			continue
		}

		keyPath, created, err := ensureImportedKeyFile(keyDir, safeName, content, id)
		if err != nil {
			logs = append(logs, fmt.Sprintf("Warning: failed to restore key '%s': %v", payload.FileName, err))
			continue
		}

		keyRefs[id] = "~/.ssh/nsh/" + filepath.Base(keyPath)
		if created {
			logs = append(logs, fmt.Sprintf("Key file restored: %s", keyPath))
		} else {
			logs = append(logs, fmt.Sprintf("Key file already exists: %s (reused)", keyPath))
		}
	}

	return keyRefs, logs, nil
}

func collectImportedKeyFiles(data *core.ExportData) map[string]importedKeyFile {
	result := make(map[string]importedKeyFile)
	if data.Secrets == nil {
		return result
	}

	for keyID, payload := range data.Secrets.KeyFiles {
		result[keyID] = importedKeyFile{
			FileName:       payload.FileName,
			EncodedContent: payload.Content,
		}
	}

	for legacyName, encodedContent := range data.Secrets.Keys {
		if _, exists := result[legacyName]; exists {
			continue
		}
		result[legacyName] = importedKeyFile{
			FileName:       legacyName,
			EncodedContent: encodedContent,
		}
	}

	return result
}

func safeImportedKeyName(name string) (string, bool) {
	safeName := filepath.Base(name)
	if safeName != name || name == "" || name == "." || name == ".." {
		return "", false
	}
	if strings.ContainsAny(name, "/\\") {
		return "", false
	}
	return safeName, true
}

func ensureImportedKeyFile(keyDir, preferredName string, content []byte, keyID string) (string, bool, error) {
	keyPath := filepath.Join(keyDir, preferredName)
	if existing, err := os.ReadFile(keyPath); err == nil {
		if bytes.Equal(existing, content) {
			return keyPath, false, nil
		}
	} else if !os.IsNotExist(err) {
		return "", false, err
	} else {
		if err := os.WriteFile(keyPath, content, 0600); err != nil {
			return "", false, err
		}
		return keyPath, true, nil
	}

	shortID := shortImportedKeyID(keyID, content)
	for attempt := 1; attempt < 1000; attempt++ {
		candidate := uniqueImportedKeyName(preferredName, shortID, attempt)
		keyPath := filepath.Join(keyDir, candidate)

		existing, err := os.ReadFile(keyPath)
		switch {
		case err == nil:
			if bytes.Equal(existing, content) {
				return keyPath, false, nil
			}
			continue
		case !os.IsNotExist(err):
			return "", false, err
		}

		if err := os.WriteFile(keyPath, content, 0600); err != nil {
			return "", false, err
		}
		return keyPath, true, nil
	}

	return "", false, fmt.Errorf("could not allocate unique filename for %q", preferredName)
}

func uniqueImportedKeyName(preferredName, shortID string, attempt int) string {
	ext := filepath.Ext(preferredName)
	base := strings.TrimSuffix(preferredName, ext)
	if base == "" {
		base = "key"
	}

	suffix := "_" + shortID
	if attempt > 1 {
		suffix = fmt.Sprintf("_%s_%d", shortID, attempt)
	}
	return base + suffix + ext
}

func shortImportedKeyID(keyID string, content []byte) string {
	if len(keyID) >= 12 {
		return keyID[:12]
	}

	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])[:12]
}

func aliasKeyRefsForImport(data *core.ExportData, result *core.ImportResult, keyRefs map[string]string) map[string]string {
	if len(keyRefs) == 0 {
		return nil
	}

	aliasRefs := make(map[string]string)
	for _, host := range data.Hosts {
		actualAlias, ok := result.AliasMap[host.Alias]
		if !ok {
			continue
		}

		keyID := host.IdentityKeyID
		if keyID == "" && host.IdentityFile != "" {
			keyID = filepath.Base(core.ExpandPath(host.IdentityFile))
		}
		if keyID == "" {
			continue
		}

		keyRef, ok := keyRefs[keyID]
		if !ok {
			continue
		}
		aliasRefs[actualAlias] = keyRef
	}

	return aliasRefs
}

func applyImportedKeyRefs(cfg *core.NSHConfig, aliasRefs map[string]string) *core.NSHConfig {
	if len(aliasRefs) == 0 {
		return cfg
	}

	newBlocks := make([]core.NSHBlock, len(cfg.Blocks))
	copy(newBlocks, cfg.Blocks)

	for i, block := range newBlocks {
		if block.Type != core.BlockHost || block.Host == nil {
			continue
		}

		newKeyRef, ok := aliasRefs[block.Host.Alias]
		if !ok {
			continue
		}

		host := *block.Host
		host.IdentityFile = newKeyRef
		host.RawPropertyLines = rewriteIdentityFileLines(host.RawPropertyLines, newKeyRef)

		newBlocks[i] = core.NSHBlock{
			Type:     core.BlockHost,
			Host:     &host,
			NshLine:  core.BuildNshLine(&host),
			HostLine: block.HostLine,
		}
	}

	return newConfigWithMetadata(newBlocks, cfg.GroupOrder, cfg.PinnedAliases)
}
