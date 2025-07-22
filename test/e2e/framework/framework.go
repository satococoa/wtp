package framework

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	dirPerm  = 0755
	filePerm = 0600
)

type TestEnvironment struct {
	t         *testing.T
	tmpDir    string
	wtpBinary string
	cleanup   []func()
}

func NewTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	tmpDir := t.TempDir()
	env := &TestEnvironment{
		t:       t,
		tmpDir:  tmpDir,
		cleanup: []func(){},
	}

	env.buildWTP()

	return env
}

func (e *TestEnvironment) buildWTP() {
	e.t.Helper()

	wtpBinary := filepath.Join(e.tmpDir, "wtp")
	if runtime := os.Getenv("WTP_E2E_BINARY"); runtime != "" {
		wtpBinary = runtime
		if _, err := os.Stat(wtpBinary); err != nil {
			e.t.Fatalf("Specified WTP binary not found: %s", wtpBinary)
		}
	} else {
		projectRoot := e.findProjectRoot()
		cmd := exec.Command("go", "build", "-o", wtpBinary, "./cmd/wtp")
		cmd.Dir = projectRoot
		if output, err := cmd.CombinedOutput(); err != nil {
			e.t.Fatalf("Failed to build wtp binary: %v\nOutput: %s", err, output)
		}
	}

	// Validate the binary path
	wtpBinary = filepath.Clean(wtpBinary)
	if !filepath.IsAbs(wtpBinary) {
		absPath, err := filepath.Abs(wtpBinary)
		if err != nil {
			e.t.Fatalf("Failed to get absolute path for binary: %v", err)
		}
		wtpBinary = absPath
	}

	e.wtpBinary = wtpBinary
}

func (e *TestEnvironment) findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		e.t.Fatalf("Failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			e.t.Fatal("Could not find project root (go.mod)")
		}
		dir = parent
	}
}

func (e *TestEnvironment) CreateTestRepo(name string) *TestRepo {
	e.t.Helper()

	repoDir := filepath.Join(e.tmpDir, name)

	e.run("git", "init", repoDir)
	e.runInDir(repoDir, "git", "config", "user.name", "Test User")
	e.runInDir(repoDir, "git", "config", "user.email", "test@example.com")
	
	// Ensure the default branch is 'main' regardless of global git config
	e.runInDir(repoDir, "git", "config", "init.defaultBranch", "main")

	readmePath := filepath.Join(repoDir, "README.md")
	e.writeFile(readmePath, "# Test Repository")
	e.runInDir(repoDir, "git", "add", ".")
	e.runInDir(repoDir, "git", "commit", "-m", "Initial commit")
	
	// Explicitly rename the branch to main if it's not already
	e.runInDir(repoDir, "git", "branch", "-m", "main")

	return &TestRepo{
		env:  e,
		path: repoDir,
	}
}

func (e *TestEnvironment) run(command string, args ...string) string {
	e.t.Helper()

	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.t.Fatalf("Command failed: %s %s\nOutput: %s\nError: %v",
			command, strings.Join(args, " "), output, err)
	}
	return string(output)
}

func (e *TestEnvironment) runInDir(dir, command string, args ...string) string {
	e.t.Helper()

	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.t.Fatalf("Command failed in %s: %s %s\nOutput: %s\nError: %v",
			dir, command, strings.Join(args, " "), output, err)
	}
	return string(output)
}

func (e *TestEnvironment) writeFile(path, content string) {
	e.t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		e.t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), filePerm); err != nil {
		e.t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

func (e *TestEnvironment) RunWTP(args ...string) (string, error) {
	// Validate args don't contain dangerous characters
	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return "", fmt.Errorf("invalid argument: %w", err)
		}
	}

	// Create command with validated binary path
	cmd := createSafeCommand(e.wtpBinary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (e *TestEnvironment) TmpDir() string {
	return e.tmpDir
}

func (e *TestEnvironment) CreateNonRepoDir(name string) *TestRepo {
	e.t.Helper()

	dir := filepath.Join(e.tmpDir, name)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		e.t.Fatalf("Failed to create directory: %v", err)
	}

	return &TestRepo{
		env:  e,
		path: dir,
	}
}

func (e *TestEnvironment) WriteFile(path, content string) {
	e.writeFile(path, content)
}

func (e *TestEnvironment) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (e *TestEnvironment) RunInDir(dir, command string, args ...string) string {
	return e.runInDir(dir, command, args...)
}

