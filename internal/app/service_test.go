package app

import (
	"strings"
	"testing"

	"github.com/AlBannaTechno/claude-sessions-list/internal/sessions"
)

type fakeHost struct{ opened, copied, revealed string }

func (f *fakeHost) OpenURL(u string) error      { f.opened = u; return nil }
func (f *fakeHost) SetClipboard(t string) error { f.copied = t; return nil }
func (f *fakeHost) ShowFolder(p string) error   { f.revealed = p; return nil }

func TestResumeCommandQuotesTheFolder(t *testing.T) {
	cases := []struct{ goos, folder, want string }{
		{"darwin", "/Users/me/Library/Application Support/it's", `cd '/Users/me/Library/Application Support/it'\''s' && claude --resume abc`},
		{"linux", "/home/o/x", `cd '/home/o/x' && claude --resume abc`},
		// PowerShell: a quote inside single quotes is written twice, and ";" (not "&&") also works in PowerShell 5.1.
		{"windows", `C:\Users\me\Sam's work`, `Set-Location -LiteralPath 'C:\Users\me\Sam''s work'; claude --resume abc`},
		{"darwin", "", "claude --resume abc"},
		{"windows", "", "claude --resume abc"},
	}
	for _, c := range cases {
		if got := ResumeCommand(c.goos, c.folder, "abc"); got != c.want {
			t.Errorf("%s %q:\ngot  %s\nwant %s", c.goos, c.folder, got, c.want)
		}
	}
}

func TestOpenInDesktop(t *testing.T) {
	h := &fakeHost{}
	s := NewService(sessions.NewIndex(sessions.Dirs{}), h)
	if err := s.OpenInDesktop("local_00000000"); err != nil || h.opened != "claude://claude.ai/epitaxy/local_00000000" {
		t.Errorf("opened %q, err %v", h.opened, err)
	}
	h.opened = ""
	if err := s.OpenInDesktop("00000000-0000"); err == nil || h.opened != "" {
		t.Errorf("a CLI session id must not open the desktop app; opened %q", h.opened)
	}
}

func TestRevealFolderRefusesAMissingFolder(t *testing.T) {
	h := &fakeHost{}
	s := NewService(sessions.NewIndex(sessions.Dirs{}), h)
	if err := s.RevealFolder("/no/such/folder"); err == nil || !strings.Contains(err.Error(), "no longer exists") || h.revealed != "" {
		t.Errorf("err %v, revealed %q", err, h.revealed)
	}
	dir := t.TempDir()
	if err := s.RevealFolder(dir); err != nil || h.revealed != dir {
		t.Errorf("err %v, revealed %q", err, h.revealed)
	}
}
