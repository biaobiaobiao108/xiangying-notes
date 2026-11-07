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

Final line with #golang again.
\#escaped #周报 #项目`

	tags := ExtractTags(input)
	if len(tags) != 6 {
		t.Fatalf("expected 6 tags, got %d: %v", len(tags), tags)
	}
	if tags[0] != "golang" || tags[1] != "xiangying" || tags[2] != "设计" ||
		tags[3] != "escaped" || tags[4] != "周报" || tags[5] != "项目" {
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

func TestMismatchedFenceDoesNotEndCodeBlock(t *testing.T) {
	input := "````go\n~~~\n# hidden-heading\n#hidden-tag\n```\n# still-hidden\n````\n# visible-heading\n#visible-tag"

	if tags := ExtractTags(input); len(tags) != 1 || tags[0] != "visible" {
		t.Fatalf("ExtractTags() = %v, want only visible", tags)
	}
	if headings := ExtractHeadings(input); len(headings) != 1 || headings[0].Text != "visible-heading" {
		t.Fatalf("ExtractHeadings() = %v, want only visible-heading", headings)
	}
	if preview := ExcerptPreview(input, 3, 200); preview != "visible-heading #visible-tag" {
		t.Fatalf("ExcerptPreview() = %q, want visible content only", preview)
	}
}

func TestFenceMustCloseWithMatchingMarkerAndLength(t *testing.T) {
	input := "~~~~lang\n```\n# hidden\n~~~\n# still-hidden\n~~~~\n#shown"
	if tags := ExtractTags(input); len(tags) != 1 || tags[0] != "shown" {
		t.Fatalf("ExtractTags() = %v, want only shown", tags)
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

func TestLooksLikeMarkdown(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "heading", text: "# 标题\n正文", want: true},
		{name: "bold", text: "这是 **重点** 内容", want: true},
		{name: "list", text: "- 第一项\n- 第二项", want: true},
		{name: "quote", text: "> 引用", want: true},
		{name: "fence", text: "```go\nfmt.Println()\n```", want: true},
		{name: "link", text: "[象映](https://example.com)", want: true},
		{name: "plain", text: "今天天气不错。\n下午出去走走。", want: false},
		{name: "number", text: "2026. 年终总结", want: false},
		{name: "dash prose", text: "会议记录 - 讨论了预算", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := LooksLikeMarkdown(test.text); got != test.want {
				t.Fatalf("LooksLikeMarkdown(%q) = %v, want %v", test.text, got, test.want)
			}
		})
	}
}

func TestExtractWikiLinks(t *testing.T) {
	input := `# 测试
关联了 [[产品想法]] 和 [[架构设计|系统架构]]。
` + "```" + `
[[代码里的链接不应提取]]
` + "```" + `
重复引用：[[产品想法]] 与 [[收件箱笔记]]。`

	links := ExtractWikiLinks(input)
	if len(links) != 3 {
		t.Fatalf("expected 3 unique links, got %d: %v", len(links), links)
	}
	if links[0] != "产品想法" || links[1] != "架构设计" || links[2] != "收件箱笔记" {
		t.Errorf("unexpected links: %v", links)
	}
}

func TestFindBacklinkMentionsAndLinkMention(t *testing.T) {
	doc := `这是项目周报。
相关笔记: [[产品想法]]、本周工作安排
- 周三: 整理 产品想法
` + "```" + `
忽略代码块中的 产品想法
` + "```"

	mentions := FindBacklinkMentions(doc, "产品想法")
	if len(mentions) != 2 {
		t.Fatalf("expected 2 mentions, got %d: %+v", len(mentions), mentions)
	}
	if !mentions[0].IsLinked || mentions[0].Snippet != "相关笔记: [[产品想法]]、本周工作安排" {
		t.Errorf("unexpected first mention: %+v", mentions[0])
	}
	if mentions[1].IsLinked || mentions[1].Snippet != "- 周三: 整理 产品想法" {
		t.Errorf("unexpected second mention: %+v", mentions[1])
	}

	linkedDoc, ok := LinkMention(doc, "产品想法")
	if !ok {
		t.Fatal("expected LinkMention to succeed")
	}
	wantLinkedLine := "- 周三: 整理 [[产品想法]]"
	if !testingContains(linkedDoc, wantLinkedLine) {
		t.Errorf("expected linkedDoc to contain %q, got:\n%s", wantLinkedLine, linkedDoc)
	}
	// First linked line remains linked
	if !testingContains(linkedDoc, "相关笔记: [[产品想法]]、本周工作安排") {
		t.Errorf("first line was corrupted:\n%s", linkedDoc)
	}
}

func testingContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
