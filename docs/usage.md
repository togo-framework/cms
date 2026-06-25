# cms — usage

## Content types
```go
c, _ := cms.FromKernel(k)
c.DefineType(cms.ContentType{Name:"article", Fields:[]cms.Field{
    {Name:"title", Type:"text", Required:true},
    {Name:"body",  Type:"richtext"},
}})
```
Entries are validated against their type: required fields must be present, and unknown fields are rejected.

## Entries & publishing
```go
e, _ := c.Create(cms.Entry{Type:"article", Slug:"hello", Fields:map[string]any{"title":"Hello"}}) // draft
c.Publish(e.ID)                                  // live now
c.Publish(e.ID, time.Now().Add(24*time.Hour))    // scheduled
c.Unpublish(e.ID); c.Update(e.ID, fields, "new-slug"); c.Delete(e.ID)
c.Entries("article", true)                       // published only (public)
c.Entries("article", false)                      // all (admin)
```

## Pages
```go
page, ok := c.Page("hello")  // published entry by slug; drafts/future are hidden
```

## Menus
```go
c.SetMenu(cms.Menu{Name:"main", Items:[]cms.MenuItem{
    {Label:"Home", URL:"/"},
    {Label:"Docs", URL:"/docs", Children:[]cms.MenuItem{{Label:"Intro", URL:"/docs/intro"}}},
}})
m, _ := c.Menu("main")
```

## REST
`/api/cms/types`, `/api/cms/entries` (`?type=&published=`), `/api/cms/entries/{id}/publish`, `DELETE /api/cms/entries/{id}`, public `GET /api/cms/pages/{slug}`, `/api/cms/menus`.

## Persistence
Default is a bounded in-memory store. Implement a DB-backed store behind the same Service methods for production.
