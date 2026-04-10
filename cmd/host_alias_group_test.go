package cmd

import (
	"testing"

	"github.com/kitdot/nsh/bridge"
)

func TestPromptAlias(t *testing.T) {
	tests := []struct {
		name           string
		responses      []bridge.TextPromptResult
		opts           aliasPromptOptions
		wantAlias      string
		wantCancelled  bool
		wantPromptRuns int
	}{
		{
			name: "required retries on empty",
			responses: []bridge.TextPromptResult{
				{Value: ""},
				{Value: "app"},
			},
			opts: aliasPromptOptions{
				Label:           "Alias",
				Required:        true,
				RequiredMessage: "Alias is required.",
				AliasAlreadyExists: func(string) bool {
					return false
				},
			},
			wantAlias:      "app",
			wantPromptRuns: 2,
		},
		{
			name: "retries on invalid and duplicate",
			responses: []bridge.TextPromptResult{
				{Value: "bad alias"},
				{Value: "dup"},
				{Value: "ok"},
			},
			opts: aliasPromptOptions{
				Label:    "Alias",
				Required: true,
				AliasAlreadyExists: func(v string) bool {
					return v == "dup"
				},
			},
			wantAlias:      "ok",
			wantPromptRuns: 3,
		},
		{
			name: "allow existing alias",
			responses: []bridge.TextPromptResult{
				{Value: "old"},
			},
			opts: aliasPromptOptions{
				Label:              "Alias",
				Required:           true,
				AllowExistingAlias: "old",
				AliasAlreadyExists: func(string) bool {
					return true
				},
			},
			wantAlias:      "old",
			wantPromptRuns: 1,
		},
		{
			name: "disallow alias",
			responses: []bridge.TextPromptResult{
				{Value: "source"},
				{Value: "target"},
			},
			opts: aliasPromptOptions{
				Label:            "Alias",
				Required:         true,
				DisallowAlias:    "source",
				DisallowAliasMsg: "not allowed",
			},
			wantAlias:      "target",
			wantPromptRuns: 2,
		},
		{
			name: "cancelled",
			responses: []bridge.TextPromptResult{
				{Cancelled: true},
			},
			opts: aliasPromptOptions{
				Label:    "Alias",
				Required: true,
			},
			wantCancelled:  true,
			wantPromptRuns: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origPrompt := promptText
			t.Cleanup(func() {
				promptText = origPrompt
			})

			calls := 0
			promptText = func(label, defaultValue string) bridge.TextPromptResult {
				if calls >= len(tt.responses) {
					t.Fatalf("unexpected extra prompt call #%d", calls+1)
				}
				r := tt.responses[calls]
				calls++
				return r
			}

			gotAlias, gotCancelled := promptAlias(tt.opts)

			if gotAlias != tt.wantAlias {
				t.Fatalf("alias mismatch: got %q want %q", gotAlias, tt.wantAlias)
			}
			if gotCancelled != tt.wantCancelled {
				t.Fatalf("cancelled mismatch: got %v want %v", gotCancelled, tt.wantCancelled)
			}
			if calls != tt.wantPromptRuns {
				t.Fatalf("prompt call count mismatch: got %d want %d", calls, tt.wantPromptRuns)
			}
		})
	}
}

