# Claude Code Integration

This guide covers using `wtp` inside a [Claude Code](https://claude.ai/code) workflow — configuring worktrees to live inside the repository, automating environment setup, and coordinating with Claude's built-in `EnterWorktree` tool.

## Recommended .wtp.yml for AI-assisted repos

By default, wtp places worktrees one level above the repo root (`../worktrees/`). For AI agent workflows it is more practical to keep them inside the repository so agents always have a predictable, relative path to each worktree:

```yaml
version: "1.0"

defaults:
  base_dir: ".claude/worktrees"

hooks:
  post_create:
    - type: copy
      from: .env
      to: .env
```

With `base_dir: ".claude/worktrees"`, a branch named `feature/228-run-primitive` is created at `.claude/worktrees/feature/228-run-primitive/` relative to the repo root. The `.claude/` directory is already gitignored in most Claude Code setups, so worktrees stay out of version control automatically.

Add `.claude/worktrees/` to `.gitignore` if it is not already covered:

```
.claude/worktrees/
```

## Typical workflow with Claude Code

Claude Code's `/worktree` skill creates feature branches with `wtp add -b`, runs self-verification inside the worktree, and opens a pull request — all without leaving the main worktree. The shell hook (`eval "$(wtp shell-init zsh)"`) allows Claude to navigate between worktrees using `wtp cd`.

A minimal Claude Code skill step looks like:

```bash
# Cut a feature branch off the integration branch
wtp add -b feature/228-run-primitive

# Navigate into it
cd "$(wtp cd feature/228-run-primitive)"

# ... implement, test, commit ...

# Return to main worktree
wtp cd @

# After the PR is merged: clean up
wtp remove --with-branch feature/228-run-primitive
```

## EnterWorktree and wtp coexistence

Claude Code has a built-in `EnterWorktree` tool that creates isolated worktrees for sub-agents. Worktrees created this way are not tracked by wtp — they will appear in `git worktree list` but not behave like `wtp`-managed worktrees (no hooks, no `wtp cd` resolution by partial name).

If you use both tools in the same repo, keep them in separate directories so they do not interfere:

```yaml
defaults:
  base_dir: ".claude/worktrees"   # wtp-managed worktrees here
```

Claude Code's `EnterWorktree` tool defaults to a system temp path, which naturally stays separate.

## Multi-branch integration pattern

When multiple feature branches are ready to merge simultaneously, `wtp exec` lets you run the same verification command across all of them before merging:

```bash
wtp exec feature/228-run-primitive -- cargo check
wtp exec feature/231-text-input -- cargo check
```

This is useful in CI scripts or when an AI agent is verifying a batch of PRs before a single install step.
