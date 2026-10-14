package main

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/AlBannaTechno/claude-sessions-list/internal/app"
	"github.com/AlBannaTechno/claude-sessions-list/internal/sessions"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	dirs, err := sessions.DefaultDirs()
	if err != nil {
		log.Fatal(err)
	}
	// "claude-sessions --list" prints the groups and stops: a check that needs no window.
	if len(os.Args) > 1 && os.Args[1] == "--list" {
		if err := printList(os.Stdout, sessions.NewIndex(dirs)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	h := &host{}
	svc := app.NewService(sessions.NewIndex(dirs), h)

	wapp := application.New(application.Options{
		Name:        "Claude Sessions",
		Description: "Lists every Claude Code session on this computer, grouped by folder",
		Services:    []application.Service{application.NewService(svc)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.albannatechno.claudesessions",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if h.window != nil {
					h.window.Show()
					h.window.Focus()
				}
			},
		},
	})
	h.app = wapp

	h.window = wapp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "Claude Sessions",
		Width:     1280,
		Height:    820,
		MinWidth:  900,
		MinHeight: 600,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 44,
			Backdrop:                application.MacBackdropNormal,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		// The page's own background for the system's theme, so a dark system never sees a white window first.
		BackgroundColour: windowBackground(systemDark()),
		URL:              "/",
	})
	wapp.Event.OnApplicationEvent(events.Common.ThemeChanged, func(*application.ApplicationEvent) {
		h.window.SetBackgroundColour(windowBackground(wapp.Env.IsDarkMode()))
	})
	// Coming back to the window reads the sessions again, so ones started meanwhile show up.
	h.window.OnWindowEvent(events.Common.WindowFocus, func(*application.WindowEvent) {
		wapp.Event.Emit("window:focus", nil)
	})

	if err := wapp.Run(); err != nil {
		log.Fatal(err)
	}
}

// windowBackground is the page's --bg in frontend/src/styles.css: #19191b dark, #ffffff light.
func windowBackground(dark bool) application.RGBA {
	if dark {
		return application.NewRGB(0x19, 0x19, 0x1b)
	}
	return application.NewRGB(0xff, 0xff, 0xff)
}

// host connects the service to Wails and the system.
type host struct {
	app    *application.App
	window *application.WebviewWindow
}

func (h *host) OpenURL(url string) error { return h.app.Browser.OpenURL(url) }

func (h *host) SetClipboard(text string) error {
	if !h.app.Clipboard.SetText(text) {
		return errors.New("could not put the text on the clipboard")
	}
	return nil
}

func (h *host) ShowFolder(path string) error {
	cmd, checkExit := openFolderCommand(runtime.GOOS, path)
	if !checkExit {
		// explorer.exe ends with exit code 1 even when it opened the folder, so only a failure to start it counts.
		if err := cmd.Start(); err != nil {
			return err
		}
		go func() { _ = cmd.Wait() }()
		return nil
	}
	return cmd.Run()
}

// openFolderCommand is the system's own way to show a folder: Finder on macOS, File Explorer on Windows, the file
// manager on Linux. checkExit is false where the exit code says nothing about success.
func openFolderCommand(goos, path string) (cmd *exec.Cmd, checkExit bool) {
	switch goos {
	case "darwin":
		return exec.Command("open", path), true
	case "windows":
		return exec.Command("explorer", path), false
	default:
		return exec.Command("xdg-open", path), true
	}
}

func printList(w io.Writer, x *sessions.Index) error {
	l, err := x.List()
	if err != nil {
		return err
	}
	for _, g := range l.Groups {
		where := g.Path
		if where == "" {
			where = g.Key
		}
		fmt.Fprintf(w, "%4d  %-40s %s\n", len(g.Sessions), g.Label, where)
	}
	fmt.Fprintf(w, "\n%d sessions in %d folders; %d have a transcript. Read %d files in %s.\n",
		l.Sessions, len(l.Groups), l.Transcripts, l.ReadFiles, l.Took)
	for _, p := range l.Problems {
		fmt.Fprintln(w, "problem:", p)
	}
	return nil
}
