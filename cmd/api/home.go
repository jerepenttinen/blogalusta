package main

import "net/http"

func (app *application) handleShowHomePage(w http.ResponseWriter, r *http.Request) {
	td := &templateData{}
	user := app.authenticatedUser(r)
	if user != nil {
		td.CanCreatePublication = true
	}
	app.render(w, r, "home.page.gohtml", td)
}
