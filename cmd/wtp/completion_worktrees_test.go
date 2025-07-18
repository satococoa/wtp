package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/satococoa/wtp/internal/git"
)

func TestPrintWorktriesForCd(t *testing.T) {
	tests := []struct {
		name      string
		worktrees []git.Worktree
		currentWt string
		want      []string
	}{
		{
			name: "shows @ for main worktree and * for current",
			worktrees: []git.Worktree{
				{Path: "/repo", Branch: "main", IsMain: true},
				{Path: "/repo/worktrees/feature", Branch: "feature", IsMain: false},
				{Path: "/repo/worktrees/bugfix", Branch: "bugfix", IsMain: false},
			},
			currentWt: "/repo/worktrees/feature",
			want: []string{
				"@",
				"feature*",
				"bugfix",
			},
		},
		{
			name: "current is main worktree",
			worktrees: []git.Worktree{
				{Path: "/repo", Branch: "main", IsMain: true},
				{Path: "/repo/worktrees/feature", Branch: "feature", IsMain: false},
			},
			currentWt: "/repo",
			want: []string{
				"@*",
				"feature",
			},
		},
		{
			name: "detached HEAD worktree",
			worktrees: []git.Worktree{
				{Path: "/repo", Branch: "main", IsMain: true},
				{Path: "/repo/worktrees/detached", Branch: "", IsMain: false},
			},
			currentWt: "/repo",
			want: []string{
				"@*",
				"detached",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printWorktriesForCd(&buf, tt.worktrees, tt.currentWt)

			got := strings.Split(strings.TrimSpace(buf.String()), "\n")

			if len(got) != len(tt.want) {
				t.Errorf("got %d lines, want %d lines", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}

			for i, want := range tt.want {
				if got[i] != want {
					t.Errorf("line %d: got %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

func TestPrintWorktriesForRemove(t *testing.T) {
	tests := []struct {
		name      string
		worktrees []git.Worktree
		want      []string
	}{
		{
			name: "excludes main worktree and no markers",
			worktrees: []git.Worktree{
				{Path: "/repo", Branch: "main", IsMain: true},
				{Path: "/repo/worktrees/feature", Branch: "feature", IsMain: false},
				{Path: "/repo/worktrees/bugfix", Branch: "bugfix", IsMain: false},
			},
			want: []string{
				"feature",
				"bugfix",
			},
		},
		{
			name: "handles detached HEAD",
			worktrees: []git.Worktree{
				{Path: "/repo", Branch: "main", IsMain: true},
				{Path: "/repo/worktrees/detached", Branch: "", IsMain: false},
			},
			want: []string{
				"detached",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printWorktriesForRemove(&buf, tt.worktrees)

			got := strings.Split(strings.TrimSpace(buf.String()), "\n")

			// Handle empty case
			if tt.want[0] == "" && len(got) == 1 && got[0] == "" {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("got %d lines, want %d lines", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}

			for i, want := range tt.want {
				if got[i] != want {
					t.Errorf("line %d: got %q, want %q", i, got[i], want)
				}
			}
		})
	}
}
