package hooks

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satococoa/wtp/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutePostCreateHooks_NilConfig(t *testing.T) {
	executor := NewExecutor(nil, "/test/repo")
	var buf bytes.Buffer
	err := executor.ExecutePostCreateHooks(&buf, "/test/worktree")
	assert.NoError(t, err)
}

func TestExecutePostCreateHooks_NoHooks(t *testing.T) {
	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{},
		},
	}
	executor := NewExecutor(cfg, "/test/repo")
	var buf bytes.Buffer
	err := executor.ExecutePostCreateHooks(&buf, "/test/worktree")
	assert.NoError(t, err)
}

func TestExecutePostCreateHooks_InvalidHookType(t *testing.T) {
	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: "invalid",
				},
			},
		},
	}
	executor := NewExecutor(cfg, "/test/repo")
	var buf bytes.Buffer
	err := executor.ExecutePostCreateHooks(&buf, "/test/worktree")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown hook type")
}

func TestExecutePostCreateHooks_CopyFile(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	// Create source file
	srcFile := filepath.Join(repoRoot, ".env.example")
	srcContent := "TEST_VAR=value"
	err = os.WriteFile(srcFile, []byte(srcContent), 0644)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: ".env.example",
					To:   ".env",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check that file was copied
	dstFile := filepath.Join(worktreeDir, ".env")
	dstContent, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, srcContent, string(dstContent))

	// Verify output contains progress messages
	output := buf.String()
	assert.Contains(t, output, "Running hook 1 of 1")
	assert.Contains(t, output, "✓ Hook 1 completed")
}

func TestExecutePostCreateHooks_Command(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping command test on Windows")
	}

	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: "echo 'Hello from hook'",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check output
	output := buf.String()
	assert.Contains(t, output, "Hello from hook")
	assert.Contains(t, output, "Running hook 1 of 1")
	assert.Contains(t, output, "✓ Hook 1 completed")
}

func TestExecutePostCreateHooks_MultipleHooks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping command test on Windows")
	}

	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	// Create source file for copy hook
	srcFile := filepath.Join(repoRoot, "template.txt")
	err = os.WriteFile(srcFile, []byte("template content"), 0644)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: "echo 'First hook'",
				},
				{
					Type: config.HookTypeCopy,
					From: "template.txt",
					To:   "output.txt",
				},
				{
					Type:    config.HookTypeCommand,
					Command: "echo 'Third hook'",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check output
	output := buf.String()
	assert.Contains(t, output, "First hook")
	assert.Contains(t, output, "Third hook")
	assert.Contains(t, output, "Running hook 1 of 3")
	assert.Contains(t, output, "Running hook 2 of 3")
	assert.Contains(t, output, "Running hook 3 of 3")
	assert.Contains(t, output, "✓ Hook 1 completed")
	assert.Contains(t, output, "✓ Hook 2 completed")
	assert.Contains(t, output, "✓ Hook 3 completed")

	// Check that file was copied
	dstFile := filepath.Join(worktreeDir, "output.txt")
	_, err = os.Stat(dstFile)
	assert.NoError(t, err)
}

func TestExecutePostCreateHooks_CommandWithEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping command test on Windows")
	}

	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: "echo $CUSTOM_VAR",
					Env: map[string]string{
						"CUSTOM_VAR": "custom_value",
					},
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check output
	output := buf.String()
	assert.Contains(t, output, "custom_value")
}

func TestExecutePostCreateHooks_CommandWithWorkDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping command test on Windows")
	}

	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")
	subDir := filepath.Join(worktreeDir, "subdir")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(subDir, directoryPermissions)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: "pwd",
					WorkDir: "subdir",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check output
	output := buf.String()
	assert.Contains(t, output, "subdir")
}

func TestExecutePostCreateHooks_CommandFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping command test on Windows")
	}

	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: "exit 1",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute hook")
}

func TestExecutePostCreateHooks_CopyNonExistentFile(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: "nonexistent.txt",
					To:   "output.txt",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute hook")
}

func TestExecutePostCreateHooks_CopyToNestedDirectory(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	// Create source file
	srcFile := filepath.Join(repoRoot, "template.txt")
	srcContent := "template content"
	err = os.WriteFile(srcFile, []byte(srcContent), 0644)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: "template.txt",
					To:   "nested/dir/output.txt",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check that file was copied and directory was created
	dstFile := filepath.Join(worktreeDir, "nested", "dir", "output.txt")
	dstContent, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, srcContent, string(dstContent))
}

func TestExecutePostCreateHooks_AbsoluteWorkDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping command test on Windows")
	}

	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")
	absoluteDir := filepath.Join(tempDir, "absolute")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(absoluteDir, directoryPermissions)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: "pwd",
					WorkDir: absoluteDir,
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check output
	output := buf.String()
	assert.Contains(t, output, absoluteDir)
}

func TestExecutePostCreateHooks_EnvironmentVariables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping command test on Windows")
	}

	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: "echo WORKTREE=$GIT_WTP_WORKTREE_PATH REPO=$GIT_WTP_REPO_ROOT",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check output
	output := buf.String()
	assert.Contains(t, output, fmt.Sprintf("WORKTREE=%s", worktreeDir))
	assert.Contains(t, output, fmt.Sprintf("REPO=%s", repoRoot))
}

