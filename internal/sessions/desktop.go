package sessions

import (
	"encoding/json"
	"os"
	"time"
)

// desktopInfo is what the Claude desktop app keeps about one of its sessions, in
// ~/Library/Application Support/Claude/claude-code-sessions/<account>/<org>/local_<id>.json.
type desktopInfo struct {
	LocalID      string    `json:"localId"`
	CliSessionID string    `json:"cliSessionId"`
	Title        string    `json:"title"`
	Cwd          string    `json:"cwd"`
	OriginCwd    string    `json:"originCwd"`
	Model        string    `json:"model"`
	Archived     bool      `json:"archived"`
	Created      time.Time `json:"created"`
	LastActive   time.Time `json:"lastActive"`
}

// desktopFile is the file's own shape; only these fields are read, the rest (MCP settings, mostly) is skipped.
type desktopFile struct {
	SessionID      string `json:"sessionId"`
	CliSessionID   string `json:"cliSessionId"`
	Cwd            string `json:"cwd"`
	OriginCwd      string `json:"originCwd"`
	Title          string `json:"title"`
	Model          string `json:"model"`
	IsArchived     bool   `json:"isArchived"`
	CreatedAt      int64  `json:"createdAt"`
	LastActivityAt int64  `json:"lastActivityAt"`
}

func readDesktop(path string) (desktopInfo, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return desktopInfo{}, err
	}
	return parseDesktop(b)
}

func parseDesktop(b []byte) (desktopInfo, error) {
	var f desktopFile
	if err := json.Unmarshal(b, &f); err != nil {
		return desktopInfo{}, err
	}
	return desktopInfo{
		LocalID:      f.SessionID,
		CliSessionID: f.CliSessionID,
		Title:        f.Title,
		Cwd:          f.Cwd,
		OriginCwd:    f.OriginCwd,
		Model:        f.Model,
		Archived:     f.IsArchived,
		Created:      fromMillis(f.CreatedAt),
		LastActive:   fromMillis(f.LastActivityAt),
	}, nil
}

func fromMillis(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}
