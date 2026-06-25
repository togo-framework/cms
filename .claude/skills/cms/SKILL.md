---
name: cms
description: Build content-managed pages in a togo app with the cms plugin — define content types, create/publish entries, route pages by slug, and manage navigation menus.
---

# togo cms

Use this skill to add content management to a togo app.

## Define a type, create & publish
```go
c, _ := cms.FromKernel(k)
c.DefineType(cms.ContentType{Name:"article", Fields:[]cms.Field{{Name:"title",Type:"text",Required:true},{Name:"body",Type:"richtext"}}})
e, _ := c.Create(cms.Entry{Type:"article", Slug:"hello", Fields:map[string]any{"title":"Hello"}}) // draft
c.Publish(e.ID)                  // or c.Publish(e.ID, scheduledTime)
page, _ := c.Page("hello")       // public lookup (published only)
```

## Rules
- Entries are validated against their content type (required fields, no unknown fields) — define the type first.
- `Page(slug)` and `Entries(type, true)` return **published, live** content only; drafts and future-scheduled entries are hidden. Use `Entries(type, false)` for the admin view.
- Pair with `richtext` for safe HTML body fields and `media` for image fields.

## REST
`/api/cms/{types,entries,menus}` + public `GET /api/cms/pages/{slug}`.
