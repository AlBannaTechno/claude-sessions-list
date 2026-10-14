package sessions

import (
	"bytes"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
)

// Entry is one thing shown in a conversation.
type Entry struct {
	Kind    string    `json:"kind"` // user | assistant | thinking | tool | summary
	Text    string    `json:"text"`
	Tool    string    `json:"tool"`    // the tool's name, for kind "tool"
	Brief   string    `json:"brief"`   // one line saying what the tool was asked: a command, a file
	Input   string    `json:"input"`   // the tool's full input, as indented JSON
	Result  string    `json:"result"`  // what the tool answered
	IsError bool      `json:"isError"` // the tool answered with an error
	Time    time.Time `json:"time"`
}

// Transcript is a session's conversation.
type Transcript struct {
	ID      string  `json:"id"`
	Entries []Entry `json:"entries"`
}

// maxShown caps one tool input or result; the rest is cut with a note, since a page of megabytes helps no one.
const maxShown = 20_000

func clip(s string) string {
	if len(s) <= maxShown {
		return s
	}
	cut := strings.ToValidUTF8(s[:maxShown], "") // a character split in half is dropped
	return cut + "\n… (" + strconv.Itoa(len(s)-len(cut)) + " more characters not shown)"
}

// LoadTranscript reads a transcript file into the entries the window draws.
func LoadTranscript(path, id string) (Transcript, error) {
	f, err := os.Open(path)
	if err != nil {
		return Transcript{ID: id, Entries: []Entry{}}, err
	}
	defer f.Close()
	t := Transcript{ID: id, Entries: []Entry{}}
	toolAt := map[string]int{} // tool_use id → its entry, to attach the answer when it comes
	sc := newLineScanner(f)
	for sc.Scan() {
		var rec record
		if json.Unmarshal(sc.Bytes(), &rec) != nil {
			continue
		}
		if (rec.Type != "user" && rec.Type != "assistant") || rec.IsSidechain {
			continue
		}
		when, _ := time.Parse(time.RFC3339Nano, rec.Timestamp)
		var m message
		if json.Unmarshal(rec.Message, &m) != nil {
			continue
		}
		if rec.Type == "user" && rec.IsCompactSummary {
			text := ""
			for _, b := range contentBlocks(m.Content) {
				text += b.Text
			}
			t.Entries = append(t.Entries, Entry{Kind: "summary", Text: strings.TrimSpace(text), Time: when})
			continue
		}
		for _, b := range contentBlocks(m.Content) {
			switch b.Type {
			case "text":
				if rec.Type == "user" {
					if rec.IsMeta {
						continue
					}
					if s := CleanPrompt(b.Text); s != "" {
						t.Entries = append(t.Entries, Entry{Kind: "user", Text: s, Time: when})
					}
				} else if s := strings.TrimSpace(b.Text); s != "" {
					t.Entries = append(t.Entries, Entry{Kind: "assistant", Text: s, Time: when})
				}
			case "image":
				if rec.Type == "user" && !rec.IsMeta {
					t.Entries = append(t.Entries, Entry{Kind: "user", Text: "(pasted image)", Time: when})
				}
			case "thinking":
				if s := strings.TrimSpace(b.Thinking); s != "" {
					t.Entries = append(t.Entries, Entry{Kind: "thinking", Text: s, Time: when})
				}
			case "tool_use":
				toolAt[b.ID] = len(t.Entries)
				t.Entries = append(t.Entries, Entry{Kind: "tool", Tool: b.Name, Brief: briefOf(b.Name, b.Input), Input: clip(pretty(b.Input)), Time: when})
			case "tool_result":
				i, ok := toolAt[b.ToolUseID]
				if !ok {
					continue
				}
				t.Entries[i].Result = clip(resultText(b.Content))
				t.Entries[i].IsError = b.IsError
			}
		}
	}
	return t, sc.Err()
}

func pretty(raw json.RawMessage) string {
	var buf bytes.Buffer
	if json.Indent(&buf, raw, "", "  ") != nil {
		return string(raw)
	}
	return buf.String()
}

// resultText is a tool's answer as text; it is either a string or a list of blocks.
func resultText(raw json.RawMessage) string {
	var parts []string
	for _, b := range contentBlocks(raw) {
		switch b.Type {
		case "text":
			parts = append(parts, b.Text)
		case "image":
			parts = append(parts, "(image)")
		}
	}
	return hiddenBlocks.ReplaceAllString(strings.Join(parts, "\n"), "")
}

// briefOf is the one line that says what a tool call did: the command run, the file read, the text searched.
func briefOf(tool string, raw json.RawMessage) string {
	var in map[string]any
	if json.Unmarshal(raw, &in) != nil {
		return ""
	}
	str := func(k string) string { s, _ := in[k].(string); return s }
	var s string
	for _, k := range []string{"command", "file_path", "notebook_path", "pattern", "url", "query", "description", "prompt", "skill", "path"} {
		if s = str(k); s != "" {
			break
		}
	}
	if s == "" {
		for _, v := range in { // any one text value is better than nothing
			if vs, ok := v.(string); ok && vs != "" {
				s = vs
				break
			}
		}
	}
	return OneLine(s, 160)
}
