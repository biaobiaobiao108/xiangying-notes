package markdown

import (
	"testing"
	"time"
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

func TestRelativeDate(t *testing.T) {
	now := time.Now()
	if d := RelativeDate(now); d != "刚刚" {
		t.Errorf("expected 刚刚, got %s", d)
	}
	if d := RelativeDate(now.Add(-24 * time.Minute)); d != "24 分钟前" {
		t.Errorf("expected 24 分钟前, got %s", d)
	}
	if d := RelativeDate(now.Add(-2 * time.Hour)); d != "2 小时前" {
		t.Errorf("expected 2 小时前, got %s", d)
	}
}
