// Package cms is a lightweight headless CMS for togo — content types, entries,
// pages, and menus with a draft/publish workflow (Statamic / Wagtail-lite).
//
// Define content types with typed fields, create entries (validated against the
// type), publish them (with optional scheduled publish), route pages by slug,
// and build navigation menus. A public read API exposes only published content;
// the admin API sees everything.
package cms

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/togo-framework/togo"
)

// Field is a typed field on a content type.
type Field struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // text | richtext | number | bool | date | media | json
	Required bool   `json:"required,omitempty"`
}

// ContentType describes the shape of entries.
type ContentType struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

// Entry status values.
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

// Entry is one piece of content of a given type.
type Entry struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Slug        string         `json:"slug"`
	Status      string         `json:"status"`
	Fields      map[string]any `json:"fields"`
	Author      string         `json:"author,omitempty"`
	PublishedAt *time.Time     `json:"published_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// MenuItem is a navigation node (recursive).
type MenuItem struct {
	Label    string     `json:"label"`
	URL      string     `json:"url"`
	Children []MenuItem `json:"children,omitempty"`
}

// Menu is a named navigation tree.
type Menu struct {
	Name  string     `json:"name"`
	Items []MenuItem `json:"items"`
}

// Service is the CMS runtime stored on the kernel (k.Get("cms")).
type Service struct {
	mu      sync.RWMutex
	types   map[string]ContentType
	entries map[string]*Entry
	menus   map[string]*Menu
	seq     int
}

func init() {
	togo.RegisterProviderFunc("cms", togo.PriorityLate+10, func(k *togo.Kernel) error {
		s := newService()
		k.Set("cms", s)
		if k.Router != nil {
			s.mountRoutes(k.Router)
		}
		return nil
	})
}

func newService() *Service {
	return &Service{
		types:   map[string]ContentType{},
		entries: map[string]*Entry{},
		menus:   map[string]*Menu{},
	}
}

// FromKernel returns the CMS Service.
func FromKernel(k *togo.Kernel) (*Service, bool) {
	v, ok := k.Get("cms")
	if !ok {
		return nil, false
	}
	s, ok := v.(*Service)
	return s, ok
}

// DefineType registers (or replaces) a content type.
func (s *Service) DefineType(ct ContentType) error {
	if ct.Name == "" {
		return fmt.Errorf("cms: content type needs a name")
	}
	s.mu.Lock()
	s.types[ct.Name] = ct
	s.mu.Unlock()
	return nil
}

// Types lists the registered content types.
func (s *Service) Types() []ContentType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ContentType, 0, len(s.types))
	for _, t := range s.types {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// validate checks entry fields against the content type (if the type is known).
func (s *Service) validate(typeName string, fields map[string]any) error {
	ct, ok := s.types[typeName]
	if !ok {
		return nil // unknown type → no schema to validate against
	}
	allowed := map[string]bool{}
	for _, f := range ct.Fields {
		allowed[f.Name] = true
		if f.Required {
			if v, present := fields[f.Name]; !present || v == nil || v == "" {
				return fmt.Errorf("cms: field %q is required", f.Name)
			}
		}
	}
	for name := range fields {
		if !allowed[name] {
			return fmt.Errorf("cms: unknown field %q for type %q", name, typeName)
		}
	}
	return nil
}

// Create makes a new entry (draft by default). Validates against the type.
func (s *Service) Create(e Entry) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.validate(e.Type, e.Fields); err != nil {
		return nil, err
	}
	s.seq++
	now := time.Now()
	rec := &Entry{
		ID: fmt.Sprintf("ent_%d", s.seq), Type: e.Type, Slug: e.Slug,
		Status: StatusDraft, Fields: e.Fields, Author: e.Author,
		CreatedAt: now, UpdatedAt: now,
	}
	if e.Status == StatusPublished {
		rec.Status = StatusPublished
		rec.PublishedAt = &now
	}
	s.entries[rec.ID] = rec
	return rec, nil
}

// Update modifies an entry's slug/fields (re-validates).
func (s *Service) Update(id string, fields map[string]any, slug string) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[id]
	if !ok {
		return nil, fmt.Errorf("cms: no entry %q", id)
	}
	if fields != nil {
		if err := s.validate(e.Type, fields); err != nil {
			return nil, err
		}
		e.Fields = fields
	}
	if slug != "" {
		e.Slug = slug
	}
	e.UpdatedAt = time.Now()
	return e, nil
}

// Publish marks an entry published (now, or at a scheduled time).
func (s *Service) Publish(id string, at ...time.Time) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[id]
	if !ok {
		return nil, fmt.Errorf("cms: no entry %q", id)
	}
	t := time.Now()
	if len(at) > 0 {
		t = at[0]
	}
	e.PublishedAt = &t
	if !t.After(time.Now()) {
		e.Status = StatusPublished
	}
	e.UpdatedAt = time.Now()
	return e, nil
}

// Unpublish reverts an entry to draft.
func (s *Service) Unpublish(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[id]
	if !ok {
		return fmt.Errorf("cms: no entry %q", id)
	}
	e.Status = StatusDraft
	e.PublishedAt = nil
	e.UpdatedAt = time.Now()
	return nil
}

// Delete removes an entry.
func (s *Service) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[id]; !ok {
		return false
	}
	delete(s.entries, id)
	return true
}

// Get returns an entry by id.
func (s *Service) Get(id string) (*Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[id]
	return e, ok
}

// isLive reports whether an entry is currently published (scheduled time passed).
func isLive(e *Entry) bool {
	return e.Status == StatusPublished && e.PublishedAt != nil && !e.PublishedAt.After(time.Now())
}

// Entries lists entries; publishedOnly filters to live content (public view).
func (s *Service) Entries(typeName string, publishedOnly bool) []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Entry
	for _, e := range s.entries {
		if typeName != "" && e.Type != typeName {
			continue
		}
		if publishedOnly && !isLive(e) {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// Page returns a published entry by slug (public lookup).
func (s *Service) Page(slug string) (*Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.entries {
		if e.Slug == slug && isLive(e) {
			return e, true
		}
	}
	return nil, false
}

// SetMenu creates or replaces a navigation menu.
func (s *Service) SetMenu(m Menu) {
	s.mu.Lock()
	s.menus[m.Name] = &m
	s.mu.Unlock()
}

// Menu returns a menu by name.
func (s *Service) Menu(name string) (*Menu, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.menus[name]
	return m, ok
}

// Menus lists all menus.
func (s *Service) Menus() []*Menu {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Menu, 0, len(s.menus))
	for _, m := range s.menus {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// DeleteMenu removes a menu.
func (s *Service) DeleteMenu(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.menus[name]; !ok {
		return false
	}
	delete(s.menus, name)
	return true
}

func (s *Service) mountRoutes(r chi.Router) {
	r.Route("/api/cms", func(r chi.Router) {
		// content types
		r.Get("/types", func(w http.ResponseWriter, req *http.Request) { writeJSON(w, 200, s.Types()) })
		r.Post("/types", func(w http.ResponseWriter, req *http.Request) {
			var ct ContentType
			if err := json.NewDecoder(req.Body).Decode(&ct); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			if err := s.DefineType(ct); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 201, ct)
		})
		// entries (admin: all)
		r.Get("/entries", func(w http.ResponseWriter, req *http.Request) {
			writeJSON(w, 200, s.Entries(req.URL.Query().Get("type"), req.URL.Query().Get("published") == "true"))
		})
		r.Post("/entries", func(w http.ResponseWriter, req *http.Request) {
			var e Entry
			if err := json.NewDecoder(req.Body).Decode(&e); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			rec, err := s.Create(e)
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 201, rec)
		})
		r.Post("/entries/{id}/publish", func(w http.ResponseWriter, req *http.Request) {
			rec, err := s.Publish(chi.URLParam(req, "id"))
			if err != nil {
				writeJSON(w, 404, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, rec)
		})
		r.Delete("/entries/{id}", func(w http.ResponseWriter, req *http.Request) {
			writeJSON(w, 200, map[string]bool{"ok": s.Delete(chi.URLParam(req, "id"))})
		})
		// public page lookup (published only)
		r.Get("/pages/{slug}", func(w http.ResponseWriter, req *http.Request) {
			if e, ok := s.Page(chi.URLParam(req, "slug")); ok {
				writeJSON(w, 200, e)
				return
			}
			writeJSON(w, 404, map[string]string{"error": "not found"})
		})
		// menus
		r.Get("/menus", func(w http.ResponseWriter, req *http.Request) { writeJSON(w, 200, s.Menus()) })
		r.Post("/menus", func(w http.ResponseWriter, req *http.Request) {
			var m Menu
			if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			s.SetMenu(m)
			writeJSON(w, 201, m)
		})
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
