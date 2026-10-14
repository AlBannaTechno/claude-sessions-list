// Package app is what the window calls: every exported method of Service is bound to the frontend by Wails.
package app

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/AlBannaTechno/claude-sessions-list/internal/sessions"
)

// Host is what the service needs from the desktop around it; tests give it a fake.
type Host interface {
	OpenURL(url string) error
	SetClipboard(text string) error
	ShowFolder(path string) error
}

type Service struct {
	index *sessions.Index
	host  Host
}

func NewService(index *sessions.Index, host Host) *Service {
	return &Service{index: index, host: host}
}

// ListGroups reads what changed since the last call and returns every session grouped by folder.
func (s *Service) ListGroups() (sessions.Listing, error) {
	return s.index.List()
}

// LoadTranscript returns a session's conversation.
func (s *Service) LoadTranscript(id string) (sessions.Transcript, error) {
	p, ok := s.index.TranscriptPath(id)
	if !ok {
		if _, err := s.index.List(); err != nil {
			return sessions.Transcript{ID: id, Entries: []sessions.Entry{}}, err
		}
		if p, ok = s.index.TranscriptPath(id); !ok {
			return sessions.Transcript{ID: id, Entries: []sessions.Entry{}}, fmt.Errorf("session %s has no transcript on this computer", id)
		}
	}
	return sessions.LoadTranscript(p, id)
}

// DesktopLink opens a session in the Claude desktop app.
func DesktopLink(localID string) string { return "claude://claude.ai/epitaxy/" + localID }

// OpenInDesktop opens the session in the Claude desktop app.
func (s *Service) OpenInDesktop(localID string) error {
	if !strings.HasPrefix(localID, "local_") {
		return errors.New("this session was started from the terminal, so the Claude app does not have it")
	}
	return s.host.OpenURL(DesktopLink(localID))
}

// ResumeCommand is the terminal command that continues a session in the CLI, for the shell this system opens by
// default: zsh or bash on macOS and Linux, PowerShell on Windows. The PowerShell form also runs in the PowerShell 5.1
// that comes with Windows, which has no "&&".
func ResumeCommand(goos, folder, id string) string {
	resume := "claude --resume " + id
	switch {
	case folder == "":
		return resume
	case goos == "windows":
		return "Set-Location -LiteralPath '" + strings.ReplaceAll(folder, "'", "''") + "'; " + resume
	default:
		return "cd '" + strings.ReplaceAll(folder, "'", `'\''`) + "' && " + resume
	}
}

// CopyResumeCommand puts the resume command on the clipboard and returns it.
func (s *Service) CopyResumeCommand(folder, id string) (string, error) {
	cmd := ResumeCommand(runtime.GOOS, folder, id)
	return cmd, s.host.SetClipboard(cmd)
}

// CopyText puts any text on the clipboard: a session id, a folder path.
func (s *Service) CopyText(text string) error { return s.host.SetClipboard(text) }

// RevealFolder shows a session's folder in the system's file manager (Finder, File Explorer).
func (s *Service) RevealFolder(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("the folder %s no longer exists", path)
		}
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s is not a folder", path)
	}
	return s.host.ShowFolder(path)
}
