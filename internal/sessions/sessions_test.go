package sessions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// line builds one transcript line from a map, so tests read as data.
func line(t *testing.T, m map[string]any) string {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(b) + "\n"
}

func userLine(t *testing.T, cwd, ts string, content any) string {
	return line(t, map[string]any{"type": "user", "cwd": cwd, "timestamp": ts, "sessionId": "x",
		"message": map[string]any{"role": "user", "content": content}})
}

func assistantLine(t *testing.T, cwd, ts string, content any) string {
	return line(t, map[string]any{"type": "assistant", "cwd": cwd, "timestamp": ts,
		"message": map[string]any{"role": "assistant", "model": "claude-opus-5-5", "content": content}})
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTitleOrder(t *testing.T) {
	d := &desktopInfo{Title: "Desktop name"}
	all := facts{CustomTitle: "Renamed", AiTitle: "Made by Claude", FirstPrompt: "what is next"}
	cases := []struct {
		d     *desktopInfo
		f     facts
		title string
		from  string
	}{
		{d, all, "Desktop name", "desktop"},
		{nil, all, "Renamed", "rename"},
		{nil, facts{AiTitle: "Made by Claude", FirstPrompt: "p"}, "Made by Claude", "generated"},
		{nil, facts{FirstPrompt: "what is next"}, "what is next", "prompt"},
		{&desktopInfo{Title: "  "}, facts{}, "Untitled session", "none"},
	}
	for _, c := range cases {
		title, from := titleOf(c.d, c.f)
		if title != c.title || from != c.from {
			t.Errorf("titleOf(%v, %+v) = %q/%q, want %q/%q", c.d, c.f, title, from, c.title, c.from)
		}
	}
}

func TestPlaceOf(t *testing.T) {
	cases := []struct{ folder, key, worktree string }{
		{"/Users/me/code/shop", "/Users/me/code/shop", ""},
		{"/Users/me/code/shop/.claude/worktrees/new-checkout", "/Users/me/code/shop", "new-checkout"},
		{"/Users/me/code/shop/.claude/worktrees/new-checkout/src/web", "/Users/me/code/shop", "new-checkout"},
		{"/Users/me/Library/Application Support/Claude/scratch-workspaces/aaa/bbb/scratch-2026-01-01-abc123", ScratchKey, ""},
		{"", NoFolderKey, ""},
		// Windows paths: the same rules with "\", and a drive letter written either way is one folder.
		{`C:\Users\me\code\shop`, `C:\Users\me\code\shop`, ""},
		{`c:\Users\me\code\shop\`, `C:\Users\me\code\shop`, ""},
		{`C:\Users\me\code\shop\.claude\worktrees\new-checkout\src\web`, `C:\Users\me\code\shop`, "new-checkout"},
		{`C:\Users\me\AppData\Roaming\Claude\scratch-workspaces\aaa\bbb\scratch-2026-01-01-abc123`, ScratchKey, ""},
		{`C:\`, `C:\`, ""},
		{"/", "/", ""},
	}
	for _, c := range cases {
		key, wt := placeOf(c.folder)
		if key != c.key || wt != c.worktree {
			t.Errorf("placeOf(%q) = %q/%q, want %q/%q", c.folder, key, wt, c.key, c.worktree)
		}
	}
}

func TestCleanPrompt(t *testing.T) {
	cases := map[string]string{
		"<system-reminder>\nnoise\n</system-reminder>\nfix the login":                                                           "fix the login",
		"<command-message>graphify</command-message>\n<command-name>/graphify</command-name>\n<command-args>src</command-args>": "/graphify src",
		"<local-command-stdout>output</local-command-stdout>":                                                                   "",
		"plain words": "plain words",
	}
	for in, want := range cases {
		if got := CleanPrompt(in); got != want {
			t.Errorf("CleanPrompt(%q) = %q, want %q", in, got, want)
		}
	}
}

// The folder comes from the records' cwd: the dashed directory name loses "." and "-", so it is never decoded.
func TestScanTakesFolderFromRecords(t *testing.T) {
	cwd := "/Users/me/code/notes-app"
	body := line(t, map[string]any{"type": "queue-operation", "timestamp": "2026-10-10T09:30:49.708Z"}) +
		userLine(t, cwd, "2026-10-10T09:30:49.761Z", "<system-reminder>x</system-reminder>what is this repo") +
		assistantLine(t, cwd, "2026-10-10T09:31:00.000Z", []any{map[string]any{"type": "tool_use", "id": "t1", "name": "Bash", "input": map[string]any{"command": "ls"}}}) +
		userLine(t, cwd, "2026-10-10T09:31:01.000Z", []any{map[string]any{"type": "tool_result", "tool_use_id": "t1", "content": "a\nb"}}) +
		line(t, map[string]any{"type": "user", "isMeta": true, "cwd": cwd, "timestamp": "2026-10-10T09:31:02.000Z", "message": map[string]any{"content": "Caveat: ignore"}}) +
		line(t, map[string]any{"type": "user", "isSidechain": true, "cwd": "/elsewhere", "timestamp": "2026-10-10T09:31:03.000Z", "message": map[string]any{"content": "agent prompt"}}) +
		userLine(t, cwd, "2026-10-10T09:40:00.000Z", "and now?") +
		line(t, map[string]any{"type": "ai-title", "aiTitle": "First name"}) +
		line(t, map[string]any{"type": "custom-title", "customTitle": "Old rename"}) +
		line(t, map[string]any{"type": "custom-title", "customTitle": "New rename"})
	f, err := scanFacts(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if f.Cwd != cwd {
		t.Errorf("cwd %q", f.Cwd)
	}
	if f.Prompts != 2 {
		t.Errorf("prompts %d, want 2 (tool answers, system lines and agent lines are not prompts)", f.Prompts)
	}
	if f.FirstPrompt != "what is this repo" {
		t.Errorf("first prompt %q", f.FirstPrompt)
	}
	if f.CustomTitle != "New rename" || f.AiTitle != "First name" {
		t.Errorf("titles %q / %q", f.CustomTitle, f.AiTitle)
	}
	if f.Model != "claude-opus-5-5" {
		t.Errorf("model %q", f.Model)
	}
	if want, _ := time.Parse(time.RFC3339, "2026-10-10T09:40:00Z"); !f.LastActive.Equal(want) {
		t.Errorf("last active %v", f.LastActive)
	}
}

// One line of several megabytes (a pasted image) is read, not dropped with the rest of the file after it.
func TestScanReadsAHugeLine(t *testing.T) {
	big := strings.Repeat("A", 3<<20)
	body := userLine(t, "/p", "2026-10-10T09:30:00Z", []any{map[string]any{"type": "image", "source": map[string]any{"data": big}}}) +
		userLine(t, "/p", "2026-10-10T09:31:00Z", "after the image")
	f, err := scanFacts(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if f.FirstPrompt != "after the image" {
		t.Errorf("first prompt %q", f.FirstPrompt)
	}
}

func TestTranscriptEntries(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "s.jsonl")
	write(t, p,
		userLine(t, "/p", "2026-10-10T09:30:00Z", "<system-reminder>hidden</system-reminder>hello")+
			assistantLine(t, "/p", "2026-10-10T09:30:01Z", []any{
				map[string]any{"type": "thinking", "thinking": "let me think"},
				map[string]any{"type": "text", "text": "| a | b |\n|---|---|\n| 1 | 2 |"},
				map[string]any{"type": "tool_use", "id": "t1", "name": "Edit", "input": map[string]any{"file_path": "/p/App.tsx", "old_string": "x"}},
			})+
			userLine(t, "/p", "2026-10-10T09:30:02Z", []any{map[string]any{"type": "tool_result", "tool_use_id": "t1", "is_error": true, "content": []any{map[string]any{"type": "text", "text": "no match"}}}})+
			line(t, map[string]any{"type": "user", "isCompactSummary": true, "timestamp": "2026-10-10T09:31:00Z", "message": map[string]any{"content": "Summary of before"}}))
	tr, err := LoadTranscript(p, "s")
	if err != nil {
		t.Fatal(err)
	}
	kinds := []string{}
	for _, e := range tr.Entries {
		kinds = append(kinds, e.Kind)
	}
	if got := strings.Join(kinds, ","); got != "user,thinking,assistant,tool,summary" {
		t.Fatalf("kinds %s", got)
	}
	if tr.Entries[0].Text != "hello" {
		t.Errorf("user text %q", tr.Entries[0].Text)
	}
	tool := tr.Entries[3]
	if tool.Tool != "Edit" || tool.Brief != "/p/App.tsx" || tool.Result != "no match" || !tool.IsError {
		t.Errorf("tool %+v", tool)
	}
}

func TestClipKeepsWholeCharacters(t *testing.T) {
	s := strings.Repeat("é", maxShown) // 2 bytes each, so the cut falls inside one
	got := clip(s)
	if !strings.Contains(got, "more characters not shown") || strings.ContainsRune(got, '�') {
		t.Errorf("clip cut badly: ends %q", got[len(got)-60:])
	}
}

// The listing joins transcripts with desktop records, groups by folder, never returns nil lists, and reads again only
// the files that changed.
func TestIndexListAndCache(t *testing.T) {
	root := t.TempDir()
	dirs := Dirs{Projects: filepath.Join(root, "projects"), Desktop: filepath.Join(root, "desktop"), Cache: filepath.Join(root, "cache", "index.json")}
	repo := "/Users/me/code/shop"
	write(t, filepath.Join(dirs.Projects, "-Users-me-code-shop", "aaa.jsonl"), userLine(t, repo, "2026-10-01T10:00:00Z", "first"))
	write(t, filepath.Join(dirs.Projects, "-Users-me-code-shop--claude-worktrees-wt1", "bbb.jsonl"), userLine(t, repo+"/.claude/worktrees/wt1", "2026-10-02T10:00:00Z", "in a worktree"))
	write(t, filepath.Join(dirs.Projects, "-Users-me-code-other", "ccc.jsonl"), userLine(t, "/Users/me/code/other", "2026-09-01T10:00:00Z", "older folder"))
	write(t, filepath.Join(dirs.Projects, "-Users-me-code-other", "ccc", "subagents", "agent-1.jsonl"), userLine(t, "/x", "2026-10-09T10:00:00Z", "an agent's own file"))
	write(t, filepath.Join(dirs.Desktop, "acct", "org", "local_1.json"), line(t, map[string]any{
		"sessionId": "local_1", "cliSessionId": "aaa", "title": "Desktop title", "cwd": repo, "originCwd": repo,
		"isArchived": false, "createdAt": 1, "lastActivityAt": time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC).UnixMilli(),
		"remoteMcpServersConfig": []any{map[string]any{"name": "big and ignored"}}}))
	write(t, filepath.Join(dirs.Desktop, "acct", "org", "local_2.json"), line(t, map[string]any{
		"sessionId": "local_2", "cliSessionId": "zzz", "title": "No transcript here", "cwd": "/Users/me/code/gone", "lastActivityAt": 5}))

	x := NewIndex(dirs)
	l, err := x.List()
	if err != nil {
		t.Fatal(err)
	}
	if l.Transcripts != 3 || l.Sessions != 4 {
		t.Fatalf("transcripts %d sessions %d, want 3 and 4", l.Transcripts, l.Sessions)
	}
	if l.ReadFiles != 5 {
		t.Errorf("read %d files, want 5", l.ReadFiles)
	}
	g := l.Groups[0]
	if g.Path != repo || g.Label != "shop" || len(g.Sessions) != 2 {
		t.Fatalf("first group %+v", g)
	}
	if g.Sessions[0].ID != "aaa" || g.Sessions[0].Title != "Desktop title" || g.Sessions[0].LocalID != "local_1" {
		t.Errorf("newest session %+v", g.Sessions[0])
	}
	if g.Sessions[1].Worktree != "wt1" {
		t.Errorf("worktree %+v", g.Sessions[1])
	}
	if p, ok := x.TranscriptPath("ccc"); !ok || !strings.HasSuffix(p, "ccc.jsonl") {
		t.Errorf("path of ccc %q", p)
	}
	b, _ := json.Marshal(l)
	if strings.Contains(string(b), "null") {
		t.Errorf("a list went out as null: %s", b)
	}

	// A fresh index reads the cache file and reads again only the one file that changed.
	write(t, filepath.Join(dirs.Projects, "-Users-me-code-other", "ccc.jsonl"), userLine(t, "/Users/me/code/other", "2026-09-01T10:00:00Z", "older folder")+userLine(t, "/Users/me/code/other", "2026-10-20T10:00:00Z", "again"))
	l2, err := NewIndex(dirs).List()
	if err != nil {
		t.Fatal(err)
	}
	if l2.ReadFiles != 1 {
		t.Errorf("read %d files after one change, want 1", l2.ReadFiles)
	}
	if l2.Groups[0].Label != "other" {
		t.Errorf("the changed folder is newest now; first group is %q", l2.Groups[0].Label)
	}
}

func TestBaseName(t *testing.T) {
	cases := map[string]string{
		"/Users/me/code/shop":   "shop",
		`C:\Users\me\code\shop`: "shop",
		`C:\`:                   `C:\`,
		"/":                     "/",
		"shop":                  "shop",
	}
	for in, want := range cases {
		if got := baseName(in); got != want {
			t.Errorf("baseName(%q) = %q, want %q", in, got, want)
		}
	}
}
