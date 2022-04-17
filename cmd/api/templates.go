package main

import (
	"blogalusta/internal/data"
	"blogalusta/internal/forms"
	"fmt"
	"github.com/gosimple/slug"
	"html/template"
	"io/fs"
	"path/filepath"
	"time"
)

type templateData struct {
	AuthenticatedUser *data.User
	CSRFToken         string
	Flash             string
	CurrentYear       int
	Form              *forms.Form
	Publication       *data.Publication
	Publications      *data.Publications
	Pending           []*data.User
	IsWriter          bool
	IsSubscribed      bool
	Writers           []*data.User
	Article           *data.Article
	Articles          []*data.Article
	HTML              template.HTML
	ProfileUser       *data.User
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
	} else if diff.Hours() < 2 {
		return fmt.Sprintf("%d hour ago", int(diff.Hours()))
	} else if diff.Hours() < 24 {
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	} else {
		return t.UTC().Format("02 Jan")
	}
}

func rfc3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func userURL(user *data.User) string {
	return fmt.Sprintf("/user/%s-%d", slug.Make(user.Name), user.ID)
}

func userPic(user *data.User) string {
	if !user.ImageID.Valid {
		return fmt.Sprintf("/img/%d.jpg", 0)
	}
	return fmt.Sprintf("/img/%d.jpg", user.ImageID.Int64)
}

func userIn(user *data.User, users []*data.User) bool {
	for _, u := range users {
		if user.ID == u.ID {
			return true
		}
	}

	return false
}

var functions = template.FuncMap{
	"humanDate": humanDate,
	"rfc3339":   rfc3339,
	"userURL":   userURL,
	"userPic":   userPic,
	"userIn":    userIn,
}

func newTemplateCache(dir string) (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := glob(dir, "*.page.gohtml")
	if err != nil {
		return nil, err
	}

	layouts, err := glob(dir, "*.layout.gohtml")
	if err != nil {
		return nil, err
	}

	partials, err := glob(dir, "*.partial.gohtml")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		ts, err := template.New(name).Funcs(functions).ParseFiles(layouts...)
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseFiles(page)
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseFiles(partials...)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}

func glob(dir string, fileMatch string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		name := filepath.Base(path)
		ok, err := filepath.Match(fileMatch, name)
		if err != nil {
			return err
		}
		if ok {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
