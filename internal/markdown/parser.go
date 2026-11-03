package markdown

import (
	"regexp"
	"strings"
	"unicode"

	"xiangying/internal/model"
)

var (
	tagRegex     = regexp.MustCompile(`(?:\s|^)#([a-zA-Z0-9_\p{Han}]+)`)
	headingRegex = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	taskRegex    = regexp.MustCompile(`^(\s*[-*+]\s+\[)([ xX])(\]\s+.*)$`)
)

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

// ExtractTasks finds all task checkbox items from markdown lines.
func ExtractTasks(markdown string, noteID, noteTitle, notebookName string) []model.TaskItem {
	var tasks []model.TaskItem
	lines := strings.Split(markdown, "\n")
	inCodeBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		if m := taskRegex.FindStringSubmatch(line); len(m) == 4 {
			isDone := m[2] == "x" || m[2] == "X"
			text := strings.TrimSpace(strings.TrimPrefix(m[3], "] "))
			if text == "" {
				text = "未命名任务"
			}
			tasks = append(tasks, model.TaskItem{
				NoteID:       noteID,
				NoteTitle:    noteTitle,
				NotebookName: notebookName,
				Text:         text,
				Done:         isDone,
				LineIndex:    i,
			})
		}
	}

	return tasks
}

// ToggleTaskInMarkdown toggles the task checkbox at lineIndex.
func ToggleTaskInMarkdown(markdown string, lineIndex int) (string, bool) {
	lines := strings.Split(markdown, "\n")
	if lineIndex < 0 || lineIndex >= len(lines) {
		return markdown, false
	}

	line := lines[lineIndex]
	m := taskRegex.FindStringSubmatch(line)
	if len(m) != 4 {
		return markdown, false
	}

	newMark := "x"
	if m[2] == "x" || m[2] == "X" {
		newMark = " "
	}

	lines[lineIndex] = m[1] + newMark + m[3]
	return strings.Join(lines, "\n"), true
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
		cleaned = taskRegex.ReplaceAllString(cleaned, "$3")
		cleaned = strings.TrimPrefix(cleaned, "] ")
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

	result := strings.Join(cleanParts, " · ")
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
