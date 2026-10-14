package main

import (
	"fmt"
	"os"
	"regexp"
	"testing"
)

// The window's background is the page's --bg, light and dark, so nothing of another colour shows before the page
// draws (U2).
func TestWindowBackgroundIsThePagesBackground(t *testing.T) {
	css, err := os.ReadFile("frontend/src/styles.css")
	if err != nil {
		t.Fatal(err)
	}
	bgs := regexp.MustCompile(`--bg: (#[0-9a-f]{6});`).FindAllStringSubmatch(string(css), -1)
	if len(bgs) != 2 {
		t.Fatalf("want --bg twice in styles.css, light then dark; found %d", len(bgs))
	}
	for i, dark := range []bool{false, true} {
		c := windowBackground(dark)
		if got := fmt.Sprintf("#%02x%02x%02x", c.Red, c.Green, c.Blue); got != bgs[i][1] {
			t.Errorf("dark %v: window %s, page %s", dark, got, bgs[i][1])
		}
	}
}
