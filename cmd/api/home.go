package main

import (
	"blogalusta/internal/data"
	"blogalusta/internal/forms"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
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

func (app *application) handleGetImage(w http.ResponseWriter, r *http.Request) {
	imageStr := chi.URLParam(r, "imageID")
	if imageStr == "" {
		app.clientError(w, http.StatusNotFound)
		return
	}

	imageID, err := strconv.Atoi(imageStr)
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	image, err := app.models.Images.Get(imageID)
	if err == data.ErrRecordNotFound {
		app.clientError(w, http.StatusNotFound)
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}

	w.Write(image)
}

func (app *application) handleGetDefaultImage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/static/img/default.png", http.StatusSeeOther)
}
