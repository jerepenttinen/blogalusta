package main

import (
	"blogalusta/internal/data"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (app *application) handleShowPublicationPage(w http.ResponseWriter, r *http.Request) {
	publicationSlug := chi.URLParam(r, "publicationSlug")
	publication, err := app.models.Publications.GetBySlug(publicationSlug)
	if err == data.ErrRecordNotFound {
		app.clientError(w, http.StatusNotFound)
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "publication.page.gohtml", &templateData{
		Publication: publication,
	})
}

func (app *application) handleShowArticlePage(w http.ResponseWriter, r *http.Request) {

}
