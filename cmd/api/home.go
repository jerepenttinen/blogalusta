package main

import (
	"blogalusta/internal/forms"
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

	w.Write(app.markdownToHTML(form.Get("content")))
}
