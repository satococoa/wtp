package hooks

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satococoa/wtp/internal/config"
)

func TestNewExecutor(t *testing.T) {
	cfg := &config.Config{}
	repoRoot := "/test/repo"

	executor := NewExecutor(cfg, repoRoot)

	if executor.config != cfg {
		t.Error("Config not set correctly")
	}

	if executor.repoRoot != repoRoot {
		t.Error("RepoRoot not set correctly")
	}
}

func TestExecutePostCreateHooks_NoHooks(t *testing.T) {
	cfg := &config.Config{}
	executor := NewExecutor(cfg, "/test/repo")

	err := executor.ExecutePostCreateHooks(io.Discard, "/test/worktree")
	if err != nil {
		t.Errorf("Expected no error for config without hooks, got %v", err)
	}
}

func TestExecuteCopyHook(t *testing.T) {
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create repo and worktree directories
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatalf("Failed to create worktree dir: %v", err)
	}

	// Create source file
	sourceFile := filepath.Join(repoRoot, "test.txt")
	sourceContent := "test content"
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: "test.txt",
					To:   "copied.txt",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	err := executor.ExecutePostCreateHooks(io.Discard, worktreeDir)
	if err != nil {
		t.Fatalf("Failed to execute copy hook: %v", err)
	}

	// Verify file was copied
	destFile := filepath.Join(worktreeDir, "copied.txt")
	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(content) != sourceContent {
		t.Errorf("Expected content %s, got %s", sourceContent, string(content))
	}
}

func TestExecuteCopyHook_AbsolutePaths(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	// Create directories
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("Failed to create dest dir: %v", err)
	}

	// Create source file
	sourceFile := filepath.Join(sourceDir, "test.txt")
	sourceContent := "absolute path test"
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: sourceFile,                           // Absolute path
					To:   filepath.Join(destDir, "copied.txt"), // Absolute path
				},
			},
		},
	}

	executor := NewExecutor(cfg, "/fake/repo")
	err := executor.ExecutePostCreateHooks(io.Discard, "/fake/worktree")
	if err != nil {
		t.Fatalf("Failed to execute copy hook with absolute paths: %v", err)
	}

	// Verify file was copied
	destFile := filepath.Join(destDir, "copied.txt")
	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(content) != sourceContent {
		t.Errorf("Expected content %s, got %s", sourceContent, string(content))
	}
}

func TestExecuteCopyHook_Directory(t *testing.T) {
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create repo and worktree directories
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatalf("Failed to create worktree dir: %v", err)
	}

	// Create source directory with files
	sourceDir := filepath.Join(repoRoot, "config")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	sourceFile := filepath.Join(sourceDir, "app.conf")
	sourceContent := "config content"
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: "config",
					To:   "config-copy",
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	err := executor.ExecutePostCreateHooks(io.Discard, worktreeDir)
	if err != nil {
		t.Fatalf("Failed to execute directory copy hook: %v", err)
	}

	// Verify directory and file were copied
	destFile := filepath.Join(worktreeDir, "config-copy", "app.conf")
	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(content) != sourceContent {
		t.Errorf("Expected content %s, got %s", sourceContent, string(content))
	}
}

func TestExecuteCopyHook_MissingSource(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: "nonexistent.txt",
					To:   "dest.txt",
				},
			},
		},
	}

	executor := NewExecutor(cfg, tempDir)
	err := executor.ExecutePostCreateHooks(io.Discard, tempDir)
	if err == nil {
		t.Error("Expected error for missing source file, got nil")
	}
}

func TestExecuteCommandHook(t *testing.T) {
	tempDir := t.TempDir()

	// Create a simple script based on OS
	var scriptContent, scriptName, command string
	if runtime.GOOS == "windows" {
		scriptName = "test.bat"
		scriptContent = "@echo test output > output.txt"
		command = "cmd"
	} else {
		scriptName = "test.sh"
		scriptContent = "#!/bin/bash\necho 'test output' > output.txt"
		command = "bash"
	}

	scriptPath := filepath.Join(tempDir, scriptName)
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	var args []string
	if runtime.GOOS == "windows" {
		args = []string{"/c", scriptName}
	} else {
		args = []string{scriptName}
	}

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: command,
					Args:    args,
					Env: map[string]string{
						"TEST_VAR": "test_value",
					},
				},
			},
		},
	}

	executor := NewExecutor(cfg, "/fake/repo")
	err := executor.ExecutePostCreateHooks(io.Discard, tempDir)
	if err != nil {
		t.Fatalf("Failed to execute command hook: %v", err)
	}

	// Verify output file was created
	outputFile := filepath.Join(tempDir, "output.txt")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected output file was not created")
	}
}

func TestExecuteCommandHook_Simple(t *testing.T) {
	tempDir := t.TempDir()

	// Use a simple command that should work on all platforms
	var command string
	var args []string

	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/c", "echo test > test_output.txt"}
	} else {
		command = "sh"
		args = []string{"-c", "echo test > test_output.txt"}
	}

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: command,
					Args:    args,
				},
			},
		},
	}

	executor := NewExecutor(cfg, "/fake/repo")
	err := executor.ExecutePostCreateHooks(io.Discard, tempDir)
	if err != nil {
		t.Fatalf("Failed to execute simple command hook: %v", err)
	}

	// Verify output file was created
	outputFile := filepath.Join(tempDir, "test_output.txt")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected output file was not created")
	}
}

