package db

import (
	"reflect"
	"strings"
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

func TestSearchConditionUsesShortIndexAndEscapesOverflowPattern(t *testing.T) {
	condition, args := searchCondition(" 中_ ")
	if !strings.Contains(condition, "notes_short_fts") || !strings.Contains(condition, "short_index_overflow") {
		t.Fatalf("condition = %q", condition)
	}
	if !reflect.DeepEqual(args, []any{`"b004e2d00005f"`, "%中\\_%", "%中\\_%"}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestSearchConditionEscapesFTSPhraseSyntax(t *testing.T) {
	_, args := searchCondition(`a"b`)
	if !reflect.DeepEqual(args, []any{`"a""b"`}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildShortSearchTermsUsesLiteralASCIIFoldedTokens(t *testing.T) {
	terms, overflow := buildShortSearchTerms("ABC", "中文%_")
	if overflow {
		t.Fatal("short text unexpectedly overflowed")
	}
	for _, want := range []string{shortSearchKey([]rune("a")), shortSearchKey([]rune("ab")), shortSearchKey([]rune("文%")), shortSearchKey([]rune("%_"))} {
		if !strings.Contains(" "+terms+" ", " "+want+" ") {
			t.Fatalf("terms do not contain %q: %s", want, terms)
		}
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
	assertFound("目标", true)     // A two-rune Chinese substring through the short index.
	assertFound("文", true)      // A one-rune Chinese substring through the short index.
	assertFound("SEARCH", true) // ASCII folding matches the LIKE fallback behavior.
	assertFound("h a\"b", true) // Quotes are treated as literal FTS phrase text.
	assertFound("%_", true)     // Special characters are indexed literally.
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
	var indexedRows int
	if err := store.db.QueryRow("SELECT count(*) FROM notes_short_fts f LEFT JOIN notes n ON n.rowid = f.rowid WHERE n.id IS NULL").Scan(&indexedRows); err != nil {
		t.Fatal(err)
	}
	if indexedRows != 0 {
		t.Fatalf("deleted note left %d short-index rows", indexedRows)
	}
}

func TestShortSearchOverflowFallsBackWithoutMissingMatches(t *testing.T) {
	store, err := Open(t.TempDir() + `/overflow.sqlite`)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	notebooks, err := store.ListNotebooks()
	if err != nil || len(notebooks) == 0 {
		t.Fatalf("ListNotebooks: %v", err)
	}
	content := strings.Repeat("ab", maxShortSearchRunes/2+1) + "独特"
	note, err := store.CreateNote(notebooks[0].ID, "长文本", content)
	if err != nil {
		t.Fatal(err)
	}
	var overflowCount int
	if err := store.db.QueryRow("SELECT count(*) FROM short_index_overflow WHERE note_id = ?", note.ID).Scan(&overflowCount); err != nil || overflowCount != 1 {
		t.Fatalf("overflow marker count=%d err=%v", overflowCount, err)
	}
	page, err := store.ListNotePage(model.ViewAll, "", "", "独特", 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range page.Notes {
		if got.ID == note.ID {
			goto found
		}
	}
	t.Fatal("short search missed match beyond indexed text")

found:
	if err := store.UpdateNote(note.ID, note.Title, "已经缩短"); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow("SELECT count(*) FROM short_index_overflow WHERE note_id = ?", note.ID).Scan(&overflowCount); err != nil || overflowCount != 0 {
		t.Fatalf("overflow marker after shortening=%d err=%v", overflowCount, err)
	}
	page, err = store.ListNotePage(model.ViewAll, "", "", "独特", 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range page.Notes {
		if got.ID == note.ID {
			t.Fatal("stale overflow search result after shortening")
		}
	}
}

func TestShortSearchUnicodeAndReusedRowID(t *testing.T) {
	store, err := Open(t.TempDir() + `/short-unicode.sqlite`)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	notebooks, err := store.ListNotebooks()
	if err != nil || len(notebooks) == 0 {
		t.Fatalf("ListNotebooks: %v", err)
	}
	old, err := store.CreateNote(notebooks[0].ID, "Δ猫🐘!", "正文")
	if err != nil {
		t.Fatal(err)
	}
	assert := func(query string, want bool) {
		t.Helper()
		page, err := store.ListNotePage(model.ViewAll, "", "", query, 100, nil)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, note := range page.Notes {
			if note.ID == old.ID {
				found = true
			}
		}
		if found != want {
			t.Fatalf("search %q found=%v, want %v", query, found, want)
		}
	}
	assert("🐘", true)
	assert("!", true)
	assert("δ", false) // SQLite LIKE does not case-fold non-ASCII characters.
	var oldRowID int64
	if err := store.db.QueryRow("SELECT rowid FROM notes WHERE id=?", old.ID).Scan(&oldRowID); err != nil {
		t.Fatal(err)
	}
	if err := store.DeletePermanently(old.ID); err != nil {
		t.Fatal(err)
	}
	newNote, err := store.CreateNote(notebooks[0].ID, "其他", "无关内容")
	if err != nil {
		t.Fatal(err)
	}
	var newRowID int64
	if err := store.db.QueryRow("SELECT rowid FROM notes WHERE id=?", newNote.ID).Scan(&newRowID); err != nil {
		t.Fatal(err)
	}
	if newRowID == oldRowID {
		page, err := store.ListNotePage(model.ViewAll, "", "", "🐘", 100, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, note := range page.Notes {
			if note.ID == newNote.ID {
				t.Fatal("reused rowid produced a stale short-search match")
			}
		}
	}
}
