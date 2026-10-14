package sessions

import (
	"sort"
	"strings"
	"time"
)

// ScratchKey is the one group for every session the desktop app started without a folder.
const ScratchKey = "scratch"

// NoFolderKey is the group for a transcript that never says which folder it ran in.
const NoFolderKey = "no-folder"

// Summary is one session as the list shows it.
type Summary struct {
	ID            string    `json:"id"`      // the transcript's id, which `claude --resume` takes
	LocalID       string    `json:"localId"` // the desktop app's id (local_…), "" for a CLI-only session
	Title         string    `json:"title"`
	TitleFrom     string    `json:"titleFrom"` // desktop | rename | generated | prompt | none
	Folder        string    `json:"folder"`    // the folder the session ran in
	GroupKey      string    `json:"groupKey"`
	Worktree      string    `json:"worktree"` // the worktree's name when it ran in <repo>/.claude/worktrees/<name>
	GitBranch     string    `json:"gitBranch"`
	Model         string    `json:"model"`
	Archived      bool      `json:"archived"`
	HasTranscript bool      `json:"hasTranscript"`
	Started       time.Time `json:"started"`
	LastActive    time.Time `json:"lastActive"`
	Prompts       int       `json:"prompts"`
	FirstPrompt   string    `json:"firstPrompt"`
}

// Group is the sessions of one folder.
type Group struct {
	Key        string    `json:"key"`
	Label      string    `json:"label"`
	Path       string    `json:"path"` // "" for the scratch and no-folder groups
	LastActive time.Time `json:"lastActive"`
	Sessions   []Summary `json:"sessions"`
}

const worktreeMark = "/.claude/worktrees/"

// cleanFolder drops trailing separators and writes a Windows drive letter in capitals, so "c:\x\" and "C:\x" are one
// folder. Paths are handled as text, not with filepath, because a transcript written on Windows can be read on a Mac
// (and the tests run on both).
func cleanFolder(folder string) string {
	folder = strings.TrimSpace(folder)
	for len(folder) > 1 && strings.ContainsAny(folder[len(folder)-1:], `/\`) && !strings.HasSuffix(folder, `:\`) {
		folder = folder[:len(folder)-1]
	}
	if len(folder) >= 2 && folder[1] == ':' {
		folder = strings.ToUpper(folder[:1]) + folder[1:]
	}
	return folder
}

// placeOf says which group a folder belongs to, and the worktree's name when the folder is inside one. Mac and
// Windows paths both work: the marks are looked for with every "\" read as "/", which keeps every position the same.
func placeOf(folder string) (key, worktree string) {
	folder = cleanFolder(folder)
	if folder == "" || folder == "." {
		return NoFolderKey, ""
	}
	slashed := strings.ReplaceAll(folder, `\`, "/")
	if strings.Contains(slashed, "/Claude/scratch-workspaces/") {
		return ScratchKey, ""
	}
	if i := strings.Index(slashed, worktreeMark); i >= 0 {
		name, _, _ := strings.Cut(slashed[i+len(worktreeMark):], "/")
		return folder[:i], name
	}
	return folder, ""
}

// baseName is a path's last folder name, after "/" or "\".
func baseName(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i >= 0 && i < len(path)-1 {
		return path[i+1:]
	}
	return path
}

// titleOf picks the session's name: the desktop's title, then a /rename, then the name Claude made, then the first prompt.
func titleOf(d *desktopInfo, f facts) (string, string) {
	switch {
	case d != nil && strings.TrimSpace(d.Title) != "":
		return OneLine(d.Title, 120), "desktop"
	case f.CustomTitle != "":
		return OneLine(f.CustomTitle, 120), "rename"
	case f.AiTitle != "":
		return OneLine(f.AiTitle, 120), "generated"
	case f.FirstPrompt != "":
		return OneLine(f.FirstPrompt, 80), "prompt"
	}
	return "Untitled session", "none"
}

// summarize joins what the transcript and the desktop app know about one session; either may be missing.
func summarize(id string, f *facts, d *desktopInfo) Summary {
	var ff facts
	if f != nil {
		ff = *f
	}
	s := Summary{ID: id, HasTranscript: f != nil, Folder: ff.Cwd, GitBranch: ff.GitBranch, Model: ff.Model,
		Started: ff.Started, LastActive: ff.LastActive, Prompts: ff.Prompts, FirstPrompt: ff.FirstPrompt}
	s.Title, s.TitleFrom = titleOf(d, ff)
	groupFrom := s.Folder
	if d != nil {
		s.LocalID = d.LocalID
		s.Archived = d.Archived
		if d.Cwd != "" {
			s.Folder = d.Cwd
		}
		groupFrom = s.Folder
		if d.OriginCwd != "" {
			groupFrom = d.OriginCwd
		}
		if s.Model == "" {
			s.Model = d.Model
		}
		if s.Started.IsZero() || (!d.Created.IsZero() && d.Created.Before(s.Started)) {
			s.Started = d.Created
		}
		if d.LastActive.After(s.LastActive) {
			s.LastActive = d.LastActive
		}
	}
	s.GroupKey, _ = placeOf(groupFrom)
	_, s.Worktree = placeOf(s.Folder)
	return s
}

// buildGroups puts sessions in their folder's group: newest group first, newest session first inside each.
func buildGroups(all []Summary) []Group {
	byKey := map[string]*Group{}
	for _, s := range all {
		g := byKey[s.GroupKey]
		if g == nil {
			g = &Group{Key: s.GroupKey, Sessions: []Summary{}}
			switch s.GroupKey {
			case ScratchKey:
				g.Label = "Scratch sessions (no folder)"
			case NoFolderKey:
				g.Label = "Folder not recorded"
			default:
				g.Path = s.GroupKey
				g.Label = baseName(s.GroupKey)
			}
			byKey[s.GroupKey] = g
		}
		g.Sessions = append(g.Sessions, s)
		if s.LastActive.After(g.LastActive) {
			g.LastActive = s.LastActive
		}
	}
	out := make([]Group, 0, len(byKey))
	for _, g := range byKey {
		sort.SliceStable(g.Sessions, func(i, j int) bool {
			a, b := g.Sessions[i], g.Sessions[j]
			if !a.LastActive.Equal(b.LastActive) {
				return a.LastActive.After(b.LastActive)
			}
			return a.ID < b.ID
		})
		out = append(out, *g)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].LastActive.Equal(out[j].LastActive) {
			return out[i].LastActive.After(out[j].LastActive)
		}
		return out[i].Key < out[j].Key
	})
	return out
}
