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

func TestExecutePostCreateHooks_CopyDirectoryRecursively(t *testing.T) {
	// Test that directories are copied recursively with nested structure preserved
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	// Create source directory with nested structure
	srcDir := filepath.Join(repoRoot, "templates")
	err = os.MkdirAll(filepath.Join(srcDir, "subdir"), directoryPermissions)
	require.NoError(t, err)

	// Create files in source directory
	err = os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0644)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: "templates",
					To:   "copied-templates",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check that directory and files were copied
	dstFile1 := filepath.Join(worktreeDir, "copied-templates", "file1.txt")
	content1, err := os.ReadFile(dstFile1)
	require.NoError(t, err)
	assert.Equal(t, "content1", string(content1))

	dstFile2 := filepath.Join(worktreeDir, "copied-templates", "subdir", "file2.txt")
	content2, err := os.ReadFile(dstFile2)
	require.NoError(t, err)
	assert.Equal(t, "content2", string(content2))

	// Verify output contains progress messages
	output := buf.String()
	assert.Contains(t, output, "Copying: templates → copied-templates")
}

func TestExecutePostCreateHooks_CopyFilePreservesPermissions(t *testing.T) {
	// Test that file permissions are preserved during copy operations
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	// Create source file with specific permissions
	srcFile := filepath.Join(repoRoot, "script.sh")
	err = os.WriteFile(srcFile, []byte("#!/bin/bash\necho hello"), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: "script.sh",
					To:   "copied-script.sh",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check that file permissions were preserved
	dstFile := filepath.Join(worktreeDir, "copied-script.sh")
	dstInfo, err := os.Stat(dstFile)
	require.NoError(t, err)

	srcInfo, err := os.Stat(srcFile)
	require.NoError(t, err)

	assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())
}

func TestExecutePostCreateHooks_CopyWithAbsolutePaths(t *testing.T) {
	// Test that copy hooks work with absolute source and destination paths
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")
	externalDir := filepath.Join(tempDir, "external")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(externalDir, directoryPermissions)
	require.NoError(t, err)

	// Create source file in external directory
	srcFile := filepath.Join(externalDir, "external.txt")
	srcContent := "external content"
	err = os.WriteFile(srcFile, []byte(srcContent), 0644)
	require.NoError(t, err)

	// Create destination directory outside worktree
	outputDir := filepath.Join(tempDir, "output")
	err = os.MkdirAll(outputDir, directoryPermissions)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: srcFile,                                // absolute path
					To:   filepath.Join(outputDir, "result.txt"), // absolute path
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check that file was copied to absolute destination
	dstFile := filepath.Join(outputDir, "result.txt")
	dstContent, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, srcContent, string(dstContent))
}

// Error handling tests for copyFile function
func TestExecutor_copyFile_SourceFileOpenError(t *testing.T) {
	executor := NewExecutor(nil, "/test/repo")

	// Try to copy non-existent file
	err := executor.copyFile("/nonexistent/source.txt", "/tmp/dest.txt")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open source file")
}

func TestExecutor_copyFile_DestinationCreateError(t *testing.T) {
	tempDir := t.TempDir()
	srcFile := filepath.Join(tempDir, "source.txt")

	// Create source file
	err := os.WriteFile(srcFile, []byte("test content"), 0644)
	require.NoError(t, err)

	executor := NewExecutor(nil, "/test/repo")

	// Try to create file in non-existent directory without creating parent dirs
	invalidDest := "/nonexistent/directory/dest.txt"
	err = executor.copyFile(srcFile, invalidDest)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create destination file")
}

func TestExecutor_copyFile_PermissionGetError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Permission test not applicable on Windows")
	}

	tempDir := t.TempDir()
	srcFile := filepath.Join(tempDir, "source.txt")
	dstFile := filepath.Join(tempDir, "dest.txt")

	// Create source file
	err := os.WriteFile(srcFile, []byte("test content"), 0644)
	require.NoError(t, err)

	executor := NewExecutor(nil, "/test/repo")

	// Copy the file first
	err = executor.copyFile(srcFile, dstFile)
	require.NoError(t, err)

	// Remove source to trigger stat error in copyFile
	err = os.Remove(srcFile)
	require.NoError(t, err)

	// Try to copy again - should fail at getting source file info
	err = executor.copyFile(srcFile, dstFile+"2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open source file")
}

// Error handling tests for copyDir function
func TestExecutor_copyDir_SourceDirectoryNotExist(t *testing.T) {
	executor := NewExecutor(nil, "/test/repo")

	// Try to copy non-existent directory
	err := executor.copyDir("/nonexistent/source", "/tmp/dest")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stat source directory")
}

func TestExecutor_copyDir_DestinationCreateError(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "source")

	// Create source directory
	err := os.MkdirAll(srcDir, directoryPermissions)
	require.NoError(t, err)

	executor := NewExecutor(nil, "/test/repo")

	// Try to create directory where parent doesn't exist and we can't create it
	invalidDest := "/root/nonexistent/dest" // Should fail on most systems
	err = executor.copyDir(srcDir, invalidDest)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create destination directory")
}

