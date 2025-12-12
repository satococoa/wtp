package testutil

import (
	"reflect"
	"testing"
)

func TestConfigureTestRepo(t *testing.T) {
	repoDir := t.TempDir()

	var calls [][]string
	runner := func(dir string, args ...string) {
		if dir != repoDir {
			t.Fatalf("runner dir = %s, want %s", dir, repoDir)
		}
		calls = append(calls, args)
	}

	ConfigureTestRepo(t, repoDir, runner)

	want := [][]string{
		{"config", "user.name", "Test User"},
		{"config", "user.email", "test@example.com"},
		{"config", "commit.gpgsign", "false"},
	}

	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("runner calls = %#v, want %#v", calls, want)
	}
}
