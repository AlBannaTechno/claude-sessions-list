package sessions

import (
	"regexp"
	"strings"
)

// Blocks Claude Code adds to a prompt that the person never typed.
var hiddenBlocks = regexp.MustCompile(`(?s)<(system-reminder|local-command-stdout|local-command-stderr|local-command-caveat|command-message)>.*?</(system-reminder|local-command-stdout|local-command-stderr|local-command-caveat|command-message)>`)

var (
	commandName = regexp.MustCompile(`(?s)<command-name>(.*?)</command-name>`)
	commandArgs = regexp.MustCompile(`(?s)<command-args>(.*?)</command-args>`)
	anyTag      = regexp.MustCompile(`</?[a-z][a-z0-9-]*>`)
)

// CleanPrompt is what the person typed: the system's added blocks removed, and a slash command shown as "/name args".
func CleanPrompt(s string) string {
	if m := commandName.FindStringSubmatch(s); m != nil {
		out := strings.TrimSpace(m[1])
		if a := commandArgs.FindStringSubmatch(s); a != nil && strings.TrimSpace(a[1]) != "" {
			out += " " + strings.TrimSpace(a[1])
		}
		return out
	}
	s = hiddenBlocks.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// OneLine squeezes text into one line of at most n characters, for names and row summaries.
func OneLine(s string, n int) string {
	s = anyTag.ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > n {
		return strings.TrimSpace(string(r[:n-1])) + "…"
	}
	return s
}
