package core

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ExportData is the top-level export structure
type ExportData struct {
	Version    int          `json:"version"`
	ExportedAt string       `json:"exported_at"`
	GroupOrder []string     `json:"group_order,omitempty"`
	Hosts      []ExportHost `json:"hosts"`
	Secrets    *Secrets     `json:"secrets,omitempty"`
}

// ExportHost holds a single host's config
type ExportHost struct {
	Alias         string `json:"alias"`
	HostName      string `json:"hostname"`
	User          string `json:"user,omitempty"`
	Port          string `json:"port,omitempty"`
	IdentityFile  string `json:"identity_file,omitempty"`
	IdentityKeyID string `json:"identity_key_id,omitempty"`
	Group         string `json:"group,omitempty"`
	Desc          string `json:"desc,omitempty"`
	Auth          string `json:"auth,omitempty"`
	Order         int    `json:"order,omitempty"`
}

// Secrets holds sensitive data (only in full export)
type Secrets struct {
	Passwords map[string]string    `json:"passwords,omitempty"` // alias → password
	Keys      map[string]string    `json:"keys,omitempty"`      // legacy: filename → base64(pem content)
	KeyFiles  map[string]KeyExport `json:"key_files,omitempty"` // key id → exported key payload
}

// KeyExport stores one exported private key payload.
type KeyExport struct {
	FileName string `json:"file_name"`
	Content  string `json:"content"`
}

// BuildExportData builds export data from config
// If full=true, includes passwords from Keychain and PEM file contents
// If groups is non-nil, only hosts in the specified groups are exported
func BuildExportData(cfg *NSHConfig, full bool, groups []string) (*ExportData, error) {
	// Build group filter set
	var groupFilter map[string]bool
	if groups != nil {
		groupFilter = make(map[string]bool, len(groups))
		for _, g := range groups {
			groupFilter[g] = true
		}
	}

	// Filter group order to only include selected groups
	var exportGroupOrder []string
	if groupFilter != nil {
		for _, g := range cfg.GroupOrder {
			if groupFilter[g] {
				exportGroupOrder = append(exportGroupOrder, g)
			}
		}
	} else {
		exportGroupOrder = cfg.GroupOrder
	}

	data := &ExportData{
		Version:    2,
		ExportedAt: time.Now().Format(time.RFC3339),
		GroupOrder: exportGroupOrder,
	}

	var secrets *Secrets
	if full {
		secrets = &Secrets{
			Passwords: make(map[string]string),
			Keys:      make(map[string]string),
			KeyFiles:  make(map[string]KeyExport),
		}
	}

	for _, host := range cfg.Hosts() {
		if host.IsWildcard {
			continue
		}

		// Skip hosts not in selected groups
		if groupFilter != nil && !groupFilter[host.Group] {
			continue
		}

		eh := ExportHost{
			Alias:        host.Alias,
			HostName:     host.HostName,
			User:         host.User,
			Port:         host.Port,
			IdentityFile: host.IdentityFile,
			Group:        host.Group,
			Desc:         host.Desc,
			Auth:         host.Auth,
			Order:        host.Order,
		}
		if full {
			// Get password from Keychain
			if host.Auth == "password" {
				if pw, ok := KeychainGetPassword(host.Alias); ok {
					secrets.Passwords[host.Alias] = pw
				}
			}

			// Store key material by content hash so same-name files from
			// different directories do not overwrite each other.
			if host.IdentityFile != "" {
				keyPath := ExpandPath(host.IdentityFile)
				if content, err := os.ReadFile(keyPath); err == nil {
					keyID := buildExportKeyID(content)
					eh.IdentityKeyID = keyID
					if _, exists := secrets.KeyFiles[keyID]; !exists {
						secrets.KeyFiles[keyID] = KeyExport{
							FileName: filepath.Base(keyPath),
							Content:  base64.StdEncoding.EncodeToString(content),
						}
					}
				}
			}
		}

		data.Hosts = append(data.Hosts, eh)
	}

	if full && (len(secrets.Passwords) > 0 || len(secrets.Keys) > 0 || len(secrets.KeyFiles) > 0) {
		data.Secrets = secrets
	}

	return data, nil
}

