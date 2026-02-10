package command

// Command represents a shell command to be executed
type Command struct {
	Name        string // Command name (e.g., "git")
	Args        []string
	WorkDir     string // Optional working directory
	Interactive bool   // Prefer direct stdio wiring for interactive commands
}

// Result represents the result of a single command execution
type Result struct {
	Command Command
	Output  string
	Error   error
}

// ExecutionResult represents the result of executing multiple commands
type ExecutionResult struct {
	Results []Result
}

// ShellExecutor interface abstracts the actual command execution
type ShellExecutor interface {
	Execute(name string, args []string, workDir string, interactive bool) (string, error)
}

// Executor interface defines how commands are executed
type Executor interface {
	Execute(commands []Command) (*ExecutionResult, error)
}
