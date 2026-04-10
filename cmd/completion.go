package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kitdot/nsh/bridge"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Manage shell completions (install/uninstall)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Step 1: Pick action
		actionOptions := []bridge.PickerOption{
			{Value: "install", Label: "Install", Description: "install shell completions"},
			{Value: "uninstall", Label: "Uninstall", Description: "remove shell completions"},
		}
		action := bridge.Pick("Shell completions", actionOptions)
		if action == "" {
			return nil
		}

		// Step 2: Pick shell
		currentShell := detectShell()
		shellOptions := []bridge.PickerOption{
			{Value: "zsh", Label: "zsh", Description: "~/.zsh/completions/_nsh", IsCurrent: currentShell == "zsh"},
			{Value: "bash", Label: "bash", Description: "~/.local/share/bash-completion/completions/nsh", IsCurrent: currentShell == "bash"},
			{Value: "fish", Label: "fish", Description: "~/.config/fish/completions/nsh.fish", IsCurrent: currentShell == "fish"},
			{Value: "powershell", Label: "powershell", Description: "manual setup (output to stdout)"},
		}
		shell := bridge.Pick("Select shell", shellOptions)
		if shell == "" {
			return nil
		}

		// Step 3: Execute
		switch action {
		case "install":
			return doInstall(shell)
		case "uninstall":
			return doUninstall(shell)
		}
		return nil
	},
}

func init() {
	// Keep raw output subcommands for manual/scripting use
	completionCmd.AddCommand(&cobra.Command{
		Use:    "zsh",
		Short:  "Generate zsh completion script (stdout)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.GenZshCompletion(os.Stdout)
		},
	})
	completionCmd.AddCommand(&cobra.Command{
		Use:    "bash",
		Short:  "Generate bash completion script (stdout)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.GenBashCompletionV2(os.Stdout, true)
		},
	})
	completionCmd.AddCommand(&cobra.Command{
		Use:    "fish",
		Short:  "Generate fish completion script (stdout)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.GenFishCompletion(os.Stdout, true)
		},
	})
	completionCmd.AddCommand(&cobra.Command{
		Use:    "powershell",
		Short:  "Generate powershell completion script (stdout)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		},
	})

	rootCmd.AddCommand(completionCmd)
}

// --- Install ---

func doInstall(shell string) error {
	switch shell {
	case "zsh":
		return installZsh()
	case "bash":
		return installBash()
	case "fish":
		return installFish()
	case "powershell":
		return installPowershell()
	}
	return nil
}

func installZsh() error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".zsh", "completions")
	path := filepath.Join(dir, "_nsh")

	fmt.Println()
	fmt.Println("Steps:")
	fmt.Printf("  1. Create directory:  %s\n", dir)
	fmt.Printf("  2. Write completion:  %s\n", path)
	fmt.Println("  3. Check ~/.zshrc for fpath setting")
	fmt.Println()

	confirm := bridge.Pick("Proceed?", []bridge.PickerOption{
		{Value: "yes", Label: "Yes"},
		{Value: "no", Label: "No"},
	})
	if confirm != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer f.Close()

	if err := rootCmd.GenZshCompletion(f); err != nil {
		return fmt.Errorf("failed to generate completion: %w", err)
	}
	fmt.Printf("Wrote: %s\n", path)

	zshrc := filepath.Join(home, ".zshrc")
	content, err := os.ReadFile(zshrc)
	if err == nil && !strings.Contains(string(content), ".zsh/completions") {
		fmt.Println()
		fmt.Println("Add the following to your ~/.zshrc:")
		fmt.Println()
		fmt.Println("  fpath=(~/.zsh/completions $fpath)")
		fmt.Println("  autoload -Uz compinit && compinit")
	}

	fmt.Println()
	fmt.Println("Run to apply:  source ~/.zshrc")
	fmt.Println()
	fmt.Println("Done.")
	return nil
}

