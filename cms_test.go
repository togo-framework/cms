package cms

import (
	"testing"
	"time"
)

func newTestService() *Service { return newService() }

func defineArticle(t *testing.T, s *Service) {
	t.Helper()
	err := s.DefineType(ContentType{
		Name: "article",
		Fields: []Field{
			{Name: "title", Type: "text", Required: true},
			{Name: "body", Type: "richtext"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDefineTypeAndValidate(t *testing.T) {
	s := newTestService()
	defineArticle(t, s)
	if len(s.Types()) != 1 {
		t.Fatalf("Types = %d, want 1", len(s.Types()))
	}
	// Missing required field is rejected.
	if _, err := s.Create(Entry{Type: "article", Slug: "x", Fields: map[string]any{"body": "hi"}}); err == nil {
		t.Fatal("expected required-field error")
	}
	// Unknown field is rejected.
	if _, err := s.Create(Entry{Type: "article", Slug: "x", Fields: map[string]any{"title": "T", "nope": 1}}); err == nil {
		t.Fatal("expected unknown-field error")
	}
	// Valid create succeeds.
	if _, err := s.Create(Entry{Type: "article", Slug: "ok", Fields: map[string]any{"title": "T"}}); err != nil {
		t.Fatalf("valid create failed: %v", err)
	}
}

func TestDraftPublishWorkflow(t *testing.T) {
	s := newTestService()
	defineArticle(t, s)
	e, err := s.Create(Entry{Type: "article", Slug: "hello", Fields: map[string]any{"title": "Hello"}})
	if err != nil {
		t.Fatal(err)
	}
	if e.Status != StatusDraft {
		t.Fatalf("new entry should be draft, got %s", e.Status)
	}
	// Draft is not in the published list.
	if len(s.Entries("article", true)) != 0 {
		t.Fatal("draft should not appear in published list")
	}
	if _, ok := s.Page("hello"); ok {
		t.Fatal("draft should not be reachable as a page")
	}
	// Publish → appears.
	if _, err := s.Publish(e.ID); err != nil {
		t.Fatal(err)
	}
	if len(s.Entries("article", true)) != 1 {
		t.Fatal("published entry missing from published list")
	}
	if p, ok := s.Page("hello"); !ok || p.ID != e.ID {
		t.Fatal("published page not reachable by slug")
	}
	// Unpublish → gone again.
	if err := s.Unpublish(e.ID); err != nil {
		t.Fatal(err)
	}
	if len(s.Entries("article", true)) != 0 {
		t.Fatal("unpublished entry still listed")
	}
}

func TestScheduledPublishNotYetLive(t *testing.T) {
	s := newTestService()
	defineArticle(t, s)
	e, _ := s.Create(Entry{Type: "article", Slug: "future", Fields: map[string]any{"title": "Soon"}})
	// Schedule in the future → not live yet.
	if _, err := s.Publish(e.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Page("future"); ok {
		t.Fatal("future-scheduled entry should not be live yet")
	}
	// Schedule in the past → live.
	if _, err := s.Publish(e.ID, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Page("future"); !ok {
		t.Fatal("past-scheduled entry should be live")
	}
}

func TestUpdateAndDelete(t *testing.T) {
	s := newTestService()
	defineArticle(t, s)
	e, _ := s.Create(Entry{Type: "article", Slug: "a", Fields: map[string]any{"title": "A"}})
	upd, err := s.Update(e.ID, map[string]any{"title": "B"}, "b")
	if err != nil || upd.Slug != "b" || upd.Fields["title"] != "B" {
		t.Fatalf("update = %+v, %v", upd, err)
	}
	// Update with an invalid field is rejected.
	if _, err := s.Update(e.ID, map[string]any{"bad": 1}, ""); err == nil {
		t.Fatal("expected validation error on update")
	}
	if !s.Delete(e.ID) {
		t.Fatal("delete failed")
	}
	if s.Delete(e.ID) {
		t.Fatal("second delete should be false")
	}
}

func TestMenus(t *testing.T) {
	s := newTestService()
	s.SetMenu(Menu{Name: "main", Items: []MenuItem{
		{Label: "Home", URL: "/"},
		{Label: "Docs", URL: "/docs", Children: []MenuItem{{Label: "Intro", URL: "/docs/intro"}}},
	}})
	m, ok := s.Menu("main")
	if !ok || len(m.Items) != 2 || len(m.Items[1].Children) != 1 {
		t.Fatalf("menu = %+v", m)
	}
	if len(s.Menus()) != 1 {
		t.Fatalf("Menus = %d", len(s.Menus()))
	}
	if !s.DeleteMenu("main") || s.DeleteMenu("main") {
		t.Fatal("menu delete semantics wrong")
	}
}
