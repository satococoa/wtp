package main

import (
	"bytes"
	"context"
	"os"

	"github.com/urfave/cli/v3"
)

const completionFlag = "--generate-shell-completion"

func configureCompletionCommand(cmd *cli.Command) {
	originalAction := cmd.Action
	if originalAction == nil {
		return
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		writer := c.Writer
		if writer == nil {
			writer = os.Stdout
		}

		var buf bytes.Buffer
		c.Writer = &buf

		if err := originalAction(ctx, c); err != nil {
			c.Writer = writer
			return err
		}

		c.Writer = writer

		var shell string
		if args := c.Args(); args != nil && args.Len() > 0 {
			shell = args.First()
		}

		script := patchCompletionScript(shell, buf.String())
		_, err := writer.Write([]byte(script))
		return err
	}
}

func patchCompletionScript(shell, script string) string {
	switch shell {
	case "zsh":
		return script
	case "bash":
		return script
	default:
		return script
	}
}

func normalizeCompletionArgs(args []string) []string {
	flagIndex := -1
	for i, arg := range args {
		if arg == completionFlag {
			flagIndex = i
			break
		}
	}

	if flagIndex == -1 {
		return args
	}

	normalized := append([]string(nil), args...)

	if flagIndex > 0 && normalized[flagIndex-1] == "--" {
		normalized[flagIndex-1] = "-"
	}

	if flagIndex != len(normalized)-1 {
		normalized = append(normalized[:flagIndex], normalized[flagIndex+1:]...)
		normalized = append(normalized, completionFlag)
	}

	return normalized
}

func completionArgsFromCommand(cmd *cli.Command) (current string, previous []string) {
	if cmd == nil {
		return "", nil
	}

	if cmdArgs := cmd.Args(); cmdArgs != nil {
		args := filterCompletionArgs(cmdArgs.Slice())
		if len(args) == 0 {
			return "", nil
		}

		current = args[len(args)-1]
		if len(args) > 1 {
			previous = append([]string(nil), args[:len(args)-1]...)
		}

		return current, previous
	}

	return "", nil
}

func filterCompletionArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == completionFlag {
			continue
		}
		filtered = append(filtered, arg)
	}

	return filtered
}
