package git

import (
	"fmt"
	"path/filepath"
)

type Worktree struct {
	Path   string
	Branch string
	HEAD   string
}

func (w *Worktree) Name() string {
	return filepath.Base(w.Path)
}

func (w *Worktree) String() string {
	if w.Branch != "" {
		return fmt.Sprintf("%s [%s]", w.Path, w.Branch)
	}
	return fmt.Sprintf("%s [%s]", w.Path, w.HEAD)
}