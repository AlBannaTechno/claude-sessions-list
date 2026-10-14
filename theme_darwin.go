//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>

// The same setting Wails reads, readable before the app runs: "Dark" while the system is dark, Auto included.
static bool systemIsDark(void) {
	NSString *style = [[NSUserDefaults standardUserDefaults] stringForKey:@"AppleInterfaceStyle"];
	return style != nil && [style isEqualToString:@"Dark"];
}
*/
import "C"

// systemDark says whether macOS is in dark mode now. Wails' own Env.IsDarkMode answers false until the app runs, and
// the window is made before that (U2).
func systemDark() bool { return bool(C.systemIsDark()) }
