package errors

import (
	"errors"
	"fmt"
	"strings"
)

// Common error messages with helpful context and suggestions

// Git Repository Errors
func NotInGitRepository() error {
	msg := `not in a git repository

Solutions:
  • Run 'git init' to create a new repository
  • Navigate to an existing git repository
  • Check if you're in the correct directory`
	return errors.New(msg)
}

func GitCommandFailed(command, output string) error {
	// Clean up the output for better readability
	cleanOutput := strings.TrimSpace(output)
	if cleanOutput == "" {
		cleanOutput = "no additional details available"
	}

	msg := fmt.Sprintf(`git command failed: %s

Details: %s

Tip: Try running the git command manually to see the full error`, command, cleanOutput)
	return errors.New(msg)
}

// Validation Errors
func BranchNameRequired(commandExample string) error {
	msg := fmt.Sprintf(`branch name is required

Usage: %s

Examples:
  • wtp add feature/auth
  • wtp add -b new-feature
  • wtp add --track origin/main main`, commandExample)
	return errors.New(msg)
}

func WorktreeNameRequired() error {
	msg := `worktree name is required

Usage: wtp cd <worktree-name>

Tip: Run 'wtp list' to see available worktrees`
	return errors.New(msg)
}

func WorktreeNameRequiredForRemove() error {
	msg := `worktree name is required

Usage: wtp remove <worktree-name>

Examples:
  • wtp remove feature/auth
  • wtp remove --with-branch feature/auth
  • wtp remove --force feature/auth

Tip: Run 'wtp list' to see available worktrees`
	return errors.New(msg)
}

func InvalidBranchName(branchName string) error {
	msg := fmt.Sprintf(`invalid branch name: '%s'

Branch names cannot contain:
  • '..' (double dots)
  • Newline characters
  • Control characters

See 'git check-ref-format --help' for full rules`, branchName)
	return errors.New(msg)
}

// Worktree Errors
func WorktreeNotFound(name string, availableWorktrees []string) error {
	msg := fmt.Sprintf("worktree '%s' not found", name)

	if len(availableWorktrees) > 0 {
		msg += "\n\nAvailable worktrees:"
		for _, wt := range availableWorktrees {
			msg += fmt.Sprintf("\n  • %s", wt)
		}
	} else {
		msg += "\n\nNo worktrees found."
	}

	msg += "\n\nTip: Run 'wtp list' to see all worktrees"
	return errors.New(msg)
}

func WorktreeCreationFailed(path, branch string, gitError error) error {
	msg := fmt.Sprintf("failed to create worktree at '%s' for branch '%s'", path, branch)

	// Add specific suggestions based on common git worktree errors
	gitErrorStr := gitError.Error()
	if strings.Contains(gitErrorStr, "already checked out") {
		msg += `

Cause: Branch is already checked out in another worktree
Solution: Use '--force' flag to allow multiple checkouts, or choose a different branch`
	} else if strings.Contains(gitErrorStr, "not a valid object name") {
		msg += `

Cause: Branch or commit does not exist
Solutions:
  • Check the branch name spelling
  • Use 'git branch -a' to see available branches
  • Create the branch first with 'wtp add -b <branch-name>'`
	} else if strings.Contains(gitErrorStr, "destination path") && strings.Contains(gitErrorStr, "already exists") {
		msg += `

Cause: Target directory already exists
Solutions:
  • Remove the existing directory
  • Use a different branch name`
	}

	msg += fmt.Sprintf("\n\nOriginal error: %v", gitError)
	return errors.New(msg)
}

func WorktreeRemovalFailed(path string, gitError error) error {
	msg := fmt.Sprintf("failed to remove worktree at '%s'", path)

	errorStr := gitError.Error()
	suggestions := []string{}

	if strings.Contains(errorStr, "not a working tree") {
		suggestions = append(suggestions,
			"Check if the worktree path is correct",
			"Run 'wtp list' to see available worktrees")
	} else if strings.Contains(errorStr, "contains modified or untracked files") {
		suggestions = append(suggestions,
			"Commit or stash changes in the worktree first",
			"Use '--force' flag to remove anyway")
	} else if strings.Contains(errorStr, "locked") {
		suggestions = append(suggestions,
			"Unlock the worktree first with 'git worktree unlock'",
			"Use '--force' flag to remove anyway")
	} else if strings.Contains(errorStr, "permission denied") {
		suggestions = append(suggestions,
			"Check file permissions for the worktree directory",
			"Ensure you have write access to the parent directory")
	}

	if len(suggestions) > 0 {
		msg += "\n\nSuggestions:"
		for _, suggestion := range suggestions {
			msg += fmt.Sprintf("\n  • %s", suggestion)
		}
	}

	msg += fmt.Sprintf("\n\nOriginal error: %v", gitError)
	return errors.New(msg)
}

func CannotRemoveCurrentWorktree(worktreeName, path string) error {
	msg := fmt.Sprintf("cannot remove worktree '%s' while you are currently inside it", worktreeName)
	msg += fmt.Sprintf("\n\nCurrent location: %s", path)
	msg += "\n\nTip: Run 'wtp cd @' or 'wtp cd <another-worktree>' to switch before removing."
	return errors.New(msg)
}

