package main

import (
	"os"
	"path/filepath"
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
	const originalBashAutocomplete = "#!/bin/bash\n\n__wtp_bash_autocomplete() {\n" +
		"    opts=$(eval \"${requestComp}\" 2>/dev/null)\n" +
		"    COMPREPLY=($(compgen -W \"${opts}\" -- ${cur}))\n}\n"

	result := patchCompletionScript("bash", originalBashAutocomplete)

	if !strings.Contains(result, "_wtp_sanitize_completion_list") {
		t.Fatalf("expected sanitize helper to be injected, got:\n%s", result)
	}

	if !strings.Contains(result, "opts=$(_wtp_sanitize_completion_list <<<\"${opts}\")") {
		t.Fatalf("expected bash script to sanitize completion output, got:\n%s", result)
	}
}

func TestPatchCompletionScriptBashMatchesGolden(t *testing.T) {
	input := readCompletionTestdata(t, "bash_input.sh")
	got := patchCompletionScript("bash", input)
	assertCompletionGolden(t, "bash_expected.sh", got)
}

func TestPatchCompletionScriptFishMatchesGolden(t *testing.T) {
	got := patchCompletionScript("fish", "ignored")
	assertCompletionGolden(t, "fish_expected.fish", got)
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

func readCompletionTestdata(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", "completion", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read testdata %s: %v", name, err)
	}
	return string(data)
}

func assertCompletionGolden(t *testing.T, name, got string) {
	t.Helper()
	expected := readCompletionTestdata(t, name)
	if got == expected {
		return
	}

	if os.Getenv("UPDATE_COMPLETION_GOLDEN") != "" {
		writeCompletionTestdata(t, name, got)
		return
	}

	t.Fatalf("completion script %s mismatch (expected len %d, got %d):\n--- expected ---\n%s\n--- got ---\n%s",
		name, len(expected), len(got), expected, got)
}

func writeCompletionTestdata(t *testing.T, name, content string) {
	t.Helper()
	path := filepath.Join("testdata", "completion", name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write testdata %s: %v", name, err)
	}
}
