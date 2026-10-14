package sessions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"time"
)

// record is one line of a transcript, with only the fields this app reads.
type record struct {
	Type             string          `json:"type"`
	Cwd              string          `json:"cwd"`
	Timestamp        string          `json:"timestamp"`
	IsMeta           bool            `json:"isMeta"`
	IsSidechain      bool            `json:"isSidechain"`
	IsCompactSummary bool            `json:"isCompactSummary"`
	CustomTitle      string          `json:"customTitle"`
	AiTitle          string          `json:"aiTitle"`
	GitBranch        string          `json:"gitBranch"`
	Message          json.RawMessage `json:"message"`
}

// message is the "message" field of a user or assistant line.
type message struct {
	Model   string          `json:"model"`
	Content json.RawMessage `json:"content"`
}

// block is one item of a message's content list.
type block struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	Name      string          `json:"name"`
	ID        string          `json:"id"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

// contentBlocks reads a content field that is either a plain string or a list of blocks.
func contentBlocks(raw json.RawMessage) []block {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}
	if raw[0] == '"' {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return []block{{Type: "text", Text: s}}
		}
		return nil
	}
	var bs []block
	_ = json.Unmarshal(raw, &bs)
	return bs
}

// facts is what one transcript file says about its session, without the conversation itself.
type facts struct {
	Cwd         string    `json:"cwd"`
	CustomTitle string    `json:"customTitle"`
	AiTitle     string    `json:"aiTitle"`
	FirstPrompt string    `json:"firstPrompt"`
	Model       string    `json:"model"`
	GitBranch   string    `json:"gitBranch"`
	Started     time.Time `json:"started"`
	LastActive  time.Time `json:"lastActive"`
	Prompts     int       `json:"prompts"`
}

// maxLine is the longest transcript line read; a pasted image or a big tool result can pass several megabytes.
const maxLine = 256 << 20

func newLineScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 1<<20), maxLine)
	return sc
}

func scanFile(path string) (facts, error) {
	f, err := os.Open(path)
	if err != nil {
		return facts{}, err
	}
	defer f.Close()
	return scanFacts(f)
}

func scanFacts(r io.Reader) (facts, error) {
	var out facts
	sc := newLineScanner(r)
	for sc.Scan() {
		line := sc.Bytes()
		var rec record
		if json.Unmarshal(line, &rec) != nil {
			continue
		}
		if t, err := time.Parse(time.RFC3339Nano, rec.Timestamp); err == nil {
			if out.Started.IsZero() || t.Before(out.Started) {
				out.Started = t
			}
			if t.After(out.LastActive) {
				out.LastActive = t
			}
		}
		switch rec.Type {
		case "custom-title":
			if rec.CustomTitle != "" {
				out.CustomTitle = rec.CustomTitle // the latest rename wins
			}
			continue
		case "ai-title":
			if rec.AiTitle != "" {
				out.AiTitle = rec.AiTitle
			}
			continue
		case "user", "assistant":
		default:
			continue
		}
		if rec.IsSidechain {
			continue
		}
		if out.Cwd == "" && rec.Cwd != "" {
			out.Cwd = rec.Cwd
		}
		if out.GitBranch == "" && rec.GitBranch != "" && rec.GitBranch != "HEAD" {
			out.GitBranch = rec.GitBranch
		}
		if rec.Type == "assistant" {
			if out.Model == "" {
				var m message
				if json.Unmarshal(rec.Message, &m) == nil && m.Model != "" && m.Model != "<synthetic>" {
					out.Model = m.Model
				}
			}
			continue
		}
		// A user line: a prompt the person typed, unless it is a tool's answer or added by the system.
		if rec.IsMeta || rec.IsCompactSummary || bytes.Contains(rec.Message, []byte(`"tool_result"`)) {
			continue
		}
		var m message
		if json.Unmarshal(rec.Message, &m) != nil {
			continue
		}
		text := ""
		for _, b := range contentBlocks(m.Content) {
			if b.Type == "text" {
				text += b.Text + "\n"
			}
		}
		text = CleanPrompt(text)
		if text == "" {
			continue
		}
		out.Prompts++
		if out.FirstPrompt == "" {
			out.FirstPrompt = OneLine(text, 200)
		}
	}
	return out, sc.Err()
}