func BranchRemovalFailed(branchName string, gitError error, isForced bool) error {
	msg := fmt.Sprintf("failed to remove branch '%s'", branchName)

	errorStr := gitError.Error()
	if !isForced && strings.Contains(errorStr, "not fully merged") {
		msg += `

Cause: Branch is not fully merged
Solution: Use '--force-branch' to delete anyway, or merge the branch first`
	} else if strings.Contains(errorStr, "not found") {
		msg += `

Cause: Branch does not exist
Tip: Run 'git branch' to see available branches`
	} else if strings.Contains(errorStr, "checked out") {
		msg += `

Cause: Branch is currently checked out
Solution: Switch to a different branch first`
	}

	msg += fmt.Sprintf("\n\nOriginal error: %v", gitError)
	return errors.New(msg)
}

// Configuration Errors
func ConfigLoadFailed(configPath string, parseError error) error {
	msg := fmt.Sprintf("failed to load configuration from '%s'", configPath)

	parseErrorStr := parseError.Error()
	if strings.Contains(parseErrorStr, "yaml") || strings.Contains(parseErrorStr, "unmarshal") {
		msg += `

Cause: YAML syntax error in configuration file
Solutions:
  • Check YAML syntax and indentation
  • Validate YAML at https://yamllint.com/
  • Run 'wtp init' to recreate the configuration`
	} else if strings.Contains(parseErrorStr, "no such file") {
		msg += `

Cause: Configuration file does not exist
Solution: Run 'wtp init' to create a configuration file`
	} else if strings.Contains(parseErrorStr, "permission denied") {
		msg += `

Cause: Permission denied reading configuration file
Solution: Check file permissions with 'ls -la .wtp.yml'`
	}

	msg += fmt.Sprintf("\n\nOriginal error: %v", parseError)
	return errors.New(msg)
}

func ConfigAlreadyExists(configPath string) error {
	msg := fmt.Sprintf(`configuration file already exists: %s

Options:
  • Edit the existing file manually
  • Delete it and run 'wtp init' again
  • Use 'wtp init --force' to overwrite (if that flag exists)`, configPath)
	return errors.New(msg)
}

// File System Errors
func DirectoryAccessFailed(operation, path string, originalError error) error {
	msg := fmt.Sprintf("failed to %s directory: %s", operation, path)

	errorStr := originalError.Error()
	if strings.Contains(errorStr, "permission denied") {
		msg += `

Cause: Permission denied
Solutions:
  • Check directory permissions
  • Run with appropriate privileges
  • Ensure you own the directory`
	} else if strings.Contains(errorStr, "no such file or directory") {
		msg += `

Cause: Directory does not exist
Solutions:
  • Create the parent directory first
  • Check the path spelling
  • Use an absolute path`
	}

	msg += fmt.Sprintf("\n\nOriginal error: %v", originalError)
	return errors.New(msg)
}

// Shell Integration Errors
func ShellIntegrationRequired() error {
	msg := `cd command requires shell integration

Setup:
  • Homebrew users: press TAB after typing 'wtp' once (automatic)
  • Other installs: eval "$(wtp shell-init <shell>)" in your shell profile (~/.bashrc, ~/.zshrc, etc.)

Help: Run 'wtp shell-init --help' for more details`
	return errors.New(msg)
}

func UnsupportedShell(shell string, supportedShells []string) error {
	msg := fmt.Sprintf("unsupported shell: %s", shell)

	if len(supportedShells) > 0 {
		msg += "\n\nSupported shells:"
		for _, s := range supportedShells {
			msg += fmt.Sprintf("\n  • %s", s)
		}
	}

	msg += "\n\nWorkaround: You can still use wtp without shell integration"
	return errors.New(msg)
}

// Branch Resolution Errors
func BranchNotFound(branchName string) error {
	msg := fmt.Sprintf(`branch '%s' not found in local or remote branches

Suggestions:
  • Check the branch name spelling
  • Run 'git branch -a' to see all branches
  • Create a new branch with 'wtp add -b %s'
  • Fetch latest changes with 'git fetch'`, branchName, branchName)
	return errors.New(msg)
}

func MultipleBranchesFound(branchName string, remotes []string) error {
	msg := fmt.Sprintf("branch '%s' exists in multiple remotes: %s", branchName, strings.Join(remotes, ", "))
	msg += fmt.Sprintf(`

Solution: Specify the remote explicitly:
  • wtp add --track %s/%s %s`, remotes[0], branchName, branchName)

	if len(remotes) > 1 {
		msg += fmt.Sprintf("\n  • wtp add --track %s/%s %s", remotes[1], branchName, branchName)
	}

	return errors.New(msg)
}

// Hook Errors
func HookExecutionFailed(hookIndex int, hookType string, originalError error) error {
	msg := fmt.Sprintf("failed to execute %s hook #%d", hookType, hookIndex+1)

	errorStr := originalError.Error()
	if strings.Contains(errorStr, "permission denied") {
		msg += `

Cause: Permission denied
Solutions:
  • Check file permissions
  • Ensure the command is executable
  • Check source/destination path permissions`
	} else if strings.Contains(errorStr, "no such file") {
		msg += `

Cause: File or command not found
Solutions:
  • Check file paths in .wtp.yml
  • Ensure the command exists in PATH
  • Use absolute paths for files`
	} else if strings.Contains(errorStr, "command not found") {
		msg += `

Cause: Command not found
Solutions:
  • Install the required command
  • Check command spelling in .wtp.yml
  • Use full path to command`
	}

	msg += fmt.Sprintf("\n\nOriginal error: %v", originalError)
	return errors.New(msg)
}
