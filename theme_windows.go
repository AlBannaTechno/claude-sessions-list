//go:build windows

package main

import "github.com/wailsapp/wails/v3/pkg/w32"

// systemDark says whether Windows apps use the dark theme now. Wails' own Env.IsDarkMode answers false until the app
// runs, and the window is made before that (U2).
func systemDark() bool { return w32.IsCurrentlyDarkMode() }