func TestPromptGroupSelection(t *testing.T) {
	t.Run("esc loops when esc value is empty", func(t *testing.T) {
		origPick := pickOption
		t.Cleanup(func() {
			pickOption = origPick
		})

		picks := []string{"", "__none__"}
		calls := 0
		pickOption = func(title string, options []bridge.PickerOption) string {
			if calls >= len(picks) {
				t.Fatalf("unexpected extra pick call #%d", calls+1)
			}
			s := picks[calls]
			calls++
			return s
		}

		got := promptGroupSelection(groupPromptOptions{
			ExistingGroups: []string{"infra"},
			NoneValue:      "",
			EscValue:       "",
		})

		if got != "" {
			t.Fatalf("group mismatch: got %q want empty", got)
		}
		if calls != 2 {
			t.Fatalf("pick call count mismatch: got %d want 2", calls)
		}
	})

	t.Run("esc returns current in edit mode", func(t *testing.T) {
		origPick := pickOption
		t.Cleanup(func() {
			pickOption = origPick
		})

		calls := 0
		pickOption = func(title string, options []bridge.PickerOption) string {
			calls++
			return ""
		}

		got := promptGroupSelection(groupPromptOptions{
			ExistingGroups: []string{"infra"},
			CurrentGroup:   "infra",
			MarkCurrent:    true,
			NoneValue:      "Uncategorized",
			EscValue:       "infra",
		})

		if got != "infra" {
			t.Fatalf("group mismatch: got %q want %q", got, "infra")
		}
		if calls != 1 {
			t.Fatalf("pick call count mismatch: got %d want 1", calls)
		}
	})

	t.Run("new group name validates reserved and comma values", func(t *testing.T) {
		origPick := pickOption
		origPrompt := promptText
		t.Cleanup(func() {
			pickOption = origPick
			promptText = origPrompt
		})

		pickOption = func(title string, options []bridge.PickerOption) string {
			return "__new__"
		}

		prompts := []bridge.TextPromptResult{
			{Value: "none"},
			{Value: "ops,team"},
			{Value: "ops"},
		}
		promptCalls := 0
		promptText = func(label, defaultValue string) bridge.TextPromptResult {
			if promptCalls >= len(prompts) {
				t.Fatalf("unexpected extra prompt call #%d", promptCalls+1)
			}
			r := prompts[promptCalls]
			promptCalls++
			return r
		}

		got := promptGroupSelection(groupPromptOptions{
			ExistingGroups: []string{"infra"},
			NoneValue:      "",
			EscValue:       "",
		})

		if got != "ops" {
			t.Fatalf("group mismatch: got %q want %q", got, "ops")
		}
		if promptCalls != 3 {
			t.Fatalf("prompt call count mismatch: got %d want 3", promptCalls)
		}
	})

	t.Run("new group cancelled returns to picker", func(t *testing.T) {
		origPick := pickOption
		origPrompt := promptText
		t.Cleanup(func() {
			pickOption = origPick
			promptText = origPrompt
		})

		picks := []string{"__new__", "infra"}
		pickCalls := 0
		pickOption = func(title string, options []bridge.PickerOption) string {
			if pickCalls >= len(picks) {
				t.Fatalf("unexpected extra pick call #%d", pickCalls+1)
			}
			s := picks[pickCalls]
			pickCalls++
			return s
		}

		promptText = func(label, defaultValue string) bridge.TextPromptResult {
			return bridge.TextPromptResult{Cancelled: true}
		}

		got := promptGroupSelection(groupPromptOptions{
			ExistingGroups: []string{"infra"},
			NoneValue:      "",
			EscValue:       "",
		})

		if got != "infra" {
			t.Fatalf("group mismatch: got %q want %q", got, "infra")
		}
		if pickCalls != 2 {
			t.Fatalf("pick call count mismatch: got %d want 2", pickCalls)
		}
	})

	t.Run("marks current option in picker data", func(t *testing.T) {
		origPick := pickOption
		t.Cleanup(func() {
			pickOption = origPick
		})

		pickOption = func(title string, options []bridge.PickerOption) string {
			foundCurrent := false
			for _, opt := range options {
				if opt.Value == "prod" && opt.IsCurrent {
					foundCurrent = true
				}
				if opt.Value == "Uncategorized" {
					t.Fatalf("uncategorized should not be listed as a normal option")
				}
			}
			if !foundCurrent {
				t.Fatal("expected current group to be marked in options")
			}
			return "__none__"
		}

		got := promptGroupSelection(groupPromptOptions{
			ExistingGroups: []string{"infra", "prod", "Uncategorized"},
			CurrentGroup:   "prod",
			MarkCurrent:    true,
			NoneValue:      "Uncategorized",
			EscValue:       "prod",
		})

		if got != "Uncategorized" {
			t.Fatalf("group mismatch: got %q want %q", got, "Uncategorized")
		}
	})
}
