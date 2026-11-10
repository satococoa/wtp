package hooks

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/satococoa/wtp/v2/internal/config"
)

func TestOptionalHooksErrorHandling(t *testing.T) {
	for _, typ := range []string{config.HookTypeCopy, config.HookTypeSymlink} {
		for _, tc := range []struct {
			name           string
			optional       bool
			source         string
			wantErr        bool
			wantPermission bool
		}{
			{"missing-default", false, "missing", true, false},
			{"missing-optional", true, "missing", false, false},
			{"permission-default", false, "blocked/file", true, true},
			{"permission-optional", true, "blocked/file", true, true},
			{"loop-optional", true, "loop", true, false},
			{"present-optional", true, "present", false, false},
		} {
			t.Run(typ+"/"+tc.name, func(t *testing.T) {
				root := t.TempDir()
				dst := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(root, "present"), []byte("x"), 0600))
				blocked := filepath.Join(root, "blocked")
				require.NoError(t, os.Mkdir(blocked, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(blocked, "file"), []byte("x"), 0600))
				require.NoError(t, os.Chmod(blocked, 0000))
				t.Cleanup(func() { _ = os.Chmod(blocked, 0700) })
				require.NoError(t, os.Symlink("loop", filepath.Join(root, "loop")))
				cfg := &config.Config{Hooks: config.Hooks{PostCreate: []config.Hook{
					{Type: typ, From: tc.source, To: "out", Optional: tc.optional},
					{Type: config.HookTypeCommand, Command: "touch after"},
				}}}
				if tc.wantPermission {
					if _, err := os.Stat(filepath.Join(blocked, "file")); !errors.Is(err, os.ErrPermission) {
						t.Skip("permission denial unavailable for this user/filesystem")
					}
				}
				var buf bytes.Buffer
				err := NewExecutor(cfg, root).ExecutePostCreateHooks(&buf, dst)
				require.Equal(t, tc.wantErr, err != nil, "error=%v", err)
				_, afterErr := os.Stat(filepath.Join(dst, "after"))
				if tc.wantErr && afterErr == nil {
					t.Fatal("ran subsequent hook after failure")
				}
				if !tc.wantErr && afterErr != nil {
					t.Fatal("did not continue after success/skip")
				}
				if tc.wantErr && strings.Contains(buf.String(), "Skipping:") {
					t.Fatal("skipped nonmissing error")
				}
				if tc.source == "missing" && tc.optional && !strings.Contains(buf.String(), "Skipping: missing (not found)") {
					t.Fatal("missing skip diagnostic")
				}
				if tc.wantPermission {
					require.ErrorIs(t, err, os.ErrPermission)
					require.Contains(t, err.Error(), "permission denied")
				}
			})
		}
	}
}
