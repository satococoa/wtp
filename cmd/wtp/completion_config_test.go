package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeCompletionArgs(t *testing.T) {
	t.Run("converts trailing sentinel to single hyphen", func(t *testing.T) {
		args := []string{"wtp", "remove", "target", "--", "--generate-shell-completion"}
		got := normalizeCompletionArgs(args)
		want := []string{"wtp", "remove", "target", "-", "--generate-shell-completion"}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("normalizeCompletionArgs() = %v, want %v", got, want)
		}
	})

	t.Run("keeps completion flag before positional arguments", func(t *testing.T) {
		args := []string{"wtp", "remove", "--generate-shell-completion", "target"}
		got := normalizeCompletionArgs(args)
		want := []string{"wtp", "remove", "--generate-shell-completion", "target"}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("normalizeCompletionArgs() = %v, want %v", got, want)
		}
	})

	t.Run("keeps arguments untouched when no normalization is needed", func(t *testing.T) {
		args := []string{"wtp", "remove", "target", "--generate-shell-completion"}
		got := normalizeCompletionArgs(args)

		if !reflect.DeepEqual(got, args) {
			t.Fatalf("normalizeCompletionArgs() = %v, want %v", got, args)
		}
	})
}

func TestPatchCompletionScriptFishUsesDynamicHelper(t *testing.T) {
	original := `# wtp fish shell completion
complete -c wtp -a 'add'
`

	result := patchCompletionScript("fish", original)

	if !strings.Contains(result, "function __fish_wtp_dynamic_complete") {
		t.Fatalf("expected dynamic helper function in fish script, got:\n%s", result)
	}

	if !strings.Contains(result, "complete -c wtp -f -a '(__fish_wtp_dynamic_complete)'") {
		t.Fatalf("expected completion registration to use dynamic helper, got:\n%s", result)
	}

	if !strings.Contains(result, "string split -m 1 \":\" --") {
		t.Fatalf("expected fish script to strip description suffixes, got:\n%s", result)
	}

	if strings.Contains(result, "__fish_wtp_no_subcommand") {
		t.Fatalf("expected legacy static completion helpers to be removed, got:\n%s", result)
	}
}

func TestPatchCompletionScriptBashSanitizesDescriptions(t *testing.T) {
	result := patchCompletionScript("bash", "#!/bin/bash\n\n__wtp_bash_autocomplete() {\n    opts=$(eval \"${requestComp}\" 2>/dev/null)\n    COMPREPLY=($(compgen -W \"${opts}\" -- ${cur}))\n}\n")

	if !strings.Contains(result, "_wtp_sanitize_completion_list") {
		t.Fatalf("expected sanitize helper to be injected, got:\n%s", result)
	}

	if !strings.Contains(result, "opts=$(_wtp_sanitize_completion_list <<<\"${opts}\")") {
		t.Fatalf("expected bash script to sanitize completion output, got:\n%s", result)
	}
}

func TestPatchCompletionScriptPassthroughForOtherShells(t *testing.T) {
	original := "original-script"

	if got := patchCompletionScript("zsh", original); got != original {
		t.Fatalf("expected zsh completions to be unchanged, got %q", got)
	}

	if got := patchCompletionScript("unknown", original); got != original {
		t.Fatalf("expected unknown shell completions to fall back to original, got %q", got)
	}
}
