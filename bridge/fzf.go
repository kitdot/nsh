package bridge

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kitdot/nsh/core"
)

// IsFzfAvailable checks if fzf is installed
func IsFzfAvailable() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

// ANSI color codes
const (
	cCyan   = "\033[36m"
	cGreen  = "\033[32m"
	cYellow = "\033[33m"
	cDim    = "\033[2m"
	cBold   = "\033[1m"
	cReset  = "\033[0m"
)

const (
	iconDefault = "○"
	iconKey     = "◆"
	iconPass    = "●"
)

// FzfSelectHost launches fzf for host selection
func FzfSelectHost(hosts []core.NSHHost, nshBinary string, groupOrder []string) string {
	filtered := filterWildcard(hosts)
	if len(filtered) == 0 {
		return ""
	}

	sorted := sortHosts(filtered, groupOrder)

	// Compute max widths for alignment
	maxGroup, maxAlias, maxHost := 1, 10, 15
	for _, h := range sorted {
		gl := 1
		if h.Group != "Uncategorized" {
			gl = len(h.Group) + 2
		}
		if gl > maxGroup {
			maxGroup = gl
		}
		if len(h.Alias) > maxAlias {
			maxAlias = len(h.Alias)
		}
		if len(h.HostName) > maxHost {
			maxHost = len(h.HostName)
		}
	}

	var lines []string
	for _, h := range sorted {
		groupStr := fmt.Sprintf("%s%s%s", cDim, pad("-", maxGroup), cReset)
		if h.Group != "Uncategorized" {
			tag := fmt.Sprintf("[%s]", h.Group)
			groupStr = fmt.Sprintf("%s%s%s", cCyan, pad(tag, maxGroup), cReset)
		}

		icon := fmt.Sprintf("%s%s%s", cDim, iconDefault, cReset)
		switch h.Auth {
		case "password":
			icon = fmt.Sprintf("%s%s%s", cYellow, iconPass, cReset)
		case "key":
			icon = fmt.Sprintf("%s%s%s", cGreen, iconKey, cReset)
		}

		aliasStr := fmt.Sprintf("%s%s%s", cBold, pad(h.Alias, maxAlias), cReset)
		hostStr := fmt.Sprintf("%s%s%s", cYellow, pad(h.HostName, maxHost), cReset)
		descStr := ""
		if h.Desc != "" {
			descStr = fmt.Sprintf("%s- %s%s", cDim, h.Desc, cReset)
		}

		lines = append(lines, fmt.Sprintf("%s\t%s  %s  %s  %s  %s", h.Alias, groupStr, icon, aliasStr, hostStr, descStr))
	}

	input := strings.Join(lines, "\n")

	// Write to temp file for fzf input
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("nsh_fzf_%d", os.Getpid()))
	defer os.Remove(tmpFile)

	escaped := strings.ReplaceAll(input, "'", "'\\''")
	cmdStr := fmt.Sprintf("printf '%%s' '%s' | fzf --ansi --delimiter='\\t' --with-nth=2 --preview '%s show {1}' --preview-window right:50%% --header 'Select SSH host (ESC to cancel)' --layout reverse --border | cut -f1 > %s",
		escaped, nshBinary, tmpFile)

	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return ""
	}

	result, err := os.ReadFile(tmpFile)
	if err != nil {
		return ""
	}

	selected := strings.TrimSpace(string(result))
	return selected
}

// CurrentBinaryPath returns the path of the current nsh executable
func CurrentBinaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "nsh"
	}
	return exe
}

func filterWildcard(hosts []core.NSHHost) []core.NSHHost {
	var result []core.NSHHost
	for _, h := range hosts {
		if !h.IsWildcard {
			result = append(result, h)
		}
	}
	return result
}

func sortHosts(hosts []core.NSHHost, groupOrder []string) []core.NSHHost {
	groupRank := map[string]int{}
	for i, g := range groupOrder {
		groupRank[g] = i
	}
	maxRank := len(groupOrder)

	sorted := make([]core.NSHHost, len(hosts))
	copy(sorted, hosts)

	// Insertion sort for stability
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0; j-- {
			a, b := sorted[j], sorted[j-1]
			aRank := getRank(a.Group, groupRank, maxRank)
			bRank := getRank(b.Group, groupRank, maxRank)

			swap := false
			if aRank != bRank {
				swap = aRank < bRank
			} else if a.Group != b.Group {
				swap = a.Group < b.Group
			} else if a.Order != b.Order {
				swap = a.Order < b.Order
			} else {
				swap = a.Alias < b.Alias
			}

			if swap {
				sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
			} else {
				break
			}
		}
	}
	return sorted
}

func getRank(group string, groupRank map[string]int, maxRank int) int {
	if group == "Uncategorized" {
		return 1<<31 - 1
	}
	if r, ok := groupRank[group]; ok {
		return r
	}
	return maxRank
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