// streamingWriter tracks when writes occur to verify real-time streaming
type streamingWriter struct {
	writes []writeRecord
	mu     sync.Mutex
}

type writeRecord struct {
	data string
	time time.Time
}

func (sw *streamingWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.writes = append(sw.writes, writeRecord{
		data: string(p),
		time: time.Now(),
	})
	return len(p), nil
}

func TestExecutePostCreateHooks_StreamingOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping streaming test on Windows")
	}

	repoRoot, worktreeDir, scriptPath := setupStreamingTestDirectories(t)
	cfg := createStreamingTestConfig(scriptPath)

	sw := &streamingWriter{}
	executor := NewExecutor(cfg, repoRoot)

	err := executor.ExecutePostCreateHooks(sw, worktreeDir)
	if err != nil {
		t.Fatalf("Failed to execute hooks: %v", err)
	}

	verifyStreamingOutput(t, sw)
}

func setupStreamingTestDirectories(t *testing.T) (repoRoot, worktreeDir, scriptPath string) {
	tempDir := t.TempDir()
	repoRoot = filepath.Join(tempDir, "repo")
	worktreeDir = filepath.Join(tempDir, "worktree")

	// Create directories
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatalf("Failed to create worktree dir: %v", err)
	}

	// Create a script that outputs multiple lines with delays
	scriptPath = filepath.Join(repoRoot, "stream-test.sh")
	scriptContent := `#!/bin/bash
echo "Starting..."
sleep 0.1
echo "Processing..."
sleep 0.1
echo "Done!"
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	return
}

func createStreamingTestConfig(scriptPath string) *config.Config {
	return &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: scriptPath,
				},
			},
		},
	}
}

func verifyStreamingOutput(t *testing.T, sw *streamingWriter) {
	// Verify we got multiple writes (streaming) not just one big write
	if len(sw.writes) < 4 { // At least: Running log + 3 echo outputs
		t.Errorf("Expected multiple writes for streaming output, got %d writes", len(sw.writes))
		for i, w := range sw.writes {
			t.Logf("Write %d: %q", i, w.data)
		}
	}

	verifyOutputContent(t, sw)
	verifyStreamingTiming(t, sw)
}

func verifyOutputContent(t *testing.T, sw *streamingWriter) {
	var allOutput strings.Builder
	for _, w := range sw.writes {
		allOutput.WriteString(w.data)
	}
	output := allOutput.String()

	// Check expected content is present
	assert.Contains(t, output, "Starting...")
	assert.Contains(t, output, "Processing...")
	assert.Contains(t, output, "Done!")
}

func verifyStreamingTiming(t *testing.T, sw *streamingWriter) {
	// Find indices of our expected outputs
	var startIdx, procIdx, doneIdx int
	for i, w := range sw.writes {
		if strings.Contains(w.data, "Starting...") {
			startIdx = i
		} else if strings.Contains(w.data, "Processing...") {
			procIdx = i
		} else if strings.Contains(w.data, "Done!") {
			doneIdx = i
		}
	}

	// Verify outputs came in the right order with delays
	if startIdx >= procIdx || procIdx >= doneIdx {
		t.Error("Expected streaming output with delays, but all writes happened too quickly")
	}
}

func TestExecutePostCreateHooks_RealTimeLineBuffering(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping line buffering test on Windows")
	}

	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	require.NoError(t, os.MkdirAll(repoRoot, 0755))
	require.NoError(t, os.MkdirAll(worktreeDir, 0755))

	// Create a script that outputs lines with delays
	scriptPath := filepath.Join(repoRoot, "line-buffer-test.sh")
	scriptContent := `#!/bin/bash
echo "Line 1"
sleep 0.1
echo "Line 2"
sleep 0.1
echo "Line 3"
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0755))

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: scriptPath,
				},
			},
		},
	}

	sw := &streamingWriter{}
	executor := NewExecutor(cfg, repoRoot)

	start := time.Now()
	err := executor.ExecutePostCreateHooks(sw, worktreeDir)
	duration := time.Since(start)

	require.NoError(t, err)

	// Should take at least 0.2 seconds due to sleeps
	assert.True(t, duration >= 200*time.Millisecond, "Expected execution to take at least 200ms, took %v", duration)

	// Verify we got line-by-line output
	var lines []string
	for _, w := range sw.writes {
		if strings.Contains(w.data, "Line") && !strings.Contains(w.data, "Running:") {
			lines = append(lines, strings.TrimSpace(w.data))
		}
	}

	assert.Equal(t, []string{"Line 1", "Line 2", "Line 3"}, lines)

	// Verify timing between lines
	var lineTimes []time.Time
	for _, w := range sw.writes {
		if strings.Contains(w.data, "Line") {
			lineTimes = append(lineTimes, w.time)
		}
	}

	if len(lineTimes) >= 2 {
		gap1 := lineTimes[1].Sub(lineTimes[0])
		assert.True(t, gap1 >= 50*time.Millisecond, "Expected at least 50ms between Line 1 and Line 2, got %v", gap1)
	}

	if len(lineTimes) >= 3 {
		gap2 := lineTimes[2].Sub(lineTimes[1])
		assert.True(t, gap2 >= 50*time.Millisecond, "Expected at least 50ms between Line 2 and Line 3, got %v", gap2)
	}
}
