package hooks

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/satococoa/wtp/internal/config"
)

const (
	directoryPermissions = 0o755
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
	if !e.config.HasHooks() {
		return nil
	}

	for i, hook := range e.config.Hooks.PostCreate {
		if err := e.executeHookWithWriter(w, &hook, worktreePath); err != nil {
			return fmt.Errorf("failed to execute hook %d: %w", i+1, err)
		}
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
	if runtime.GOOS == "windows" {
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

	// Set environment variables
	cmd.Env = os.Environ()
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

	// Connect stdout and stderr to the writer for streaming output
	cmd.Stdout = w
	cmd.Stderr = w

	// Execute command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %s", err)
	}

	return nil
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