func buildExportKeyID(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// ImportResult holds structured results from an import operation
type ImportResult struct {
	Logs     []string          // human-readable log messages
	AliasMap map[string]string // original alias → actual alias (after rename)
	Skipped  map[string]bool   // aliases that were skipped
	Imported map[string]bool   // actual aliases that were imported
}

// ImportHosts merges exported hosts into an existing config
// conflictAction: "skip", "overwrite", "rename"
func ImportHosts(cfg *NSHConfig, data *ExportData, conflictActions map[string]string) (*NSHConfig, *ImportResult) {
	result := &ImportResult{
		AliasMap: make(map[string]string),
		Skipped:  make(map[string]bool),
		Imported: make(map[string]bool),
	}
	newBlocks := make([]NSHBlock, len(cfg.Blocks))
	copy(newBlocks, cfg.Blocks)
	usedAliases := make(map[string]bool)
	for _, host := range cfg.Hosts() {
		if !host.IsWildcard {
			usedAliases[host.Alias] = true
		}
	}

	// Update group order
	newGroupOrder := append([]string(nil), cfg.GroupOrder...)
	if len(data.GroupOrder) > 0 {
		seen := map[string]bool{}
		for _, g := range newGroupOrder {
			seen[g] = true
		}
		for _, g := range data.GroupOrder {
			if !seen[g] {
				newGroupOrder = append(newGroupOrder, g)
				seen[g] = true
			}
		}
	}

	for _, eh := range data.Hosts {
		alias := eh.Alias
		aliasInUse := usedAliases[alias]
		existing := cfg.HostByAlias(alias)

		if aliasInUse {
			action := conflictActions[alias]
			if action == "" && existing == nil {
				action = "rename"
			}
			switch action {
			case "skip":
				result.Logs = append(result.Logs, "Skipped: "+alias)
				result.Skipped[alias] = true
				continue
			case "overwrite":
				// Remove existing block
				var filtered []NSHBlock
				for _, b := range newBlocks {
					if b.Type == BlockHost && b.Host != nil && b.Host.Alias == alias {
						continue
					}
					filtered = append(filtered, b)
				}
				newBlocks = filtered
				result.Logs = append(result.Logs, "Overwritten: "+alias)
				result.AliasMap[eh.Alias] = alias
				result.Imported[alias] = true
			case "rename":
				// Find unique name
				oldAlias := alias
				alias = nextAvailableAlias(alias, usedAliases)
				result.Logs = append(result.Logs, "Renamed: "+oldAlias+" → "+alias)
				result.AliasMap[eh.Alias] = alias
				result.Imported[alias] = true
			default:
				result.Logs = append(result.Logs, "Skipped (no action): "+alias)
				result.Skipped[alias] = true
				continue
			}
		} else {
			result.Logs = append(result.Logs, "Imported: "+alias)
			result.AliasMap[eh.Alias] = alias
			result.Imported[alias] = true
		}

		// Build new host block
		host := &NSHHost{
			Group:        eh.Group,
			Desc:         eh.Desc,
			Auth:         eh.Auth,
			Order:        eh.Order,
			Alias:        alias,
			HostName:     eh.HostName,
			User:         eh.User,
			Port:         eh.Port,
			IdentityFile: eh.IdentityFile,
		}

		var rawLines []string
		if eh.HostName != "" {
			rawLines = append(rawLines, "    HostName "+eh.HostName)
		}
		if eh.User != "" {
			rawLines = append(rawLines, "    User "+eh.User)
		}
		if eh.Port != "" && eh.Port != "22" {
			rawLines = append(rawLines, "    Port "+eh.Port)
		}
		if eh.IdentityFile != "" {
			rawLines = append(rawLines, "    IdentityFile "+eh.IdentityFile)
		}
		host.RawPropertyLines = rawLines

		nshLine := BuildNshLine(host)

		if len(newBlocks) > 0 {
			newBlocks = append(newBlocks, NSHBlock{Type: BlockBlank, Raw: ""})
		}
		newBlocks = append(newBlocks, NSHBlock{
			Type:     BlockHost,
			Host:     host,
			NshLine:  nshLine,
			HostLine: "Host " + alias,
		})
		usedAliases[alias] = true
	}

	return &NSHConfig{
		Blocks:        newBlocks,
		GroupOrder:    newGroupOrder,
		PinnedAliases: append([]string(nil), cfg.PinnedAliases...),
	}, result
}

func nextAvailableAlias(alias string, usedAliases map[string]bool) string {
	for i := 2; ; i++ {
		candidate := alias + fmt.Sprintf("_%d", i)
		if !usedAliases[candidate] {
			return candidate
		}
	}
}
