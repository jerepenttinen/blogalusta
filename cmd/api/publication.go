package main

import (
	"blogalusta/internal/data"
	"blogalusta/internal/forms"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

func (app *application) handleShowPublicationPage(w http.ResponseWriter, r *http.Request) {
	articles, err := app.models.Articles.GetArticlesOfPublication(app.publication(r))
	if err != nil {
		app.serverError(w, err)
		return
	}

	for i := range articles {
		articles[i].Writer, err = app.models.Users.Get(int(articles[i].WriterID))
		if err != nil {
			app.serverError(w, err)
			return
		}
	}

	app.render(w, r, "publication.page.gohtml", &templateData{
		Articles: articles,
	})
}

func (app *application) handleShowArticlePage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "article.page.gohtml", nil)
}

func (app *application) handleShowCreateArticlePage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "new_article.page.gohtml", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) handleCreateArticle(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)
	publication := app.publication(r)

	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("content", "title")

	if !form.Valid() {
		app.render(w, r, "new_article.page.gohtml", &templateData{
			Form: form,
		})
		return
	}

	article, err := app.models.Articles.Publish(user, publication, form.Get("title"), form.Get("content"))
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, publication.GetArticleURL(article), http.StatusSeeOther)
}

func (app *application) handleShowPublicationAboutPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "publication_about.page.gohtml", nil)
}

func (app *application) handleShowPublicationSettingsPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "publication_settings.page.gohtml", nil)
}

func (app *application) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)
	publication := app.publication(r)

	isSubscribed, err := app.models.Publications.UserIsSubscribed(publication, user)
	if isSubscribed || err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// don't allow writers to subscribe
	isWriter, err := app.models.Publications.UserIsWriter(publication, user)
	if isWriter || err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = app.models.Users.SubscribeTo(user, publication)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, publication.GetBaseURL(), http.StatusSeeOther)
}

func (app *application) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)
	publication := app.publication(r)

	isSubscribed, err := app.models.Publications.UserIsSubscribed(publication, user)
	if !isSubscribed || err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = app.models.Users.UnsubscribeFrom(user, publication)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, publication.GetBaseURL(), http.StatusSeeOther)
}

func (app *application) handleInviteWriter(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)
	publication := app.publication(r)

	if user.ID != publication.OwnerID {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("email")
	form.ValidEmail("email")

	if !form.Valid() {
		app.session.Put(r, "flash", "Invalid email")
		http.Redirect(w, r, publication.GetSettingsURL(), http.StatusSeeOther)
		return
	}

	invited, err := app.models.Users.GetByEmail(form.Get("email"))
	if err != nil {
		app.session.Put(r, "flash", "User with this email not found")
		http.Redirect(w, r, publication.GetSettingsURL(), http.StatusSeeOther)
		return
	}

	isWriter, err := app.models.Publications.UserIsWriter(publication, invited)
	if err != nil {
		app.serverError(w, err)
		return
	}

	if isWriter {
		app.session.Put(r, "flash", "This user is already a writer here!")
		http.Redirect(w, r, publication.GetSettingsURL(), http.StatusSeeOther)
		return
	}

	err = app.models.Publications.Invite(publication, invited)
	if err == data.ErrDuplicateRecord {
		app.session.Put(r, "flash", "This user is already invited!")
		http.Redirect(w, r, publication.GetSettingsURL(), http.StatusSeeOther)
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, publication.GetSettingsURL(), http.StatusSeeOther)
}

func (app *application) handleWithdrawInvitation(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)
	publication := app.publication(r)

	if user.ID != publication.OwnerID {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	err = app.models.Publications.Withdraw(publication, id)
	if err == data.ErrRecordNotFound {
		app.clientError(w, http.StatusNotFound)
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, publication.GetSettingsURL(), http.StatusSeeOther)
}
