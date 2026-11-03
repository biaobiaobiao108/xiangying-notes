package markdown

import (
	"testing"
)

func TestExtractTags(t *testing.T) {
	input := `# Title
This is a note with #golang and #xiangying tags.
Another tag: #设计.

` + "```" + `
#notatag in code fence
` + "```" + `

Final line with #golang again.`

	tags := ExtractTags(input)
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d: %v", len(tags), tags)
	}
	if tags[0] != "golang" || tags[1] != "xiangying" || tags[2] != "设计" {
		t.Errorf("unexpected tags: %v", tags)
	}
}

func TestExtractHeadings(t *testing.T) {
	input := `# Heading 1
Text
## Heading 2
More text
### Subheading
` + "```" + `
# Not a heading in code
` + "```"

	headings := ExtractHeadings(input)
	if len(headings) != 3 {
		t.Fatalf("expected 3 headings, got %d", len(headings))
	}
	if headings[0].Level != 1 || headings[0].Text != "Heading 1" {
		t.Errorf("heading 0 mismatch: %+v", headings[0])
	}
	if headings[1].Level != 2 || headings[1].Text != "Heading 2" {
		t.Errorf("heading 1 mismatch: %+v", headings[1])
	}
	if headings[2].Level != 3 || headings[2].Text != "Subheading" {
		t.Errorf("heading 2 mismatch: %+v", headings[2])
	}
}

func TestTasks(t *testing.T) {
	input := `# Tasks
- [ ] Task 1
- [x] Task 2
* [ ] Task 3`

	tasks := ExtractTasks(input, "n1", "Note 1", "Default")
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].Done != false || tasks[0].Text != "Task 1" || tasks[0].LineIndex != 1 {
		t.Errorf("task 0 mismatch: %+v", tasks[0])
	}
	if tasks[1].Done != true || tasks[1].Text != "Task 2" || tasks[1].LineIndex != 2 {
		t.Errorf("task 1 mismatch: %+v", tasks[1])
	}

	// Toggle task 0
	updated, ok := ToggleTaskInMarkdown(input, 1)
	if !ok {
		t.Fatal("failed to toggle task 0")
	}
	t1 := ExtractTasks(updated, "n1", "Note 1", "Default")
	if !t1[0].Done {
		t.Errorf("expected task 0 to be done after toggle")
	}
}

func TestExcerptPreview(t *testing.T) {
	input := `# Welcome
This is **bold** text and ` + "`code`" + `.
> A quote block
- List item`

	preview := ExcerptPreview(input, 2, 50)
	if preview == "" {
		t.Errorf("preview should not be empty")
	}
}
