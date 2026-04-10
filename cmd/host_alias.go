package cmd

import (
	"fmt"

	"github.com/kitdot/nsh/bridge"
)

var promptText = bridge.PromptText

type aliasPromptOptions struct {
	Label              string
	DefaultValue       string
	Required           bool
	RequiredMessage    string
	AllowExistingAlias string
	DisallowAlias      string
	DisallowAliasMsg   string
	AliasAlreadyExists func(string) bool
}

func promptAlias(opts aliasPromptOptions) (string, bool) {
	label := opts.Label
	if label == "" {
		label = "Alias"
	}

	for {
		r := promptText(label, opts.DefaultValue)
		if r.Cancelled {
			return "", true
		}

		if opts.Required && r.Value == "" && opts.RequiredMessage != "" {
			fmt.Println(opts.RequiredMessage)
			continue
		}

		if !isValidAlias(r.Value) {
			fmt.Println("Alias may only contain letters, digits, dots, hyphens, and underscores.")
			continue
		}

		if opts.DisallowAlias != "" && r.Value == opts.DisallowAlias {
			if opts.DisallowAliasMsg == "" {
				fmt.Println("Alias is not allowed.")
			} else {
				fmt.Println(opts.DisallowAliasMsg)
			}
			continue
		}

		if opts.AliasAlreadyExists != nil && r.Value != opts.AllowExistingAlias && opts.AliasAlreadyExists(r.Value) {
			fmt.Printf("Error: Host '%s' already exists. Please choose another.\n", r.Value)
			continue
		}

		return r.Value, false
	}
}
