package cmd

import (
	"fmt"
	"strings"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/core"
)

func selectAuthMethod(currentAuth string) (string, bool) {
	selected := bridge.Pick("Authentication method", authMethodOptions(currentAuth))
	if selected == "" {
		return "", true
	}
	if selected == "none" {
		return "", false
	}
	return selected, false
}

func authMethodOptions(currentAuth string) []bridge.PickerOption {
	return []bridge.PickerOption{
		{Value: "none", Label: "None", Description: "default SSH behavior", IsCurrent: currentAuth == ""},
		{Value: "password", Label: "Password", Description: "stored in macOS Keychain", IsCurrent: currentAuth == "password"},
		{Value: "key", Label: "Private key", Description: "auto ssh-add", IsCurrent: currentAuth == "key"},
	}
}

func promptIdentityFile(currentIdentityFile string) (string, bool) {
	if currentIdentityFile != "" {
		fmt.Printf("Current IdentityFile: %s\n", currentIdentityFile)
		pick := bridge.Pick("IdentityFile", []bridge.PickerOption{
			{Value: "__keep__", Label: fmt.Sprintf("Keep current (%s)", currentIdentityFile)},
			{Value: "__change__", Label: "Change IdentityFile"},
		})
		if pick == "" {
			return "", true
		}
		if pick == "__keep__" {
			return currentIdentityFile, false
		}
	}

	selected := selectSSHKey()
	if selected == "" {
		selected = bridge.PromptPath("IdentityFile path (tab to complete):")
		if selected == "" {
			fmt.Println("Private key auth requires an IdentityFile.")
			return "", true
		}
	}
	return selected, false
}

func buildManagedRawPropertyLines(host *core.NSHHost) []string {
	rawLines := []string{
		fmt.Sprintf("    HostName %s", host.HostName),
		fmt.Sprintf("    User %s", host.User),
	}
	if host.Port != "22" {
		rawLines = append(rawLines, fmt.Sprintf("    Port %s", host.Port))
	}
	if host.IdentityFile != "" {
		rawLines = append(rawLines, fmt.Sprintf("    IdentityFile %s", host.IdentityFile))
	}
	return rawLines
}

func buildHostBlock(host *core.NSHHost, hostLine string) core.NSHBlock {
	hostCopy := *host
	hostCopy.RawPropertyLines = append([]string(nil), host.RawPropertyLines...)
	if hostLine == "" {
		hostLine = fmt.Sprintf("Host %s", hostCopy.Alias)
	}
	return core.NSHBlock{
		Type:     core.BlockHost,
		Host:     &hostCopy,
		NshLine:  core.BuildNshLine(&hostCopy),
		HostLine: hostLine,
	}
}

func buildManagedHostBlock(host *core.NSHHost) core.NSHBlock {
	hostCopy := *host
	hostCopy.RawPropertyLines = buildManagedRawPropertyLines(&hostCopy)
	return buildHostBlock(&hostCopy, "")
}

func appendManagedHostBlock(blocks []core.NSHBlock, host *core.NSHHost) []core.NSHBlock {
	newBlocks := make([]core.NSHBlock, len(blocks))
	copy(newBlocks, blocks)
	if len(newBlocks) > 0 {
		newBlocks = append(newBlocks, core.NSHBlock{Type: core.BlockBlank, Raw: ""})
	}
	return append(newBlocks, buildManagedHostBlock(host))
}

func replaceManagedHostBlock(blocks []core.NSHBlock, alias string, host *core.NSHHost) []core.NSHBlock {
	newBlocks := make([]core.NSHBlock, len(blocks))
	copy(newBlocks, blocks)
	replacement := buildManagedHostBlock(host)
	for i, block := range newBlocks {
		if block.Type == core.BlockHost && block.Host != nil && block.Host.Alias == alias {
			newBlocks[i] = replacement
			break
		}
	}
	return newBlocks
}

func rewriteIdentityFileLines(lines []string, newIdentityFile string) []string {
	var updated []string
	found := false
	for _, line := range lines {
		trimmed := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(trimmed, "identityfile ") || strings.HasPrefix(trimmed, "identityfile\t") {
			found = true
			if newIdentityFile != "" {
				updated = append(updated, "    IdentityFile "+newIdentityFile)
			}
			continue
		}
		updated = append(updated, line)
	}
	if !found && newIdentityFile != "" {
		updated = append(updated, "    IdentityFile "+newIdentityFile)
	}
	return updated
}

func printHostPreview(host *core.NSHHost) {
	block := buildManagedHostBlock(host)

	fmt.Println()
	fmt.Println("── Preview ──────────────────")
	if block.NshLine != "" {
		fmt.Println(block.NshLine)
	}
	fmt.Println(block.HostLine)
	for _, line := range block.Host.RawPropertyLines {
		fmt.Println(line)
	}
	fmt.Println("─────────────────────────────")
	fmt.Println()
}

func confirmSave(title string) bool {
	if title == "" {
		title = "Confirm"
	}
	return bridge.Pick(title, []bridge.PickerOption{
		{Value: "save", Label: "Save"},
		{Value: "cancel", Label: "Cancel"},
	}) == "save"
}
