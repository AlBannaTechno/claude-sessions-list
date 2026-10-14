# Claude Sessions

A small desktop app that lists every [Claude Code](https://www.anthropic.com/claude-code) session on your computer,
grouped by the folder it ran in, and shows each conversation.

It was made after a Claude desktop app update removed **Group by → Project** from the Code tab's sidebar
([anthropics/claude-code#102157](https://github.com/anthropics/claude-code/issues/102157)), so sessions from every
folder ended up in one long list.

## What it does

- **One group per folder**, newest activity first. Sessions inside each group are newest first.
  - Sessions that ran in a worktree (`<repo>/.claude/worktrees/<name>`) are listed under their repo, with the worktree's name on the row.
  - Sessions the desktop app started without a folder are in one group, "Scratch sessions (no folder)".
- **Each session shows** its name, its id, when it was last active, and labels for worktree, archived, and CLI-only sessions.
- **The conversation** shows your messages, Claude's answers as formatted text (tables, code, lists), and each tool call
  as one line that opens to show what it was asked and what it answered. Claude's thinking is hidden until you switch it on.
- **All, CLI or Desktop:** list every session, only the ones started from the terminal, or only the ones started in the
  Claude desktop app. **Desktop** is the default; the app remembers what you pick.
- **Search** by session name, id, folder or first prompt.
- **Buttons** to open the session in the Claude desktop app, copy the command that continues it in the terminal
  (`claude --resume <id>`), or open its folder.
- **Keyboard:** ↑ and ↓ move between sessions, ← and → close and open a folder.

The session's name is taken, in this order, from: the title in the Claude desktop app, a name you gave with `/rename`,
the name Claude Code generated, and finally your first message.

## Where it reads from

The app only **reads** these files. It never changes or deletes anything that belongs to Claude.

| What | macOS | Windows |
|---|---|---|
| Conversations (CLI and desktop) | `~/.claude/projects/*/*.jsonl` | `%USERPROFILE%\.claude\projects\*\*.jsonl` |
| Desktop app titles and ids | `~/Library/Application Support/Claude/claude-code-sessions/` | `%APPDATA%\Claude\claude-code-sessions\` |

If `CLAUDE_CONFIG_DIR` is set, conversations are read from `$CLAUDE_CONFIG_DIR/projects` instead.

The only file the app writes is its own cache, so the next start does not read every conversation again:
`~/Library/Application Support/ClaudeSessions/index.json` on macOS, `%APPDATA%\ClaudeSessions\index.json` on Windows.
Deleting it is safe.

## Platforms

- **macOS:** built and used.
- **Windows (x86):** built and used on Windows 11.
- **Windows on ARM:** should build; not tried.
- **Linux:** should build; not tried.

## Build

You need:

- Go 1.25 or newer
- Node.js 20.19 or newer (or 22.12 or newer)
- The Wails 3 command line tool, version `v3.0.0-beta.20`:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.20
```

- On macOS only: Apple's Command Line Tools, because parts of the app are compiled with the Mac's C compiler. Most
  developers already have them; if the build fails with an error about `clang` or `xcrun`, install them:

```bash
xcode-select --install
```

An app you build yourself opens without any macOS warning, because it was not downloaded.

Install the frontend packages once:

```bash
cd frontend && npm install
```

Run it while developing (reloads on change):

```bash
wails3 dev
```

Build the macOS app into `bin/claude-sessions.app`:

```bash
wails3 task darwin:package
```

Build for Windows into `bin/claude-sessions.exe`, from any system. `ARCH=amd64` is for normal Intel and AMD PCs, `ARCH=arm64`
for Windows on ARM; without it the build matches the computer you build on.

```bash
wails3 task windows:build ARCH=amd64
```

Use this task rather than a plain `go build`: it also puts the app's icon into the `.exe`. Windows needs no C compiler.

After changing an exported method of `internal/app/Service`, regenerate the frontend's bindings:

```bash
wails3 generate bindings -clean=true -ts -i
```

## Check it

Go tests:

```bash
go test ./...
```

Frontend tests and type check:

```bash
cd frontend && npm test && npx tsc --noEmit
```

Print every folder and its session count without opening a window (the frontend must have been built once, since it is
embedded in the binary):

```bash
go run . --list
```

## Code layout

| Path | What is in it |
|---|---|
| `main.go` | The window, and the `--list` command |
| `internal/sessions/` | Finding sessions, reading transcripts, grouping by folder, the cache |
| `internal/app/` | The methods the window calls (list, load a conversation, copy, open) |
| `frontend/src/` | The React window: `FolderTree` (left), `SessionView` and `Transcript` (right) |
| `build/` | Icons and packaging settings for macOS, Windows and Linux |

Built with [Wails 3](https://v3.wails.io), React 19, Vite and Tailwind CSS 4.

## Not affiliated with Anthropic

This is an independent project. It is not made, endorsed or supported by Anthropic. "Claude" and "Claude Code" are
Anthropic's names for their products. The format of the files this app reads is not documented and can change with any
Claude Code update, which could stop this app from working until it is updated.

## License

[MIT](LICENSE)
