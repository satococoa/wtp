package main

import (
	"reflect"
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

	t.Run("moves completion flag to end when necessary", func(t *testing.T) {
		args := []string{"wtp", "remove", "--generate-shell-completion", "target"}
		got := normalizeCompletionArgs(args)
		want := []string{"wtp", "remove", "target", "--generate-shell-completion"}

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
