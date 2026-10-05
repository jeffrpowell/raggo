package tika

import (
	"regexp"
	"strings"
)

var (
	dotLeaders         = regexp.MustCompile(`\.{4,}`)
	lineBreakHyphen    = regexp.MustCompile(`\p{L}+-(?:[ \t]+|[ \t]*\n[ \t]*)\p{Ll}\p{L}*`)
	hyphenCompound     = regexp.MustCompile(`\p{L}+-\p{L}+`)
	horizontalSpace    = regexp.MustCompile(`[ \t\x{00A0}]+`)
	spaceAroundNewline = regexp.MustCompile(` *\n *`)
	extraBlankLines    = regexp.MustCompile(`\n{3,}`)
)

// CleanText removes PDF layout noise from extracted text: unmapped glyphs
// (U+FFFD), lines repeated on most pages (watermarks, running headers), table
// of contents dot leaders, words hyphenated across line breaks, and runs of
// whitespace. Repeated-line removal needs pageCount; pass 0 to skip it.
func CleanText(text string, pageCount int) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "�", "")
	text = dropRepeatedLines(text, pageCount)
	text = dotLeaders.ReplaceAllString(text, " ")
	text = joinLineBreakHyphens(text)
	text = horizontalSpace.ReplaceAllString(text, " ")
	text = spaceAroundNewline.ReplaceAllString(text, "\n")
	text = extraBlankLines.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// dropRepeatedLines removes lines that occur at least once per two pages.
func dropRepeatedLines(text string, pageCount int) string {
	if pageCount < 4 {
		return text
	}
	lines := strings.Split(text, "\n")
	counts := make(map[string]int)
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			counts[trimmed]++
		}
	}
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if counts[strings.TrimSpace(line)]*2 < pageCount {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

// joinLineBreakHyphens rejoins words split at a line break ("conver- sations"
// -> "conversations"). The hyphen is kept when the document spells the word
// with one elsewhere ("self- defense" -> "self-defense").
func joinLineBreakHyphens(text string) string {
	compounds := make(map[string]bool)
	for _, c := range hyphenCompound.FindAllString(text, -1) {
		compounds[strings.ToLower(c)] = true
	}
	return lineBreakHyphen.ReplaceAllStringFunc(text, func(m string) string {
		i := strings.IndexByte(m, '-')
		head, tail := m[:i], strings.TrimLeft(m[i+1:], " \t\n")
		if compounds[strings.ToLower(head+"-"+tail)] {
			return head + "-" + tail
		}
		return head + tail
	})
}
