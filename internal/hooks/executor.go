package hooks

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/satococoa/wtp/v2/internal/config"
)

const (
	directoryPermissions = 0o755
	windowsOS            = "windows"
)

// Executor handles hook execution
type Executor struct {
	config   *config.Config
	repoRoot string
}

// NewExecutor creates a new hook executor
func NewExecutor(cfg *config.Config, repoRoot string) *Executor {
	return &Executor{
		config:   cfg,
		repoRoot: repoRoot,
	}
}

// ExecutePostCreateHooks executes all post-create hooks and streams output to writer
func (e *Executor) ExecutePostCreateHooks(w io.Writer, worktreePath string) error {
	if e.config == nil || !e.config.HasHooks() {
		return nil
	}

	totalHooks := len(e.config.Hooks.PostCreate)
	for i, hook := range e.config.Hooks.PostCreate {
		// Log which hook is starting
		fmt.Fprintf(w, "\n→ Running hook %d of %d...\n", i+1, totalHooks)

		if err := e.executeHookWithWriter(w, &hook, worktreePath); err != nil {
			return fmt.Errorf("failed to execute hook %d: %w", i+1, err)
		}

		// Log successful completion
		fmt.Fprintf(w, "✓ Hook %d completed\n", i+1)
	}

	return nil
}

// executeHookWithWriter executes a single hook with output directed to writer
func (e *Executor) executeHookWithWriter(w io.Writer, hook *config.Hook, worktreePath string) error {
	switch hook.Type {
	case config.HookTypeCopy:
		return e.executeCopyHookWithWriter(w, hook, worktreePath)
	case config.HookTypeCommand:
		return e.executeCommandHookWithWriter(w, hook, worktreePath)
	default:
		return fmt.Errorf("unknown hook type: %s", hook.Type)
	}
}

// executeCopyHookWithWriter executes a copy hook with output directed to writer
func (e *Executor) executeCopyHookWithWriter(w io.Writer, hook *config.Hook, worktreePath string) error {
	// Resolve source path (relative to repo root)
	srcPath := hook.From
	if !filepath.IsAbs(srcPath) {
		srcPath = filepath.Join(e.repoRoot, srcPath)
	}

	// Resolve destination path (relative to worktree)
	dstPath := hook.To
	if !filepath.IsAbs(dstPath) {
		dstPath = filepath.Join(worktreePath, dstPath)
	}

	// Check if source exists
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("source path does not exist: %s", srcPath)
	}

	// Create destination directory if needed
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, directoryPermissions); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Log the copy operation to writer
	relSrc, _ := filepath.Rel(e.repoRoot, srcPath)
	relDst, _ := filepath.Rel(worktreePath, dstPath)
	fmt.Fprintf(w, "  Copying: %s → %s\n", relSrc, relDst)

	if srcInfo.IsDir() {
		return e.copyDir(srcPath, dstPath)
	}
	return e.copyFile(srcPath, dstPath)
}

// executeCommandHookWithWriter executes a command hook with output directed to writer
func (e *Executor) executeCommandHookWithWriter(w io.Writer, hook *config.Hook, worktreePath string) error {
	// Execute command using shell for unified command format
	var cmd *exec.Cmd
	if runtime.GOOS == windowsOS {
		// #nosec G204 - Commands come from project configuration file controlled by developer
		cmd = exec.Command("cmd", "/c", hook.Command)
	} else {
		// #nosec G204 - Commands come from project configuration file controlled by developer
		cmd = exec.Command("sh", "-c", hook.Command)
	}

	// Set working directory
	workDir := hook.WorkDir
	if workDir == "" {
		workDir = worktreePath
	} else if !filepath.IsAbs(workDir) {
		workDir = filepath.Join(worktreePath, workDir)
	}
	cmd.Dir = workDir

	// Set environment variables (filter out WTP_SHELL_INTEGRATION)
	env := os.Environ()
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "WTP_SHELL_INTEGRATION=") {
			filtered = append(filtered, e)
		}
	}
	cmd.Env = filtered
	for key, value := range hook.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Add worktree-specific environment variables
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("GIT_WTP_WORKTREE_PATH=%s", worktreePath),
		fmt.Sprintf("GIT_WTP_REPO_ROOT=%s", e.repoRoot))

	// Log the command execution to writer
	fmt.Fprintf(w, "  Running: %s", hook.Command)
	fmt.Fprintln(w)

	// Create pipes for stdout and stderr to enable real-time streaming
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Stream output in real-time using goroutines
	const numStreams = 2 // stdout and stderr
	done := make(chan error, numStreams)

	synchronized := newSynchronizedWriter(w)

	go func() {
		_, err := io.Copy(synchronized, stdout)
		done <- err
	}()

	go func() {
		_, err := io.Copy(synchronized, stderr)
		done <- err
	}()

	// Wait for streaming to complete
	for i := 0; i < numStreams; i++ {
		if err := <-done; err != nil {
			return fmt.Errorf("output streaming failed: %w", err)
		}
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command failed: %s", err)
	}

	return nil
}

type synchronizedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func newSynchronizedWriter(w io.Writer) *synchronizedWriter {
	return &synchronizedWriter{w: w}
}

func (sw *synchronizedWriter) Write(p []byte) (int, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

// copyFile copies a single file
func (e *Executor) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, copyErr := io.Copy(destFile, sourceFile); copyErr != nil {
		return fmt.Errorf("failed to copy file: %w", copyErr)
	}

	// Copy file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// copyDir recursively copies a directory
func (e *Executor) copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory: %w", err)
	}

	if mkdirErr := os.MkdirAll(dst, srcInfo.Mode()); mkdirErr != nil {
		return fmt.Errorf("failed to create destination directory: %w", mkdirErr)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := e.copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := e.copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
