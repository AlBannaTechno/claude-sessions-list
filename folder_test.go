package main

import (
	"path/filepath"
	"testing"
)

// Each system opens a folder with its own file manager; on Windows the exit code of explorer.exe is ignored, because it
// is 1 even when the folder opened.
func TestOpenFolderCommand(t *testing.T) {
	cases := []struct {
		goos, program string
		checkExit     bool
	}{
		{"darwin", "open", true},
		{"windows", "explorer", false},
		{"linux", "xdg-open", true},
	}
	for _, c := range cases {
		cmd, checkExit := openFolderCommand(c.goos, `C:\Users\o\projects`)
		if filepath.Base(cmd.Path) != c.program && cmd.Args[0] != c.program {
			t.Errorf("%s: runs %q, want %q", c.goos, cmd.Args[0], c.program)
		}
		if cmd.Args[len(cmd.Args)-1] != `C:\Users\o\projects` {
			t.Errorf("%s: folder passed as %q", c.goos, cmd.Args[len(cmd.Args)-1])
		}
		if checkExit != c.checkExit {
			t.Errorf("%s: checkExit %v, want %v", c.goos, checkExit, c.checkExit)
		}
	}
}
