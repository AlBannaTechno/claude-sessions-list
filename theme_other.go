//go:build !darwin && !windows

package main

// systemDark is false here: the window starts light, and the theme change event sets it after (U2).
func systemDark() bool { return false }
