package cmd

import "github.com/kitdot/nsh/bridge"

var pickOption = bridge.Pick

type groupPromptOptions struct {
	ExistingGroups []string
	CurrentGroup   string
	MarkCurrent    bool
	NoneValue      string
	EscValue       string
}

func promptGroupSelection(opts groupPromptOptions) string {
	var groups []string
	for _, g := range opts.ExistingGroups {
		if g != "Uncategorized" {
			groups = append(groups, g)
		}
	}

	options := []bridge.PickerOption{
		{Value: "__none__", Label: "None", IsCurrent: opts.MarkCurrent && opts.CurrentGroup == "Uncategorized"},
		{Value: "__new__", Label: "New group", Description: "enter a group name"},
	}
	if len(groups) > 0 {
		options = append(options, bridge.PickerOption{IsSeparator: true, Value: "__divider__"})
		for _, g := range groups {
			options = append(options, bridge.PickerOption{
				Value:     g,
				Label:     g,
				IsCurrent: opts.MarkCurrent && g == opts.CurrentGroup,
			})
		}
	}

	for {
		selected := pickOption("Select group", options)
		switch selected {
		case "":
			if opts.EscValue != "" {
				return opts.EscValue
			}
			continue
		case "__none__":
			return opts.NoneValue
		case "__new__":
			name := promptNewGroupName()
			if name == "" {
				continue
			}
			return name
		default:
			return selected
		}
	}
}