func installBash() error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".local", "share", "bash-completion", "completions")
	path := filepath.Join(dir, "nsh")

	fmt.Println()
	fmt.Println("Steps:")
	fmt.Printf("  1. Create directory:  %s\n", dir)
	fmt.Printf("  2. Write completion:  %s\n", path)
	fmt.Println()
	fmt.Println("Note: Requires bash-completion (brew install bash-completion@2)")
	fmt.Println()

	confirm := bridge.Pick("Proceed?", []bridge.PickerOption{
		{Value: "yes", Label: "Yes"},
		{Value: "no", Label: "No"},
	})
	if confirm != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer f.Close()

	if err := rootCmd.GenBashCompletionV2(f, true); err != nil {
		return fmt.Errorf("failed to generate completion: %w", err)
	}

	fmt.Printf("Wrote: %s\n", path)
	fmt.Println()
	fmt.Println("Restart your terminal to apply.")
	fmt.Println()
	fmt.Println("Done.")
	return nil
}

func installFish() error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "fish", "completions")
	path := filepath.Join(dir, "nsh.fish")

	fmt.Println()
	fmt.Println("Steps:")
	fmt.Printf("  1. Create directory:  %s\n", dir)
	fmt.Printf("  2. Write completion:  %s\n", path)
	fmt.Println()

	confirm := bridge.Pick("Proceed?", []bridge.PickerOption{
		{Value: "yes", Label: "Yes"},
		{Value: "no", Label: "No"},
	})
	if confirm != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer f.Close()

	if err := rootCmd.GenFishCompletion(f, true); err != nil {
		return fmt.Errorf("failed to generate completion: %w", err)
	}

	fmt.Printf("Wrote: %s\n", path)
	fmt.Println()
	fmt.Println("Fish completions are auto-loaded. Open a new terminal to apply.")
	fmt.Println()
	fmt.Println("Done.")
	return nil
}

func installPowershell() error {
	fmt.Println()
	fmt.Println("PowerShell does not support auto-install.")
	fmt.Println()
	fmt.Println("To set up manually, add to your $PROFILE:")
	fmt.Println()
	fmt.Println("  nsh completion powershell | Out-String | Invoke-Expression")
	fmt.Println()
	fmt.Println("Or generate the script:")
	fmt.Println()
	fmt.Println("  nsh completion powershell > nsh.ps1")
	fmt.Println()
	return nil
}

// --- Uninstall ---

func doUninstall(shell string) error {
	home, _ := os.UserHomeDir()
	var path string

	switch shell {
	case "zsh":
		path = filepath.Join(home, ".zsh", "completions", "_nsh")
	case "bash":
		path = filepath.Join(home, ".local", "share", "bash-completion", "completions", "nsh")
	case "fish":
		path = filepath.Join(home, ".config", "fish", "completions", "nsh.fish")
	case "powershell":
		fmt.Println()
		fmt.Println("To remove PowerShell completions, edit your $PROFILE and remove the nsh line.")
		fmt.Println()
		return nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("No %s completion found at %s\n", shell, path)
		return nil
	}

	fmt.Println()
	fmt.Printf("Will remove: %s\n", path)
	fmt.Println()

	confirm := bridge.Pick("Proceed?", []bridge.PickerOption{
		{Value: "yes", Label: "Yes"},
		{Value: "no", Label: "No"},
	})
	if confirm != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove: %w", err)
	}
	fmt.Printf("Removed %s completion.\n", shell)

	switch shell {
	case "zsh":
		fmt.Println()
		fmt.Println("Run to apply:  source ~/.zshrc")
	case "bash":
		fmt.Println()
		fmt.Println("Restart your terminal to apply.")
	case "fish":
		fmt.Println()
		fmt.Println("Open a new terminal to apply.")
	}

	fmt.Println()
	fmt.Println("Done.")
	return nil
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return "zsh"
	}
	if strings.Contains(shell, "bash") {
		return "bash"
	}
	if strings.Contains(shell, "fish") {
		return "fish"
	}
	return ""
}
