package main

import (
	"blogalusta/internal/data"
	"blogalusta/internal/forms"
	"bytes"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
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
		if form.Errors.Has("email") {
			app.session.Put(r, "flash", form.Errors.Get("email"))
		} else if form.Errors.Has("password") {
			app.session.Put(r, "flash", form.Errors.Get("password"))
		}
		app.render(w, r, "signup.page.gohtml", &templateData{Form: form})
		return
	}

	email, _ := mail.ParseAddress(form.Get("email"))

	err = app.models.Users.Insert(form.Get("name"), email.Address, form.Get("password"))
	if err == data.ErrDuplicateRecord {
		app.session.Put(r, "flash", "Email address already in use")
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
		app.session.Put(r, "flash", "Email or Password is incorrect")
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
		app.session.Put(r, "flash", form.Errors.Get("name"))
		app.render(w, r, "create_publication.page.gohtml", &templateData{Form: form})
		return
	}

	user := app.authenticatedUser(r)
	url, err := app.models.Publications.Insert(user.ID, form.Get("name"), form.Get("description"))
	if err == data.ErrDuplicateRecord {
		app.session.Put(r, "flash", "Publication name already in use")
		app.render(w, r, "create_publication.page.gohtml", &templateData{Form: form})
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}

	app.session.Put(r, "flash", "Created a new publication")
	http.Redirect(w, r, "/"+url, http.StatusSeeOther)
}

func (app *application) handleShowProfilePage(w http.ResponseWriter, r *http.Request) {
	user := app.profileUser(r)
	if user == nil {
		app.serverError(w, errors.New("profile user not found"))
		return
	}

	publications, err := app.models.Publications.GetUsersPublications(user.ID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "profile.page.gohtml", &templateData{
		ProfilePublications: publications,
	})
}

func (app *application) handleDeletePublication(w http.ResponseWriter, r *http.Request) {
	publication := app.publication(r)
	err := app.models.Publications.Delete(publication)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.session.Put(r, "flash", fmt.Sprintf("%s deleted!", publication.Name))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) handleShowChoosePublicationPage(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)

	publications, err := app.models.Publications.GetUsersPublications(user.ID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "choose_publication.page.gohtml", &templateData{
		ProfilePublications: publications,
	})
}

func (app *application) handleShowUserSettingsPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "user_settings.page.gohtml", nil)
}

func (app *application) handleChangeUserProfilePicture(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(int64(app.config.avatar.maxSize))
	if err != nil {
		app.clientError(w, http.StatusRequestEntityTooLarge)
		app.errorLog.Print(err)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		app.errorLog.Print(err)
		return
	}
	defer file.Close()

	buf := make([]byte, 512)
	_, err = file.Read(buf)
	if err != nil {
		app.serverError(w, err)
		return
	}

	filetype := http.DetectContentType(buf)
	if filetype != "image/jpeg" && filetype != "image/png" {
		app.clientError(w, http.StatusUnsupportedMediaType)
		app.errorLog.Print(filetype)
		return
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		app.serverError(w, err)
		return
	}

	img, _, err := image.Decode(file)
	if err != nil {
		app.serverError(w, err)
		return
	}

	img, err = cropCenterResize(img, app.config.avatar.sideLength)
	if err != nil {
		app.serverError(w, err)
		return
	}

	buffer := new(bytes.Buffer)
	err = jpeg.Encode(buffer, img, nil)
	if err != nil {
		app.serverError(w, err)
		return
	}

	id, err := app.models.Images.Insert(buffer.Bytes())
	if err != nil {
		app.serverError(w, err)
		return
	}

	err = app.models.Users.ChangeProfilePicture(app.authenticatedUser(r), id)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	http.Redirect(w, r, "/user/settings", http.StatusSeeOther)
}

func (app *application) handleShowUserInvitationsPage(w http.ResponseWriter, r *http.Request) {
	invitations, err := app.models.Users.Invitations(app.authenticatedUser(r))
	if err == data.ErrRecordNotFound {
		// do nothing
	} else if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "user_invitations.page.gohtml", &templateData{
		InvitingPublications: invitations,
	})
}

func (app *application) handleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	err = app.models.Users.AcceptInvitation(app.authenticatedUser(r), id)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.session.Put(r, "flash", "Invalid invitation")
			http.Redirect(w, r, "/user/invitations", http.StatusSeeOther)
			return
		case data.ErrDuplicateRecord:
			app.session.Put(r, "flash", "You're already writer in that publication")
			http.Redirect(w, r, "/user/invitations", http.StatusSeeOther)
			return
		}

		app.serverError(w, err)
		return
	}

	app.session.Put(r, "flash", "Accepted invite")
	http.Redirect(w, r, "/user/invitations", http.StatusSeeOther)
}

func (app *application) handleDeclineInvitation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	err = app.models.Users.DeclineInvitation(app.authenticatedUser(r), id)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.session.Put(r, "flash", "Invalid invitation")
			http.Redirect(w, r, "/user/invitations", http.StatusSeeOther)
			return
		}

		app.serverError(w, err)
		return
	}

	app.session.Put(r, "flash", "Declined invite")
	http.Redirect(w, r, "/user/invitations", http.StatusSeeOther)
}

func (app *application) handleLeavePublication(w http.ResponseWriter, r *http.Request) {
	publicationID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	err = app.models.Users.Leave(app.authenticatedUser(r), publicationID)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.session.Put(r, "flash", "Invalid publication")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		app.serverError(w, err)
		return
	}

	app.session.Put(r, "flash", "Left the publication")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