func TestExecuteCommandHook_WithWorkDir(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Use a simple command that should work on all platforms
	var command string
	var args []string

	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/c", "echo test > workdir_test.txt"}
	} else {
		command = "sh"
		args = []string{"-c", "echo test > workdir_test.txt"}
	}

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: command,
					Args:    args,
					WorkDir: "subdir",
				},
			},
		},
	}

	executor := NewExecutor(cfg, "/fake/repo")
	err := executor.ExecutePostCreateHooks(io.Discard, tempDir)
	if err != nil {
		t.Fatalf("Failed to execute command hook with work dir: %v", err)
	}

	// Verify output file was created in subdirectory
	outputFile := filepath.Join(subDir, "workdir_test.txt")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected output file was not created in work directory")
	}
}

func TestExecuteCommandHook_FailingCommand(t *testing.T) {
	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: "nonexistent-command",
					Args:    []string{"arg1"},
				},
			},
		},
	}

	executor := NewExecutor(cfg, "/fake/repo")
	err := executor.ExecutePostCreateHooks(io.Discard, "/fake/worktree")
	if err == nil {
		t.Error("Expected error for failing command, got nil")
	}
}

func TestExecuteHook_InvalidType(t *testing.T) {
	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: "invalid-type",
				},
			},
		},
	}

	executor := NewExecutor(cfg, "/fake/repo")
	err := executor.ExecutePostCreateHooks(io.Discard, "/fake/worktree")
	if err == nil {
		t.Error("Expected error for invalid hook type, got nil")
	}
}

func TestExecutePostCreateHooks_MultipleHooks(t *testing.T) {
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create directories
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatalf("Failed to create worktree dir: %v", err)
	}

	// Create source file
	sourceFile := filepath.Join(repoRoot, "source.txt")
	if err := os.WriteFile(sourceFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	var command string
	var args []string
	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/c", "echo command executed > command_output.txt"}
	} else {
		command = "sh"
		args = []string{"-c", "echo command executed > command_output.txt"}
	}

	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: "source.txt",
					To:   "copied.txt",
				},
				{
					Type:    config.HookTypeCommand,
					Command: command,
					Args:    args,
				},
			},
		},
	}

	executor := NewExecutor(cfg, repoRoot)
	err := executor.ExecutePostCreateHooks(io.Discard, worktreeDir)
	if err != nil {
		t.Fatalf("Failed to execute multiple hooks: %v", err)
	}

	// Verify both hooks executed
	copiedFile := filepath.Join(worktreeDir, "copied.txt")
	if _, err := os.Stat(copiedFile); os.IsNotExist(err) {
		t.Error("Copy hook did not execute")
	}

	commandFile := filepath.Join(worktreeDir, "command_output.txt")
	if _, err := os.Stat(commandFile); os.IsNotExist(err) {
		t.Error("Command hook did not execute")
	}
}

func TestExecutePostCreateHooks_WithWriter(t *testing.T) {
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	worktreeDir := filepath.Join(tempDir, "worktree")

	// Create repo and worktree directories
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatalf("Failed to create worktree dir: %v", err)
	}

	// Create a test source file for copy hook
	sourceFile := filepath.Join(repoRoot, "test.txt")
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create config with both copy and command hooks
	cfg := &config.Config{
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: "test.txt",
					To:   "copied.txt",
				},
				{
					Type:    config.HookTypeCommand,
					Command: "echo",
					Args:    []string{"Hello from hook"},
				},
			},
		},
	}

	// Create a buffer to capture output
	var output strings.Builder

	executor := NewExecutor(cfg, repoRoot)

	// Test the new writer-based implementation
	err := executor.ExecutePostCreateHooks(&output, worktreeDir)
	if err != nil {
		t.Fatalf("Failed to execute hooks with writer: %v", err)
	}

	// Verify output contains hook execution logs
	outputStr := output.String()
	if !strings.Contains(outputStr, "Copying: test.txt → copied.txt") {
		t.Error("Output should contain copy hook log")
	}
	if !strings.Contains(outputStr, "Running: echo [Hello from hook]") {
		t.Error("Output should contain command hook log")
	}
	if !strings.Contains(outputStr, "Hello from hook") {
		t.Error("Output should contain command output")
	}

	// Verify hooks actually executed
	copiedFile := filepath.Join(worktreeDir, "copied.txt")
	if _, err := os.Stat(copiedFile); os.IsNotExist(err) {
		t.Error("Copy hook did not execute")
	}
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
	outputStr := allOutput.String()

	expectedStrings := []string{"Starting...", "Processing...", "Done!"}
	for _, expected := range expectedStrings {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Output should contain '%s'", expected)
		}
	}
}

func verifyStreamingTiming(t *testing.T, sw *streamingWriter) {
	if len(sw.writes) < 2 {
		return
	}

	// Calculate time differences between writes
	var timeDiffs []time.Duration
	for i := 1; i < len(sw.writes); i++ {
		diff := sw.writes[i].time.Sub(sw.writes[i-1].time)
		timeDiffs = append(timeDiffs, diff)
	}

	// At least one time difference should be >= 50ms (half of our sleep time)
	hasDelay := hasSignificantDelay(timeDiffs)
	if !hasDelay {
		t.Error("Expected streaming output with delays, but all writes happened too quickly")
		for i, diff := range timeDiffs {
			t.Logf("Time diff %d: %v", i, diff)
		}
	}
}

func hasSignificantDelay(timeDiffs []time.Duration) bool {
	for _, diff := range timeDiffs {
		if diff >= 50*time.Millisecond {
			return true
		}
	}
	return false
}
