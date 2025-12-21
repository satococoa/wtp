// Package hooks handles executing hooks for worktrees.
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

// NewExecutor returns an Executor configured with cfg for hook configuration and repoRoot as the repository root used to resolve paths.
func NewExecutor(cfg *config.Config, repoRoot string) *Executor {
	return &Executor{
		config:   cfg,
		repoRoot: repoRoot,
	}
}

// ExecutePostCreateHooks executes all post-create hooks and streams output to writer.
// Relative paths are resolved from the worktree path.
func (e *Executor) ExecutePostCreateHooks(w io.Writer, worktreePath string) error {
	if e.config == nil || !e.config.HasPostCreateHooks() {
		return nil
	}
	return e.executeHooksWithWriter(
		w,
		e.config.Hooks.PostCreate,
		e.repoRoot,   // copy source base path
		worktreePath, // copy destination base path
		worktreePath, // command base path
	)
}

// ExecutePreRemoveHooks executes all pre-remove hooks and streams output to writer.
// Relative "from" paths resolve from the target worktree, while "to" paths resolve
// from the repository root.
func (e *Executor) ExecutePreRemoveHooks(w io.Writer, worktreePath string) error {
	if e.config == nil || !e.config.HasPreRemoveHooks() {
		return nil
	}
	return e.executeHooksWithWriter(
		w,
		e.config.Hooks.PreRemove,
		worktreePath, // copy source base path
		e.repoRoot,   // copy destination base path (dest)
		worktreePath, // command base path (execute in worktree)
	)
}

// ExecutePostRemoveHooks executes all post-remove hooks and streams output to writer.
// Relative paths are resolved from the worktree path unless configured otherwise.
func (e *Executor) ExecutePostRemoveHooks(w io.Writer, worktreePath string) error {
	if e.config == nil || !e.config.HasPostRemoveHooks() {
		return nil
	}
	return e.executeHooksWithWriter(
		w,
		e.config.Hooks.PostRemove,
		e.repoRoot,   // copy source base path
		worktreePath, // copy destination base path
		worktreePath, // command base path
	)
}

func (e *Executor) executeHooksWithWriter(
	w io.Writer,
	hooks []config.Hook,
	copySourceBasePath string,
	copyDestinationBasePath string,
	commandBasePath string,
) error {
	totalHooks := len(hooks)
	for i, hook := range hooks {
		// Log which hook is starting
		if _, err := fmt.Fprintf(w, "\n→ Running hook %d of %d...\n", i+1, totalHooks); err != nil {
			return err
		}

		if err := e.executeHookWithWriter(w, &hook, copySourceBasePath, copyDestinationBasePath, commandBasePath); err != nil {
			return fmt.Errorf("failed to execute hook %d: %w", i+1, err)
		}

		// Log successful completion
		if _, err := fmt.Fprintf(w, "✓ Hook %d completed\n", i+1); err != nil {
			return err
		}
	}

	return nil
}

// executeHookWithWriter executes a single hook with output directed to writer
func (e *Executor) executeHookWithWriter(
	w io.Writer,
	hook *config.Hook,
	copySourceBasePath string,
	copyDestinationBasePath string,
	commandBasePath string,
) error {
	switch hook.Type {
	case config.HookTypeCopy:
		return e.executeCopyHookWithWriter(w, hook, copySourceBasePath, copyDestinationBasePath)
	case config.HookTypeCommand:
		return e.executeCommandHookWithWriter(w, hook, commandBasePath)
	default:
		return fmt.Errorf("unknown hook type: %s", hook.Type)
	}
}

// executeCopyHookWithWriter executes a copy hook with output directed to writer
func (e *Executor) executeCopyHookWithWriter(w io.Writer, hook *config.Hook, sourceBasePath, destinationBasePath string) error {
	// Resolve source path (relative to source base path)
	srcPath := hook.From
	if !filepath.IsAbs(srcPath) {
		srcPath = filepath.Join(sourceBasePath, srcPath)
	}
	srcPath = filepath.Clean(srcPath)
	if !filepath.IsAbs(hook.From) {
		if err := ensureWithinBase(sourceBasePath, srcPath); err != nil {
			return err
		}
	}

	// Resolve destination path (relative to destination base path)
	dstPath := hook.To
	if !filepath.IsAbs(dstPath) {
		dstPath = filepath.Join(destinationBasePath, dstPath)
	}
	dstPath = filepath.Clean(dstPath)
	if !filepath.IsAbs(hook.To) {
		if err := ensureWithinBase(destinationBasePath, dstPath); err != nil {
			return err
		}
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
	relSrc, _ := filepath.Rel(sourceBasePath, srcPath)
	relDst, _ := filepath.Rel(destinationBasePath, dstPath)
	if _, err := fmt.Fprintf(w, "  Copying: %s → %s\n", relSrc, relDst); err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return e.copyDir(srcPath, dstPath)
	}
	return e.copyFile(srcPath, dstPath)
}

// ensureWithinBase verifies that target is located within base.
// It returns an error if resolving the relative path fails or if target escapes base (for example via ".." or a parent-directory prefix).
func ensureWithinBase(base, target string) error {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return fmt.Errorf("failed to resolve path %s relative to %s: %w", target, base, err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path %s escapes base directory %s", target, base)
	}

	return nil
}

// executeCommandHookWithWriter executes a command hook with output directed to writer
func (e *Executor) executeCommandHookWithWriter(w io.Writer, hook *config.Hook, basePath string) error {
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
		workDir = basePath
	} else if !filepath.IsAbs(workDir) {
		workDir = filepath.Join(basePath, workDir)
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
		fmt.Sprintf("GIT_WTP_WORKTREE_PATH=%s", basePath),
		fmt.Sprintf("GIT_WTP_REPO_ROOT=%s", e.repoRoot))

	// Log the command execution to writer
	if _, err := fmt.Fprintf(w, "  Running: %s", hook.Command); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

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
		return fmt.Errorf("command failed: %w", err)
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
func (*Executor) copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}

	if srcInfo.Mode().Perm()&0o400 == 0 {
		return fmt.Errorf("failed to copy file: source file is not readable")
	}

	// #nosec G304 -- src is validated against the repository root above
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	dstParent := filepath.Dir(dst)
	if writableErr := ensureDirWritable(dstParent); writableErr != nil {
		return fmt.Errorf("failed to create destination file: %w", writableErr)
	}

	// #nosec G304 -- dst is validated against the worktree path above
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		_ = destFile.Close()
	}()

	if _, copyErr := io.Copy(destFile, sourceFile); copyErr != nil {
		return fmt.Errorf("failed to copy file: %w", copyErr)
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

	if srcInfo.Mode().Perm()&0o400 == 0 {
		return fmt.Errorf("failed to read source directory: permission denied")
	}

	parentDir := filepath.Dir(dst)
	if writableErr := ensureDirWritable(parentDir); writableErr != nil {
		return fmt.Errorf("failed to create destination directory: %w", writableErr)
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

func ensureDirWritable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}

	if info.Mode().Perm()&0o222 == 0 {
		return fmt.Errorf("write permission denied for directory: %s", path)
	}

	return nil
}