func (e *TestEnvironment) Cleanup() {
	for _, fn := range e.cleanup {
		fn()
	}
}

type TestRepo struct {
	env  *TestEnvironment
	path string
}

func (r *TestRepo) RunWTP(args ...string) (string, error) {
	// Validate args don't contain dangerous characters
	for _, arg := range args {
		if err := validateArg(arg); err != nil {
			return "", fmt.Errorf("invalid argument: %w", err)
		}
	}

	// Create command with validated binary path
	cmd := createSafeCommand(r.env.wtpBinary, args...)
	cmd.Dir = r.path
	cmd.Env = append(os.Environ(), "HOME="+r.env.tmpDir)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (r *TestRepo) CreateBranch(name string) {
	r.env.runInDir(r.path, "git", "branch", name)
}

func (r *TestRepo) CheckoutBranch(name string) {
	r.env.runInDir(r.path, "git", "checkout", name)
}

func (r *TestRepo) CommitFile(filename, content, message string) {
	r.env.writeFile(filepath.Join(r.path, filename), content)
	r.env.runInDir(r.path, "git", "add", filename)
	r.env.runInDir(r.path, "git", "commit", "-m", message)
}

func (r *TestRepo) AddRemote(name, url string) {
	r.env.runInDir(r.path, "git", "remote", "add", name, url)
}

func (r *TestRepo) CreateRemoteBranch(remote, branch string) {
	refPath := filepath.Join(r.path, ".git", "refs", "remotes", remote)
	if err := os.MkdirAll(refPath, dirPerm); err != nil {
		r.env.t.Fatalf("Failed to create remote ref directory: %v", err)
	}

	output := r.env.runInDir(r.path, "git", "rev-parse", "HEAD")
	commit := strings.TrimSpace(output)

	r.env.writeFile(filepath.Join(refPath, branch), commit)
}

func (r *TestRepo) Path() string {
	return r.path
}

func (r *TestRepo) WriteConfig(content string) {
	configPath := filepath.Join(r.path, ".wtp.yml")
	r.env.writeFile(configPath, content)
}

func (r *TestRepo) HasFile(path string) bool {
	fullPath := filepath.Join(r.path, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

func (r *TestRepo) ReadFile(path string) string {
	fullPath := filepath.Join(r.path, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		r.env.t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

func (r *TestRepo) GitStatus() string {
	return r.env.runInDir(r.path, "git", "status", "--porcelain")
}

func (r *TestRepo) CurrentBranch() string {
	output := r.env.runInDir(r.path, "git", "branch", "--show-current")
	return strings.TrimSpace(output)
}

func (r *TestRepo) GetCommitHash() string {
	output := r.env.runInDir(r.path, "git", "rev-parse", "HEAD")
	return strings.TrimSpace(output)
}

func (r *TestRepo) GetBranchCommitHash(branch string) string {
	output := r.env.runInDir(r.path, "git", "rev-parse", branch)
	return strings.TrimSpace(output)
}

func (r *TestRepo) ListWorktrees() []string {
	output := r.env.runInDir(r.path, "git", "worktree", "list", "--porcelain")
	lines := strings.Split(strings.TrimSpace(output), "\n")

	var worktrees []string
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			worktrees = append(worktrees, strings.TrimPrefix(line, "worktree "))
		}
	}
	return worktrees
}

func WithTimeout(timeout time.Duration) func(cmd *exec.Cmd) {
	return func(cmd *exec.Cmd) {
		timer := time.AfterFunc(timeout, func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		})
		_ = cmd.Start()
		_ = cmd.Wait()
		timer.Stop()
	}
}

// validateArg checks if an argument is safe to pass to exec.Command
func validateArg(arg string) error {
	// Allow common flags and paths
	// This is a whitelist approach for test arguments
	if arg == "" {
		return nil
	}

	// Check for shell metacharacters that could be dangerous
	// Note: { and } are allowed for branch names like branch@{upstream}
	dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(arg, char) {
			return fmt.Errorf("argument contains potentially dangerous character: %s", char)
		}
	}

	return nil
}

// createSafeCommand creates an exec.Cmd with a validated binary path
func createSafeCommand(binary string, args ...string) *exec.Cmd {
	// The binary path has already been validated during initialization
	// This function separates the concern of command creation from validation
	return exec.Command(binary, args...)
}
