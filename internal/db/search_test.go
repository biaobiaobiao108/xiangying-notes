package db

import (
	"reflect"
	"testing"

	"xiangying/internal/model"
)

func TestSearchConditionUsesTrigramForThreeUnicodeCharacters(t *testing.T) {
	condition, args := searchCondition("中文搜索")
	if condition != `n.rowid IN (SELECT rowid FROM notes_fts WHERE notes_fts MATCH ?)` {
		t.Fatalf("condition = %q", condition)
	}
	if !reflect.DeepEqual(args, []any{`"中文搜索"`}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestSearchConditionFallsBackToLikeForShortQueries(t *testing.T) {
	condition, args := searchCondition(" 中_ ")
	if condition != `(n.title LIKE ? ESCAPE '\' OR n.content_markdown LIKE ? ESCAPE '\')` {
		t.Fatalf("condition = %q", condition)
	}
	if !reflect.DeepEqual(args, []any{"%中\\_%", "%中\\_%"}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestSearchConditionEscapesFTSPhraseSyntax(t *testing.T) {
	_, args := searchCondition(`a"b`)
	if !reflect.DeepEqual(args, []any{`"a""b"`}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestListNotePageSearchSemanticsAndFTSSync(t *testing.T) {
	store, err := Open(t.TempDir() + `/search.sqlite`)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	notebooks, err := store.ListNotebooks()
	if err != nil || len(notebooks) == 0 {
		t.Fatalf("ListNotebooks: %v", err)
	}
	created, err := store.CreateNote(notebooks[0].ID, `search a"b note`, "中文目标XYZ; literal %_ token")
	if err != nil {
		t.Fatal(err)
	}
	assertFound := func(query string, want bool) {
		t.Helper()
		page, err := store.ListNotePage(model.ViewAll, "", "", query, 100, nil)
		if err != nil {
			t.Fatalf("search %q: %v", query, err)
		}
		found := false
		for _, note := range page.Notes {
			if note.ID == created.ID {
				found = true
			}
		}
		if found != want {
			t.Fatalf("search %q found=%v, want %v", query, found, want)
		}
	}
	assertFound("文目标", true)    // Chinese substring through FTS5 trigram.
	assertFound("h a\"b", true) // Quotes are treated as literal FTS phrase text.
	assertFound("%_", true)     // Short query uses LIKE with wildcard escaping.
	assertFound("不存在词", false)

	if err := store.UpdateNote(created.ID, created.Title, "更新后的检索词ABC"); err != nil {
		t.Fatal(err)
	}
	assertFound("文目标", false)
	assertFound("索词A", true)
	if err := store.DeletePermanently(created.ID); err != nil {
		t.Fatal(err)
	}
	assertFound("索词A", false)
}
