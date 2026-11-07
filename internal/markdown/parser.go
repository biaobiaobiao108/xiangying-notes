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
	wikiLinkRegex    = regexp.MustCompile(`\[\[([^\]\n|]+)(?:\|([^\]\n]+))?\]\]`)
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

type codeFence struct {
	marker byte
	length int
}

// skipCodeFence reports whether line belongs to a fenced code block, including
// its opening and closing fence lines. A fence can only be closed by a run of
// the same marker that is at least as long as its opener.
func skipCodeFence(line string, fence *codeFence) bool {
	trimmed := strings.TrimLeft(line, " ")
	indent := len(line) - len(trimmed)
	if indent > 3 || trimmed == "" {
		return fence.length > 0
	}

	marker := trimmed[0]
	if marker != '`' && marker != '~' {
		return fence.length > 0
	}
	run := 0
	for run < len(trimmed) && trimmed[run] == marker {
		run++
	}

	if fence.length > 0 {
		if marker == fence.marker && run >= fence.length && strings.TrimSpace(trimmed[run:]) == "" {
			*fence = codeFence{}
		}
		return true
	}
	if run < 3 {
		return false
	}
	// Backticks cannot appear in the info string of a backtick fence.
	if marker == '`' && strings.Contains(trimmed[run:], "`") {
		return false
	}
	fence.marker = marker
	fence.length = run
	return true
}

// ExtractTags finds all #tags outside fenced code blocks.
func ExtractTags(markdown string) []string {
	var tags []string
	seen := make(map[string]bool)

	lines := strings.Split(markdown, "\n")
	var fence codeFence

	for _, line := range lines {
		if skipCodeFence(line, &fence) {
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

// ExtractWikiLinks finds all [[note title]] references outside fenced code blocks.
func ExtractWikiLinks(markdown string) []string {
	var links []string
	seen := make(map[string]bool)

	lines := strings.Split(markdown, "\n")
	var fence codeFence

	for _, line := range lines {
		if skipCodeFence(line, &fence) {
			continue
		}

		matches := wikiLinkRegex.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) > 1 {
				target := strings.TrimSpace(m[1])
				if target != "" && !seen[target] {
					seen[target] = true
					links = append(links, target)
				}
			}
		}
	}

	return links
}

// BacklinkMention describes an occurrence of a target note title within a document.
type BacklinkMention struct {
	Snippet  string
	IsLinked bool
}

// FindBacklinkMentions finds references or mentions of targetTitle in markdown text.
func FindBacklinkMentions(markdown, targetTitle string) []BacklinkMention {
	targetTitle = strings.TrimSpace(targetTitle)
	if targetTitle == "" || targetTitle == "未命名笔记" {
		return nil
	}

	var results []BacklinkMention
	lines := strings.Split(markdown, "\n")
	var fence codeFence

	for _, line := range lines {
		if skipCodeFence(line, &fence) {
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !strings.Contains(trimmed, targetTitle) {
			continue
		}

		matches := wikiLinkRegex.FindAllStringSubmatch(trimmed, -1)
		isLinked := false
		for _, m := range matches {
			if len(m) > 1 && strings.TrimSpace(m[1]) == targetTitle {
				isLinked = true
				break
			}
		}

		snippet := trimmed
		runes := []rune(snippet)
		if len(runes) > 120 {
			snippet = string(runes[:120]) + "…"
		}

		results = append(results, BacklinkMention{
			Snippet:  snippet,
			IsLinked: isLinked,
		})
	}

	return results
}

// LinkMention converts the first unlinked mention of targetTitle into [[targetTitle]].
func LinkMention(markdown, targetTitle string) (string, bool) {
	targetTitle = strings.TrimSpace(targetTitle)
	if targetTitle == "" || targetTitle == "未命名笔记" {
		return markdown, false
	}

	lines := strings.Split(markdown, "\n")
	var fence codeFence
	replaced := false

	for i, line := range lines {
		if skipCodeFence(line, &fence) {
			continue
		}
		if replaced || !strings.Contains(line, targetTitle) {
			continue
		}

		newLine, ok := replaceUnlinkedMention(line, targetTitle)
		if ok {
			lines[i] = newLine
			replaced = true
			break
		}
	}

	if !replaced {
		return markdown, false
	}
	return strings.Join(lines, "\n"), true
}

func replaceUnlinkedMention(line, targetTitle string) (string, bool) {
	idx := 0
	for {
		pos := strings.Index(line[idx:], targetTitle)
		if pos < 0 {
			return line, false
		}
		matchStart := idx + pos
		matchEnd := matchStart + len(targetTitle)

		openBracket := strings.LastIndex(line[:matchStart], "[[")
		closeBracket := strings.LastIndex(line[:matchStart], "]]")
		if openBracket != -1 && (closeBracket == -1 || closeBracket < openBracket) {
			afterClose := strings.Index(line[matchEnd:], "]]")
			if afterClose != -1 {
				idx = matchEnd
				continue
			}
		}

		result := line[:matchStart] + "[[" + targetTitle + "]]" + line[matchEnd:]
		return result, true
	}
}

// ExtractHeadings finds headings for document outline.
func ExtractHeadings(markdown string) []model.HeadingItem {
	var headings []model.HeadingItem
	lines := strings.Split(markdown, "\n")
	var fence codeFence

	for _, line := range lines {
		if skipCodeFence(line, &fence) {
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
	var fence codeFence

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if skipCodeFence(line, &fence) {
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
