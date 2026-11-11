package config

import "testing"

func TestSaveConfig_LiteralMergeKeyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Hooks: Hooks{PostCreate: []Hook{{
		Type: "command", Command: "true", Env: map[string]string{"<<": "literal", "NAME": "value"},
	}}}}
	if err := SaveConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load saved configuration: %v", err)
	}
	if got := loaded.Hooks.PostCreate[0].Env["<<"]; got != "literal" {
		t.Fatalf("literal key changed after save/load: got %q", got)
	}
}
