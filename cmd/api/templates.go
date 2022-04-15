package main

import (
	"blogalusta/internal/data"
	"blogalusta/internal/forms"
	"html/template"
	"path/filepath"
	"time"
)

type templateData struct {
	AuthenticatedUser *data.User
	CSRFToken         string
	CurrentYear       int
	Form              *forms.Form
	Publication       *data.Publication
	Publications      *data.Publications
	IsWriter          bool
	Article           *data.Article
	HTML              template.HTML
}

func humanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("02 Jan 2006 at 15:04")
}

var functions = template.FuncMap{
	"humanDate": humanDate,
}

func newTemplateCache(dir string) (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := filepath.Glob(filepath.Join(dir, "*.page.gohtml"))
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		ts, err := template.New(name).Funcs(functions).ParseFiles(page)
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseGlob(filepath.Join(dir, "*.layout.gohtml"))
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseGlob(filepath.Join(dir, "*.partial.gohtml"))
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
