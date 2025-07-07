package command

// GitWorktreeAddOptions represents options for git worktree add command
type GitWorktreeAddOptions struct {
	Force  bool
	Detach bool
	Branch string
	Track  string
}

// GitWorktreeAdd builds a git worktree add command
func GitWorktreeAdd(path, commitish string, opts GitWorktreeAddOptions) Command {
	args := []string{"worktree", "add"}

	// Add flags
	if opts.Force {
		args = append(args, "--force")
	}
	if opts.Detach {
		args = append(args, "--detach")
	}
	if opts.Branch != "" {
		args = append(args, "-b", opts.Branch)
	}
	if opts.Track != "" {
		args = append(args, "--track")
		if !opts.Detach && opts.Branch == "" {
			// When tracking without explicit branch, create branch with same name
			args = append(args, "-b", extractBranchName(commitish))
		}
	}

	// Add path
	args = append(args, path)

	// Add commitish only if not creating a new branch or if explicitly provided
	if opts.Branch == "" && commitish != "" {
		args = append(args, commitish)
	}

	return Command{
		Name: "git",
		Args: args,
	}
}

// GitBranchDelete builds a git branch delete command
func GitBranchDelete(branchName string, force bool) Command {
	args := []string{"branch"}

	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}

	args = append(args, branchName)

	return Command{
		Name: "git",
		Args: args,
	}
}

// GitWorktreeRemove builds a git worktree remove command
func GitWorktreeRemove(path string, force bool) Command {
	args := []string{"worktree", "remove"}

	if force {
		args = append(args, "--force")
	}

	args = append(args, path)

	return Command{
		Name: "git",
		Args: args,
	}
}

// GitWorktreeList builds a git worktree list command
func GitWorktreeList() Command {
	return Command{
		Name: "git",
		Args: []string{"worktree", "list", "--porcelain"},
	}
}

// extractBranchName extracts branch name from a remote reference
// e.g., "origin/feature" -> "feature"
func extractBranchName(ref string) string {
	// Simple implementation - in real code this might be more sophisticated
	if ref == "" {
		return ref
	}

	// If it contains a slash, take the last part
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '/' {
			return ref[i+1:]
		}
	}

	return ref
}
