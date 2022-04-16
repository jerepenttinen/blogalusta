package main

import (
	"blogalusta/internal/data"
	"blogalusta/internal/forms"
	"net/http"
	"net/mail"
	"strconv"
)

func (app *application) handleShowSignupPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "signup.page.gohtml", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) handleSignup(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.errorLog.Print(err)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("name", "email", "password")
	form.MatchesPattern("email", forms.EmailRX)
	form.MinLength("password", 10)
	form.MaxLength("password", 72)

	if !form.Valid() {
		app.render(w, r, "signup.page.gohtml", &templateData{Form: form})
		return
	}

	email, _ := mail.ParseAddress(form.Get("email"))

	err = app.models.Users.Insert(form.Get("name"), email.Address, form.Get("password"))
	if err == data.ErrDuplicateEmail {
		form.Errors.Add("email", "Address is already in use")
		app.render(w, r, "signup.page.gohtml", &templateData{Form: form})
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}

	app.session.Put(r, "flash", "Your signup was successful. Please log in.")
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) handleShowLoginPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "login.page.gohtml", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) handleLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.errorLog.Print(err)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	id, err := app.models.Users.Authenticate(form.Get("email"), form.Get("password"))
	if err != nil {
		app.errorLog.Print(err)
	}
	if err == data.ErrInvalidCredentials {
		form.Errors.Add("generic", "Email or Password is incorrect")
		app.render(w, r, "login.page.gohtml", &templateData{Form: form})
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}

	app.session.Put(r, "userID", id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) handleLogout(w http.ResponseWriter, r *http.Request) {
	app.session.Remove(r, "userID")
	app.session.Put(r, "flash", "You've been logged out")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) handleShowCreatePublicationPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "create_publication.page.gohtml", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) handleCreatePublication(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("name", "description")
	form.MaxLength("name", 24)
	form.MinLength("name", 4)
	form.RestrictedValues("name", "user")

	if !form.Valid() {
		app.render(w, r, "create_publication.page.gohtml", &templateData{Form: form})
		return
	}

	user := app.authenticatedUser(r)
	url, err := app.models.Publications.Insert(user.ID, form.Get("name"), form.Get("description"))
	if err == data.ErrDuplicateUrl {
		form.Errors.Add("name", "Title already in use")
		app.render(w, r, "create_publication.page.gohtml", &templateData{Form: form})
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}

	app.session.Put(r, "flash", "Created a new publication")
	http.Redirect(w, r, "/"+url, http.StatusSeeOther)
}

func (app *application) handleShowMyProfilePage(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)

	publications, err := app.models.Publications.GetUsersPublications(user.ID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "profile.page.gohtml", &templateData{
		Publications: publications,
	})
}

func (app *application) handleDeletePublication(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	user := app.authenticatedUser(r)
	publicationID, err := strconv.Atoi(form.Get("publication-id"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = app.models.Publications.DeleteByID(user.ID, int64(publicationID))
	if err != nil {
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}

	app.session.Put(r, "flash", "Deleted a publication")
	http.Redirect(w, r, "/user/publication", http.StatusSeeOther)
}

func (app *application) handleShowChoosePublicationPage(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)

	publications, err := app.models.Publications.GetUsersPublications(user.ID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "choose_publication.page.gohtml", &templateData{
		Publications: publications,
	})
}
