package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/connect"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

// nshVersion is set at build time via -ldflags "-X github.com/kitdot/nsh/cmd.nshVersion=..."
// Falls back to "dev" if not set.
var nshVersion = "dev"

// ANSI — Claude's warm red, bold
const (
	hlColor = "\033[1;38;2;217;119;88m" // bold #D97758
	hlReset = "\033[0m"
)

// aliasMap maps command name → alias for highlight
var aliasMap = map[string]string{
	"conn":       "c",
	"pin":        "p",
	"new":        "n",
	"copy":       "cp",
	"edit":       "e",
	"del":        "d",
	"order":      "o",
	"auth":       "au",
	"export":     "exp",
	"import":     "imp",
	"list":       "l",
	"show":       "s",
	"config":     "conf",
	"completion": "",
	"help":       "h",
}

// cmdOrder defines the display order in help
var cmdOrder = []string{
	"conn", "pin", "new", "copy", "edit", "del", "order", "auth",
	"export", "import",
	"list", "show", "config", "completion", "help",
}

var (
	sshConfigFlag string
	versionFlag   bool
)

var rootCmd = &cobra.Command{
	Use:   "nsh",
	Short: "SSH connection manager for macOS & Ghostty",
	Long:  "nsh manages SSH hosts in ~/.ssh/nsh/config with a tag protocol for organizing and authenticating hosts.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if versionFlag {
			fmt.Println(nshVersion)
			return nil
		}

		manager := core.NewConfigManager(sshConfigFlag)
		cfg, err := manager.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		for {
			result := bridge.TreeBrowser("nsh", cfg)

			// Save pinned changes if modified
			if !slicesEqual(result.PinnedAliases, cfg.PinnedAliases) {
				if err := savePinnedAliases(cfg, manager, result.PinnedAliases); err != nil {
					return fmt.Errorf("failed to save pinned hosts: %w", err)
				}
				cfg, err = manager.Load()
				if err != nil {
					return fmt.Errorf("failed to reload config: %w", err)
				}
			}

			switch result.Action {
			case "connect":
				host := cfg.HostByAlias(result.Alias)
				if host == nil {
					continue
				}
				connect.Exec(host)
				return nil

			case "edit":
				editCmd.SetArgs([]string{result.Alias})
				return editCmd.RunE(editCmd, []string{result.Alias})

			case "delete":
				err := deleteHostWithOptions(result.Alias, cfg, manager, true)
				if err != nil {
					return err
				}
				cfg, err = manager.Load()
				if err != nil {
					return err
				}
				continue

			case "new":
				return addCmd.RunE(addCmd, nil)

			default:
				return nil
			}
		}
	},
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVar(&sshConfigFlag, "ssh-config", "~/.ssh/config", "Path to SSH config file")
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Show the version")

	// Custom help function
	rootCmd.SetHelpFunc(customHelp)

	// Add "h" alias for help command (Cobra's built-in help doesn't support aliases)
	rootCmd.AddCommand(&cobra.Command{
		Use:    "h",
		Short:  "Help about any command",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			customHelp(rootCmd, args)
		},
	})

}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// colorizeCommandName highlights alias characters in a command name
func colorizeCommandName(name, alias string) string {
	if alias == "" {
		return name
	}

	var result strings.Builder
	ai := 0
	for _, ch := range name {
		if ai < len(alias) && ch == rune(alias[ai]) {
			result.WriteString(hlColor)
			result.WriteRune(ch)
			result.WriteString(hlReset)
			ai++
		} else {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func savePinnedAliases(cfg *core.NSHConfig, manager *core.NSHConfigManager, pinned []string) error {
	newConfig := newConfigWithMetadata(cfg.Blocks, cfg.GroupOrder, pinned)
	return manager.Save(newConfig)
}

func customHelp(cmd *cobra.Command, args []string) {
	if cmd != rootCmd {
		cmd.Usage()
		return
	}

	fmt.Println(cmd.Long)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s [flags]\n", cmd.Use)
	fmt.Printf("  %s [command]\n", cmd.Use)
	fmt.Println()

	fmt.Println("Available Commands:")

	// Build command map for lookup
	cmdMap := map[string]*cobra.Command{}
	for _, sub := range cmd.Commands() {
		cmdMap[sub.Name()] = sub
	}

	// Calculate max name length
	maxLen := 0
	for _, name := range cmdOrder {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}

	// Print in specified order
	for _, name := range cmdOrder {
		sub, ok := cmdMap[name]
		if !ok {
			continue
		}

		alias := aliasMap[name]
		colored := colorizeCommandName(name, alias)

		padding := strings.Repeat(" ", maxLen-len(name)+2)
		fmt.Printf("  %s%s%s\n", colored, padding, sub.Short)
	}

	fmt.Println()
	fmt.Println("Flags:")
	fmt.Print(cmd.LocalFlags().FlagUsages())
	fmt.Println()
	fmt.Printf("Use \"%s [command] --help\" for more information about a command.\n", cmd.Use)
}
