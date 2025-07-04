package command

// Command represents a shell command to be executed
type Command struct {
	Name    string   // Command name (e.g., "git")
	Args    []string // Command arguments
	WorkDir string   // Optional working directory
}

// CommandResult represents the result of a single command execution
type CommandResult struct {
	Command Command
	Output  string
	Error   error
}

// ExecutionResult represents the result of executing multiple commands
type ExecutionResult struct {
	Results []CommandResult
}

// ShellExecutor interface abstracts the actual command execution
type ShellExecutor interface {
	Execute(name string, args []string, workDir string) (string, error)
}

// CommandExecutor interface defines how commands are executed
type CommandExecutor interface {
	Execute(commands []Command) (*ExecutionResult, error)
}