package tika

import (
	"fmt"
	"strings"
	"testing"
)

func TestCleanTextJoinsLineBreakHyphens(t *testing.T) {
	in := "Group conver- sations matter.\nShe trained in self- defense and self-defense.\nA labyrin-\nthine maze by Chung- Ling."
	want := "Group conversations matter.\nShe trained in self-defense and self-defense.\nA labyrinthine maze by Chung- Ling."
	if got := CleanText(in, 0); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestCleanTextDropsLinesRepeatedAcrossPages(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 6; i++ {
		fmt.Fprintf(&b, "(Order #123)\nText of page %d.\n", i)
		if i < 2 {
			b.WriteString("Chapter heading\n")
		}
	}

	got := CleanText(b.String(), 6)
	if strings.Contains(got, "(Order #123)") {
		t.Error("watermark on every page was kept")
	}
	if !strings.Contains(got, "Chapter heading") || !strings.Contains(got, "Text of page 5.") {
		t.Error("lines that are not on most pages were dropped")
	}

	if got := CleanText(b.String(), 0); !strings.Contains(got, "(Order #123)") {
		t.Error("repeated lines dropped without a page count")
	}
}

func TestCleanTextRemovesLayoutNoise(t *testing.T) {
	in := "Chapter 3: Play........ 93\n\n\n\n��  this was a victory\t\tWhen you  win "
	want := "Chapter 3: Play 93\n\nthis was a victory When you win"
	if got := CleanText(in, 0); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}
