package main

import (
	"blogalusta/internal/data"
	"blogalusta/internal/forms"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

func (app *application) handleShowHomePage(w http.ResponseWriter, r *http.Request) {
	page := 1
	var err error
	values := r.URL.Query()
	if values.Has("p") {
		page, err = strconv.Atoi(values.Get("p"))
		if err != nil {
			app.clientError(w, http.StatusNotFound)
			return
		}
		if page < 1 {
			app.clientError(w, http.StatusNotFound)
			return
		}
	}

	var filters data.Filters
	filters.Page = page
	filters.PageSize = 10

	articles, metaData, err := app.models.Articles.GetNewestArticles(filters)
	if err == data.ErrRecordNotFound {
		app.clientError(w, http.StatusNotFound)
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}
	pubs, err := app.models.Publications.GetArticlePublications(articles)
	if err != nil {
		app.serverError(w, err)
		return
	}
	writers, err := app.models.Users.GetArticleWriters(articles)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "home.page.gohtml", &templateData{
		Articles:  articles,
		Metadata:  metaData,
		PubMap:    pubs,
		WriterMap: writers,
	})
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
