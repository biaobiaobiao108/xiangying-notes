package markdown

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"xiangying/internal/model"
)

var (
	tagRegex         = regexp.MustCompile(`(?:\s|^)\\?#([a-zA-Z0-9_\p{Han}]+)`)
	headingRegex     = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	orderedListRegex = regexp.MustCompile(`^\d{1,3}[.)]\s+`)
)

// LooksLikeMarkdown reports whether text carries Markdown syntax, so a
// plain-text paste can be rendered instead of inserted literally.
func LooksLikeMarkdown(text string) bool {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "```"), strings.HasPrefix(line, "~~~"):
			return true
		case headingRegex.MatchString(line):
			return true
		case strings.HasPrefix(line, "> "):
			return true
		case strings.HasPrefix(line, "- "), strings.HasPrefix(line, "* "), strings.HasPrefix(line, "+ "):
			return true
		case orderedListRegex.MatchString(line):
			return true
		case strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|"):
			return true
		case line == "---", line == "***", line == "___":
			return true
		}
	}
	for _, marker := range []string{"**", "__", "~~", "](", "<u>", "<s>", "<b>", "<i>"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

// ExtractTags finds all #tags outside fenced code blocks.
func ExtractTags(markdown string) []string {
	var tags []string
	seen := make(map[string]bool)

	lines := strings.Split(markdown, "\n")
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		matches := tagRegex.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) > 1 {
				tag := strings.TrimSpace(m[1])
				if tag != "" && !seen[tag] {
					seen[tag] = true
					tags = append(tags, tag)
				}
			}
		}
	}

	return tags
}

// ExtractHeadings finds headings for document outline.
func ExtractHeadings(markdown string) []model.HeadingItem {
	var headings []model.HeadingItem
	lines := strings.Split(markdown, "\n")
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		if m := headingRegex.FindStringSubmatch(line); len(m) == 3 {
			headings = append(headings, model.HeadingItem{
				Level: len(m[1]),
				Text:  strings.TrimSpace(m[2]),
			})
		}
	}

	return headings
}

// ExcerptPreview generates a clean plain-text preview of the note content.
func ExcerptPreview(markdown string, maxLines int, maxLen int) string {
	lines := strings.Split(markdown, "\n")
	var cleanParts []string
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		// Strip markdown syntax
		cleaned := trimmed
		cleaned = headingRegex.ReplaceAllString(cleaned, "$2")
		cleaned = strings.TrimPrefix(cleaned, "- [ ] ")
		cleaned = strings.TrimPrefix(cleaned, "- [x] ")
		cleaned = strings.TrimPrefix(cleaned, "- [X] ")
		cleaned = strings.TrimPrefix(cleaned, "- ")
		cleaned = strings.TrimPrefix(cleaned, "* ")
		cleaned = strings.TrimPrefix(cleaned, "> ")
		cleaned = strings.ReplaceAll(cleaned, "**", "")
		cleaned = strings.ReplaceAll(cleaned, "__", "")
		cleaned = strings.ReplaceAll(cleaned, "`", "")
		cleaned = strings.TrimSpace(cleaned)

		if cleaned != "" {
			cleanParts = append(cleanParts, cleaned)
			if len(cleanParts) >= maxLines {
				break
			}
		}
	}

	result := strings.Join(cleanParts, " ")
	runes := []rune(result)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return result
}

// WordStats calculates word count and character count.
func WordStats(text string) (words int, chars int) {
	chars = len([]rune(text))
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r)
	})
	words = len(fields)
	return words, chars
}

// RelativeDate formats a timestamp in a clean relative format matching original Xiangying.
func RelativeDate(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute {
		return "刚刚"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%d 分钟前", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%d 小时前", int(diff.Hours()))
	}
	if diff < 7*24*time.Hour {
		return fmt.Sprintf("%d 天前", int(diff.Hours()/24))
	}
	return t.Format("01-02")
}
