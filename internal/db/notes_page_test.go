package db

import (
	"path/filepath"
	"testing"

	"xiangying/internal/model"
)

func TestListNotePageUsesStableCursorAndSummaries(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "notes.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if _, err := store.CreateNote(notebooks[0].ID, "分页测试", "摘要正文 #page"); err != nil {
			t.Fatal(err)
		}
	}

	seen := map[string]bool{}
	var cursor *NoteCursor
	pageCount := 0
	for {
		page, err := store.ListNotePage(model.ViewAll, "", "", "", 2, cursor)
		if err != nil {
			t.Fatal(err)
		}
		pageCount++
		if len(page.Notes) > 2 {
			t.Fatalf("page has %d notes, limit is 2", len(page.Notes))
		}
		for _, note := range page.Notes {
			if seen[note.ID] {
				t.Fatalf("note %s appeared on multiple pages", note.ID)
			}
			seen[note.ID] = true
			if note.ContentMarkdown != "" {
				t.Fatalf("summary unexpectedly loaded body: %q", note.ContentMarkdown)
			}
		}
		if !page.HasMore {
			if page.Next != nil {
				t.Fatal("final page returned a next cursor")
			}
			break
		}
		if page.Next == nil {
			t.Fatal("page with more results has no cursor")
		}
		cursor = page.Next
	}
	if pageCount < 2 || len(seen) != 11 { // six seeded notes and five new notes
		t.Fatalf("got %d pages and %d unique notes, want multiple pages and 11 notes", pageCount, len(seen))
	}
}

func TestGetNoteForViewHonorsFilters(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "notes.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	note, err := store.CreateNote(notebooks[0].ID, "筛选目标", "正文 #目标标签")
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.GetNoteForView(note.ID, model.ViewNotebook, notebooks[0].ID, "", "")
	if err != nil || !ok {
		t.Fatalf("GetNoteForView() = (%v, %v), want note and true", got, err)
	}
	if got.ContentMarkdown != "" || got.Preview == "" {
		t.Fatalf("expected summary without body, got body=%q preview=%q", got.ContentMarkdown, got.Preview)
	}
	if _, ok, err := store.GetNoteForView(note.ID, model.ViewTag, "", "missing", ""); err != nil || ok {
		t.Fatalf("nonmatching tag returned ok=%v err=%v", ok, err)
	}
}
