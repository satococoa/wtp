package main

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/urfave/cli/v3"
)

const (
	maxHyphenPrefix   = 2
	sentinelArgOffset = 2
)

func completeFlagSuggestions(cmd *cli.Command, current string) bool {
	if cmd == nil {
		return false
	}

	writer := commandWriter(cmd)

	trimmed, doubleDash := normalizeCurrent(current)
	if trimmed == "" && !strings.HasPrefix(current, "-") {
		return false
	}

	seen := make(map[string]struct{})
	emitted := false

	for _, flag := range cmd.Flags {
		if !isFlagVisible(flag) {
			continue
		}

		match := selectMatchingName(flag.Names(), trimmed, doubleDash)
		if match == "" {
			continue
		}

		completion := formatCompletion(match)
		if _, exists := seen[completion]; exists {
			continue
		}

		seen[completion] = struct{}{}
		fmt.Fprintln(writer, completion)
		emitted = true
	}

	return emitted
}

func commandWriter(cmd *cli.Command) io.Writer {
	writer := cmd.Root().Writer
	if writer == nil {
		return os.Stdout
	}
	return writer
}

func normalizeCurrent(current string) (trimmed string, doubleDash bool) {
	return strings.TrimLeft(current, "-"), strings.HasPrefix(current, "--")
}

func isFlagVisible(flag cli.Flag) bool {
	if visibility, ok := flag.(interface{ IsVisible() bool }); ok && !visibility.IsVisible() {
		return false
	}
	return true
}

func selectMatchingName(names []string, trimmed string, doubleDash bool) string {
	for _, candidate := range names {
		name := strings.TrimSpace(candidate)
		if name == "" {
			continue
		}

		if doubleDash && utf8.RuneCountInString(name) == 1 {
			continue
		}

		if trimmed != "" && !strings.HasPrefix(name, trimmed) {
			continue
		}

		if trimmed == name {
			continue
		}

		return name
	}

	return ""
}

func formatCompletion(name string) string {
	count := utf8.RuneCountInString(name)
	if count > maxHyphenPrefix {
		count = maxHyphenPrefix
	}
	return strings.Repeat("-", count) + name
}

func tryFlagCompletion(cmd *cli.Command, candidate string) bool {
	if strings.HasPrefix(candidate, "-") {
		return completeFlagSuggestions(cmd, candidate)
	}
	return false
}

func maybeCompleteFlagSuggestions(cmd *cli.Command, current string, previous []string) bool {
	currentNormalized := strings.TrimSuffix(current, "*")
	if currentNormalized != "" && tryFlagCompletion(cmd, currentNormalized) {
		return true
	}

	if len(previous) > 0 {
		last := strings.TrimSuffix(previous[len(previous)-1], "*")
		if last == "-" || last == "--" {
			// Sentinel separating flags from positionals; ignore for flag completion.
		} else if last != "" && last != currentNormalized && tryFlagCompletion(cmd, last) {
			return true
		}
	}

	if candidate, ok := flagCandidateFromOSArgs(); ok {
		if candidate != "" && candidate != currentNormalized && tryFlagCompletion(cmd, candidate) {
			return true
		}
	}

	return false
}

func flagCandidateFromOSArgs() (string, bool) {
	index := slices.Index(os.Args, completionFlag)
	if index <= 0 {
		return "", false
	}

	candidate := os.Args[index-1]
	if candidate == "-" || candidate == "--" {
		if index >= sentinelArgOffset {
			candidate = os.Args[index-2]
		} else {
			return "", false
		}
	}

	if candidate == completionFlag {
		return "", false
	}

	return candidate, candidate != ""
}
