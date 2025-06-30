package main

import (
	"testing"

	"github.com/urfave/cli/v3"
)

func TestRemoveCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		flags   map[string]any
		wantErr bool
	}{
		{
			name:    "no branch name",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "valid branch name",
			args:    []string{"feature/auth"},
			wantErr: false, // Would need mock git repo
		},
		{
			name:    "force-branch without with-branch",
			args:    []string{"feature/auth"},
			flags:   map[string]any{"force-branch": true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test requires a full CLI context and git repository mock
			t.Skip("Full CLI command testing requires repository mock")
		})
	}
}

func TestNewRemoveCommand(t *testing.T) {
	cmd := NewRemoveCommand()
	if cmd == nil {
		t.Fatal("NewRemoveCommand() returned nil")
	}
	if cmd.Name != "remove" {
		t.Errorf("Expected command name 'remove', got '%s'", cmd.Name)
	}
	if cmd.Action == nil {
		t.Error("Remove command has no action")
	}

	// Test flags exist
	expectedFlags := []string{"force", "with-branch", "force-branch"}
	flagMap := make(map[string]bool)
	for _, flag := range cmd.Flags {
		if f, ok := flag.(*cli.BoolFlag); ok {
			flagMap[f.Name] = true
		}
	}

	for _, expected := range expectedFlags {
		if !flagMap[expected] {
			t.Errorf("Expected flag '%s' not found", expected)
		}
	}
}
