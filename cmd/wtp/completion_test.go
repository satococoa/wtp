package main

import (
	"testing"
)

func TestNewCompletionCommand(t *testing.T) {
	cmd := NewCompletionCommand()
	if cmd == nil {
		t.Fatal("NewCompletionCommand() returned nil")
	}
	if cmd.Name != "completion" {
		t.Errorf("Expected command name 'completion', got '%s'", cmd.Name)
	}

	// Check subcommands
	expectedSubcommands := []string{"bash", "zsh", "fish", "__branches", "__worktrees"}
	if len(cmd.Commands) != len(expectedSubcommands) {
		t.Errorf("Expected %d subcommands, got %d", len(expectedSubcommands), len(cmd.Commands))
	}

	subCommandMap := make(map[string]bool)
	for _, subcmd := range cmd.Commands {
		subCommandMap[subcmd.Name] = true
	}

	for _, expected := range expectedSubcommands {
		if !subCommandMap[expected] {
			t.Errorf("Expected subcommand '%s' not found", expected)
		}
	}
}

func TestNewShellInitCommand(t *testing.T) {
	cmd := NewShellInitCommand()
	if cmd == nil {
		t.Fatal("NewShellInitCommand() returned nil")
	}
	if cmd.Name != "shell-init" {
		t.Errorf("Expected command name 'shell-init', got '%s'", cmd.Name)
	}
	if cmd.Action == nil {
		t.Error("Shell-init command has no action")
	}
}

func TestCompletionFunctions(t *testing.T) {
	t.Run("completionBash", func(t *testing.T) {
		// This would require capturing output
		t.Skip("Completion function testing requires output capture")
	})

	t.Run("completionZsh", func(t *testing.T) {
		// This would require capturing output
		t.Skip("Completion function testing requires output capture")
	})

	t.Run("completionFish", func(t *testing.T) {
		// This would require a CLI command mock
		t.Skip("Fish completion testing requires CLI command mock")
	})
}
