package cmd

import (
	"fmt"

	"github.com/kitdot/nsh/bridge"
)

func promptHostName(label, defaultValue string, required bool) (string, bool) {
	if label == "" {
		label = "HostName"
	}

	for {
		r := bridge.PromptText(label, defaultValue)
		if r.Cancelled {
			return "", true
		}
		if required && r.Value == "" {
			fmt.Println("HostName is required.")
			continue
		}
		return r.Value, false
	}
}

func promptUser(defaultValue string) (string, bool) {
	r := bridge.PromptText("User", defaultValue)
	if r.Cancelled {
		return "", true
	}
	return r.Value, false
}

func promptPort(defaultValue string) (string, bool) {
	if defaultValue == "" {
		defaultValue = "22"
	}

	r := bridge.PromptText("Port", defaultValue)
	if r.Cancelled {
		return "", true
	}
	return r.Value, false
}

func promptDescription(defaultValue string) (string, bool) {
	for {
		r := bridge.PromptText("Description", defaultValue)
		if r.Cancelled {
			return "", true
		}
		if containsComma(r.Value) {
			fmt.Println("Description cannot contain commas.")
			continue
		}
		return r.Value, false
	}
}

func promptAuthPassword(alias string, hasExisting, requireNonEmpty bool, requiredMessage string) (string, bool, bool) {
	pr := bridge.PromptPassword(fmt.Sprintf("Password for %s", alias), hasExisting)
	if pr.Cancelled {
		return "", true, false
	}
	if pr.Value != "" {
		return pr.Value, false, false
	}
	if requireNonEmpty {
		if requiredMessage == "" {
			requiredMessage = "Password is required. Cancelled."
		}
		fmt.Println(requiredMessage)
		return "", false, true
	}
	return "", false, false
}
