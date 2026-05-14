package services

import (
	"regexp"
	"strings"
)

var (
	vttTimestamp = regexp.MustCompile(`\d{2}:\d{2}:\d{2}\.\d{3}\s*-->\s*\d{2}:\d{2}:\d{2}\.\d{3}[^\n]*`)
	srtTimestamp = regexp.MustCompile(`\d{2}:\d{2}:\d{2},\d{3}\s*-->\s*\d{2}:\d{2}:\d{2},\d{3}`)
	srtSequence  = regexp.MustCompile(`(?m)^\d+\s*$`)
	vttHeader    = regexp.MustCompile(`(?i)^WEBVTT.*`)
	htmlTags     = regexp.MustCompile(`<[^>]+>`)
	multiSpace   = regexp.MustCompile(`[ \t]+`)
	multiNewline = regexp.MustCompile(`\n{3,}`)
)

func CleanTranscript(text string) string {
	text = vttHeader.ReplaceAllString(text, "")
	text = vttTimestamp.ReplaceAllString(text, "")
	text = srtTimestamp.ReplaceAllString(text, "")
	text = srtSequence.ReplaceAllString(text, "")
	text = htmlTags.ReplaceAllString(text, "")

	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		line = multiSpace.ReplaceAllString(line, " ")
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	result := strings.Join(cleaned, "\n")
	result = multiNewline.ReplaceAllString(result, "\n\n")
	return strings.TrimSpace(result)
}
