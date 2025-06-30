package main

import (
	"testing"
)

func TestInitCommand(t *testing.T) {
	// This test requires a full git repository mock
	t.Skip("Full init command testing requires repository mock")
}

func TestNewInitCommand(t *testing.T) {
	cmd := NewInitCommand()
	if cmd == nil {
		t.Fatal("NewInitCommand() returned nil")
	}
	if cmd.Name != "init" {
		t.Errorf("Expected command name 'init', got '%s'", cmd.Name)
	}
	if cmd.Action == nil {
		t.Error("Init command has no action")
	}
}

func TestConfigFileMode(t *testing.T) {
	if configFileMode != 0o600 {
		t.Errorf("Expected configFileMode to be 0o600, got %o", configFileMode)
	}
}
