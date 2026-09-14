<div align="center">
  <picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/togo-mark-dark.svg" /><img src=".github/assets/togo-mark.svg" alt="ToGO" height="64" /></picture>
  <h1>togo-framework/cms</h1>
  <p>
    <a href="https://to-go.dev/marketplace"><img src="https://img.shields.io/badge/marketplace-to--go.dev-1F8A99" alt="marketplace" /></a>
    <a href="https://pkg.go.dev/github.com/togo-framework/cms"><img src="https://pkg.go.dev/badge/github.com/togo-framework/cms.svg" alt="pkg.go.dev" /></a>
    <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT" />
  </p>
  <p><strong>Lightweight headless CMS for <a href="https://to-go.dev">togo</a> — content types, entries, pages & menus.</strong></p>
</div>

## Install

```bash
togo install togo-framework/cms
```

A headless CMS (Statamic / Wagtail-lite): define **content types** with typed fields, create **entries** validated against the type, **draft → publish** (with scheduled publish), route **pages** by slug, and build navigation **menus**. The public read API exposes only published content; the admin API sees everything.

## Usage

```go
c, _ := cms.FromKernel(k)

// 1. Define a content type
c.DefineType(cms.ContentType{Name: "article", Fields: []cms.Field{
    {Name: "title", Type: "text", Required: true},
    {Name: "body",  Type: "richtext"},
}})

// 2. Create an entry (draft, validated against the type)
e, _ := c.Create(cms.Entry{Type: "article", Slug: "hello-world",
    Fields: map[string]any{"title": "Hello", "body": "<p>Hi</p>"}})

// 3. Publish it (now, or schedule)
c.Publish(e.ID)                          // live now
c.Publish(e.ID, time.Now().Add(24*time.Hour)) // scheduled

// 4. Read a published page by slug
page, ok := c.Page("hello-world")
```

## REST API

| Method | Path | Description |
|---|---|---|
| `GET/POST` | `/api/cms/types` | list / define content types |
| `GET/POST` | `/api/cms/entries` | list (`?type=&published=true`) / create |
| `POST` | `/api/cms/entries/{id}/publish` | publish an entry |
| `DELETE` | `/api/cms/entries/{id}` | delete |
| `GET` | `/api/cms/pages/{slug}` | **public** — published page by slug |
| `GET/POST` | `/api/cms/menus` | list / set navigation menus |

## Configuration

No required env. Content is held in a bounded in-memory store with a pluggable
seam — swap a DB-backed store for persistence. Entries validate against their
content type (required fields + no unknown fields).

---

<div align="center">
  <h3>Premium sponsors</h3>
  <p>
    <a href="https://id8media.com"><strong>ID8 Media</strong></a> &nbsp;·&nbsp;
    <a href="https://one-studio.co"><strong>One Studio</strong></a>
  </p>
  <p><sub>Support togo — <a href="https://github.com/sponsors/fadymondy">become a sponsor</a>.</sub></p>
</div>
