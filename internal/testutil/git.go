// Package testutil provides helpers shared across tests.
// Package testutil provides helpers shared across tests.
// Package testutil provides helpers shared across tests.
// Package testutil provides helpers shared across tests.
package testutil

import "testing"

// ConfigureTestRepo applies common git configuration used in tests.
//
// The runner is responsible for executing git commands within the provided
// repository directory and should handle errors appropriately.
func ConfigureTestRepo(t *testing.T, repoDir string, runner func(dir string, args ...string)) {
	t.Helper()

	commands := [][]string{
		{"config", "user.name", "Test User"},
		{"config", "user.email", "test@example.com"},
		{"config", "commit.gpgsign", "false"},
	}

	for _, args := range commands {
		runner(repoDir, args...)
	}
}
