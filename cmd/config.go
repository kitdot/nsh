package cmd

import (
	"fmt"

	"github.com/kitdot/nsh/bridge"
	"github.com/kitdot/nsh/core"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:     "config [key] [value]",
	Aliases: []string{"conf"},
	Short:   "View or update nsh settings",
	Args:    cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		settings := core.LoadSettings()

		// No args: interactive — pick setting, then pick value
		if len(args) == 0 {
			settingOptions := []bridge.PickerOption{
				{Value: "mode", Label: "mode", Description: fmt.Sprintf("current: %s", settings.Mode)},
			}
			key := bridge.Pick("Select setting", settingOptions)
			if key == "" {
				return nil
			}
			return configInteractive(key, &settings)
		}

		key := args[0]

		// Key only: interactive picker for value
		if len(args) == 1 {
			return configInteractive(key, &settings)
		}

		// Key + value: direct set
		value := args[1]
		return configDirect(key, value, &settings)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func configInteractive(key string, settings *core.NSHSettings) error {
	switch key {
	case "mode":
		options := []bridge.PickerOption{
			{Value: "auto", Label: "auto", Description: "auto-detect (use built-in TUI)", IsCurrent: settings.Mode == "auto"},
			{Value: "fzf", Label: "fzf", Description: "interactive fuzzy search (requires fzf)", IsCurrent: settings.Mode == "fzf"},
			{Value: "list", Label: "list", Description: "built-in TUI selector", IsCurrent: settings.Mode == "list"},
		}
		selected := bridge.Pick("Select mode", options)
		if selected == "" {
			return nil
		}
		if selected == "fzf" && !bridge.IsFzfAvailable() {
			return fmt.Errorf("fzf is not installed. Install: brew install fzf")
		}
		settings.Mode = selected
		if err := settings.Save(); err != nil {
			return err
		}
		fmt.Printf("mode set to '%s'\n", selected)
	default:
		fmt.Printf("Unknown setting: %s\n", key)
		fmt.Println("Available: mode")
		return fmt.Errorf("unknown setting")
	}
	return nil
}

func configDirect(key, value string, settings *core.NSHSettings) error {
	switch key {
	case "mode":
		valid := map[string]bool{"auto": true, "fzf": true, "list": true}
		if !valid[value] {
			return fmt.Errorf("invalid mode: %s (options: auto, fzf, list)", value)
		}
		if value == "fzf" && !bridge.IsFzfAvailable() {
			return fmt.Errorf("fzf is not installed. Install: brew install fzf")
		}
		settings.Mode = value
		if err := settings.Save(); err != nil {
			return err
		}
		fmt.Printf("mode set to '%s'\n", value)
	default:
		fmt.Printf("Unknown setting: %s\n", key)
		fmt.Println("Available: mode")
		return fmt.Errorf("unknown setting")
	}
	return nil
}
