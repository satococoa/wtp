package command

// executor implements CommandExecutor interface
type executor struct {
	shell ShellExecutor
}

// NewCommandExecutor creates a new command executor with the given shell executor
func NewCommandExecutor(shell ShellExecutor) CommandExecutor {
	return &executor{
		shell: shell,
	}
}

// Execute executes the given commands in sequence and returns the results
func (e *executor) Execute(commands []Command) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Results: make([]CommandResult, 0, len(commands)),
	}

	for _, cmd := range commands {
		output, err := e.shell.Execute(cmd.Name, cmd.Args, cmd.WorkDir)
		
		commandResult := CommandResult{
			Command: cmd,
			Output:  output,
			Error:   err,
		}
		
		result.Results = append(result.Results, commandResult)
	}

	return result, nil
}