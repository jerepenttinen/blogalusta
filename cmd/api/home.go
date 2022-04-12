package main

import "net/http"

func (app *application) handleShowHomePage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "home.page.gohtml", nil)
}
