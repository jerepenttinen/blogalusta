package main

import (
	"blogalusta/internal/forms"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"net/http"
)

func (app *application) handleShowHomePage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "home.page.gohtml", nil)
}

func (app *application) handleRender(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("content", "title")

	if !form.Valid() {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{
		Flags: htmlFlags,
		Title: form.Get("title"),
	}
	renderer := html.NewRenderer(opts)

	unsafeHTML := markdown.ToHTML([]byte(form.Get("content")), nil, renderer)
	w.Write(app.policy.SanitizeBytes(unsafeHTML))
}
