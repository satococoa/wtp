package e2e

import (
	"strings"
	"testing"

	"github.com/satococoa/wtp/v2/test/e2e/framework"
)

func TestOptionalHooks(t *testing.T) {
	for _, hookType := range []string{"copy", "symlink"} {
		for _, optional := range []bool{false, true} {
			name := hookType + "-required"
			if optional {
				name = hookType + "-optional"
			}
			t.Run(name, func(t *testing.T) {
				env := framework.NewTestEnvironment(t)
				defer env.Cleanup()
				repo := env.CreateTestRepo(name)
				cfg := `version: "1.0"
defaults:
  base_dir: ../worktrees
hooks:
  post_create:
    - type: TYPE
      from: missing.env
      to: copied.env
OPTIONAL
    - type: command
      command: touch after-hook
`
				cfg = strings.ReplaceAll(cfg, "TYPE", hookType)
				setting := ""
				if optional {
					setting = "      optional: true"
				}
				cfg = strings.ReplaceAll(cfg, "OPTIONAL", setting)
				repo.WriteConfig(cfg)
				output, err := repo.RunWTP("add", "-b", name)
				// Hook failures remain warnings after the worktree was created.
				framework.AssertNoError(t, err)
				after := env.FileExists(env.TmpDir() + "/worktrees/" + name + "/after-hook")
				framework.AssertEqual(t, optional, after)
				if optional {
					framework.AssertOutputContains(t, output, "Skipping: missing.env (not found)")
				} else {
					framework.AssertOutputContains(t, output, "source path does not exist")
				}
			})
		}
	}
}
