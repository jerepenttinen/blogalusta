package main

import (
	"blogalusta/internal/data"
	"blogalusta/internal/forms"
	"fmt"
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
	Articles          []*data.Article
	HTML              template.HTML
}

func humanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	now := time.Now().UTC()

	if t.UTC().Year() != now.Year() {
		return t.UTC().Format("02 Jan 2006")
	}

	diff := time.Now().UTC().Sub(t.UTC())

	if diff.Minutes() < 2 {
		return "Just now"
	} else if diff.Minutes() < 60 {
		return fmt.Sprintf("%d mins ago", int(diff.Minutes()))
	} else if diff.Hours() < 24 {
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	} else {
		return t.UTC().Format("02 Jan")
	}
}

func rfc3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

var functions = template.FuncMap{
	"humanDate": humanDate,
	"rfc3339":   rfc3339,
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