func TestExecutor_copyDir_ReadDirectoryError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Permission test not applicable on Windows")
	}

	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "source")
	dstDir := filepath.Join(tempDir, "dest")

	// Create source directory
	err := os.MkdirAll(srcDir, directoryPermissions)
	require.NoError(t, err)

	// Remove read permission from source directory
	err = os.Chmod(srcDir, 0200) // write-only
	require.NoError(t, err)

	// Restore permissions after test
	defer func() {
		_ = os.Chmod(srcDir, directoryPermissions)
	}()

	executor := NewExecutor(nil, "/test/repo")

	err = executor.copyDir(srcDir, dstDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read source directory")
}

func TestExecutor_copyDir_NestedDirectorySuccess(t *testing.T) {
	// Test successful recursive directory copying with multiple levels
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "source")
	dstDir := filepath.Join(tempDir, "dest")

	// Create nested source structure
	level1 := filepath.Join(srcDir, "level1")
	level2 := filepath.Join(level1, "level2")
	err := os.MkdirAll(level2, directoryPermissions)
	require.NoError(t, err)

	// Create files at different levels
	err = os.WriteFile(filepath.Join(srcDir, "root.txt"), []byte("root content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(level1, "level1.txt"), []byte("level1 content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(level2, "level2.txt"), []byte("level2 content"), 0644)
	require.NoError(t, err)

	executor := NewExecutor(nil, "/test/repo")

	err = executor.copyDir(srcDir, dstDir)
	assert.NoError(t, err)

	// Verify all files were copied correctly
	rootContent, err := os.ReadFile(filepath.Join(dstDir, "root.txt"))
	require.NoError(t, err)
	assert.Equal(t, "root content", string(rootContent))

	level1Content, err := os.ReadFile(filepath.Join(dstDir, "level1", "level1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "level1 content", string(level1Content))

	level2Content, err := os.ReadFile(filepath.Join(dstDir, "level1", "level2", "level2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "level2 content", string(level2Content))
}

func TestExecutor_copyDir_FailsWhenNestedFileCannotBeCopied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Permission test not applicable on Windows")
	}

	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "source")
	dstDir := filepath.Join(tempDir, "dest")

	// Create source directory with nested structure
	nestedDir := filepath.Join(srcDir, "nested")
	err := os.MkdirAll(nestedDir, directoryPermissions)
	require.NoError(t, err)

	// Create a file that we'll make unreadable
	unreadableFile := filepath.Join(nestedDir, "unreadable.txt")
	err = os.WriteFile(unreadableFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Remove read permission from the file
	err = os.Chmod(unreadableFile, 0000)
	require.NoError(t, err)

	// Restore permissions after test
	defer func() {
		_ = os.Chmod(unreadableFile, 0644)
	}()

	executor := NewExecutor(nil, "/test/repo")

	err = executor.copyDir(srcDir, dstDir)
	assert.Error(t, err)
	// The error should propagate from the nested copyFile call
}

// Edge case tests
func TestExecutePostCreateHooks_CopyEmptyFile(t *testing.T) {
	// Test copying empty files (edge case for io.Copy)
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	// Create empty source file
	srcFile := filepath.Join(repoRoot, "empty.txt")
	err = os.WriteFile(srcFile, []byte(""), 0644)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: "empty.txt",
					To:   "copied-empty.txt",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check that empty file was copied
	dstFile := filepath.Join(worktreeDir, "copied-empty.txt")
	dstContent, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, "", string(dstContent))
}

func TestExecutePostCreateHooks_CopyEmptyDirectory(t *testing.T) {
	// Test copying empty directories (edge case for directory iteration)
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	// Create empty source directory
	srcDir := filepath.Join(repoRoot, "empty-dir")
	err = os.MkdirAll(srcDir, directoryPermissions)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: "empty-dir",
					To:   "copied-empty-dir",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check that empty directory was copied
	dstDir := filepath.Join(worktreeDir, "copied-empty-dir")
	dstInfo, err := os.Stat(dstDir)
	require.NoError(t, err)
	assert.True(t, dstInfo.IsDir())

	// Verify directory is empty
	entries, err := os.ReadDir(dstDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestExecutePostCreateHooks_CopyFileWithSpecialCharacters(t *testing.T) {
	// Test copying files with special characters in names
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	err := os.MkdirAll(repoRoot, directoryPermissions)
	require.NoError(t, err)
	err = os.MkdirAll(worktreeDir, directoryPermissions)
	require.NoError(t, err)

	// Create source file with special characters (that are filesystem-safe)
	specialName := "file-with-spaces and_underscores.txt"
	srcFile := filepath.Join(repoRoot, specialName)
	err = os.WriteFile(srcFile, []byte("special content"), 0644)
	require.NoError(t, err)

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: specialName,
					To:   "copied-" + specialName,
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	var buf bytes.Buffer
	err = executor.ExecutePostCreateHooks(&buf, worktreeDir)
	assert.NoError(t, err)

	// Check that file with special characters was copied
	dstFile := filepath.Join(worktreeDir, "copied-"+specialName)
	dstContent, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, "special content", string(dstContent))
}
