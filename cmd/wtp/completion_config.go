package main

import (
	"bytes"
	"context"
	"os"
	"strings"

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
	case "fish":
		return buildFishCompletionScript()
	case "bash":
		return patchBashCompletionScript(script)
	case "zsh":
		return patchZshCompletionScript(script)
	default:
		return script
	}
}

func patchZshCompletionScript(script string) string {
	if strings.Contains(script, "WTP_SHELL_COMPLETION=1") {
		return script
	}

	currentReplacement := "opts=(\"${(@f)$(env WTP_SHELL_COMPLETION=1 ${words[@]:0:#words[@]-1} " +
		"${current} --generate-shell-completion)}\")"
	subcommandReplacement := "opts=(\"${(@f)$(env WTP_SHELL_COMPLETION=1 ${words[@]:0:#words[@]-1} " +
		"--generate-shell-completion)}\")"

	replacements := []struct {
		target      string
		replacement string
	}{
		{
			target:      `opts=("${(@f)$(${words[@]:0:#words[@]-1} ${current} --generate-shell-completion)}")`,
			replacement: currentReplacement,
		},
		{
			target:      `opts=("${(@f)$(${words[@]:0:#words[@]-1} --generate-shell-completion)}")`,
			replacement: subcommandReplacement,
		},
	}

	for _, r := range replacements {
		script = strings.Replace(script, r.target, r.replacement, 1)
	}

	return script
}

func buildFishCompletionScript() string {
	return `# wtp fish shell completion

function __fish_wtp_dynamic_complete --description 'wtp dynamic completion helper'
	set -l tokens (commandline -opc)
	set -l args
	set -l token_count (count $tokens)
	if test $token_count -gt 1
		set args $tokens[2..-1]
	end

	set -l current (commandline -ct)

	if test -n "$current"
		if string match -q -- '-*' $current
			set args $args $current
		end
	end

	set args $args --generate-shell-completion

	if not command -sq wtp
		return
	end

	set -l raw (env WTP_SHELL_COMPLETION=1 command wtp $args)
	for line in $raw
		if test -z "$line"
			continue
		end

		set -l parts (string split -m 1 ":" -- $line)
		if test (count $parts) -gt 1
			set -l remainder $parts[2]
			if string match -q "* *" $remainder
				echo $parts[1]
				continue
			end
		end

		echo $line
	end
end

complete -c wtp -f -a '(__fish_wtp_dynamic_complete)'
`
}

func patchBashCompletionScript(script string) string {
	if strings.Contains(script, "_wtp_sanitize_completion_list") {
		return script
	}

	const helper = `
_wtp_sanitize_completion_list() {
	local line suffix result=()
	while IFS= read -r line; do
		if [[ "$line" == *:* ]]; then
			suffix=${line#*:}
			if [[ "$suffix" == *" "* ]]; then
				result+=("${line%%:*}")
				continue
			fi
		fi
		result+=("$line")
	done
	printf "%s\n" "${result[@]}"
}
`

	script = strings.Replace(script, "__wtp_bash_autocomplete() {", helper+"\n__wtp_bash_autocomplete() {", 1)

	const target = "    opts=$(eval \"${requestComp}\" 2>/dev/null)\n    COMPREPLY=($(compgen -W \"${opts}\" -- ${cur}))"
	const replacement = "    opts=$(WTP_SHELL_COMPLETION=1 eval \"${requestComp}\" 2>/dev/null)\n" +
		"    opts=$(_wtp_sanitize_completion_list <<<\"${opts}\")\n" +
		"    COMPREPLY=($(compgen -W \"${opts}\" -- ${cur}))"

	script = strings.Replace(script, target, replacement, 1)
	return script
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

	if flagIndex > 0 && normalized[flagIndex-1] == "--" && inShellCompletionContext() {
		normalized[flagIndex-1] = "-"
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

func inShellCompletionContext() bool {
	if os.Getenv("WTP_SHELL_COMPLETION") != "" {
		return true
	}
	if os.Getenv("COMP_LINE") != "" {
		return true
	}
	if os.Getenv("COMP_POINT") != "" {
		return true
	}
	return false
}
