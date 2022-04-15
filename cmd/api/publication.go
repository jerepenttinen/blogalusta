package main

import (
	"blogalusta/internal/forms"
	"net/http"
)

func (app *application) handleShowPublicationPage(w http.ResponseWriter, r *http.Request) {
	isWriter, err := app.models.Publications.UserIsWriter(app.authenticatedUser(r), app.publication(r))
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "publication.page.gohtml", &templateData{
		IsWriter: isWriter,
	})
}

func (app *application) handleShowArticlePage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "article.page.gohtml", nil)
}

func (app *application) handleShowCreateArticlePage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "create_article.page.gohtml", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) handleCreateArticle(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)
	publication := app.publication(r)

	isWriter, err := app.models.Publications.UserIsWriter(user, publication)
	if err != nil {
		app.serverError(w, err)
		return
	}

	if !isWriter {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	err = r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("content", "title")

	if !form.Valid() {
		app.render(w, r, "create_article.page.gohtml", &templateData{
			Form: form,
		})
		return
	}

	article, err := app.models.Articles.Publish(user, publication, form.Get("title"), form.Get("content"))
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/"+publication.URL+"/"+article.URL, http.StatusSeeOther)
}
