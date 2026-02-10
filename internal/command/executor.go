package command

// executor implements CommandExecutor interface
type executor struct {
	shell ShellExecutor
}

// NewExecutor creates a new command executor with the given shell executor
func NewExecutor(shell ShellExecutor) Executor {
	return &executor{
		shell: shell,
	}
}

// NewRealExecutor creates a new command executor with real shell execution
func NewRealExecutor() Executor {
	return &executor{
		shell: NewRealShellExecutor(),
	}
}

// Execute executes the given commands in sequence and returns the results
func (e *executor) Execute(commands []Command) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Results: make([]Result, 0, len(commands)),
	}

	for _, cmd := range commands {
		output, err := e.shell.Execute(cmd.Name, cmd.Args, cmd.WorkDir, cmd.Interactive)

		commandResult := Result{
			Command: cmd,
			Output:  output,
			Error:   err,
		}

		result.Results = append(result.Results, commandResult)
	}

	return result, nil
}
