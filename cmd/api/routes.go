package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (app *application) routes() *chi.Mux {
	r := chi.NewRouter()

	r.Use(app.recoverPanic, app.logRequest, secureHeaders)

	dynamic := []func(http.Handler) http.Handler{app.session.Enable, noSurf, app.authenticate}

	r.With(dynamic...).Get("/", app.handleShowHomePage)

	r.Route("/user", func(r chi.Router) {
		r.Use(dynamic...)
		r.Get("/signup", app.handleShowSignupPage)
		r.Post("/signup", app.handleSignup)
		r.Get("/login", app.handleShowLoginPage)
		r.Post("/login", app.handleLogin)
		r.Post("/logout", app.handleLogout)
		r.Get("/publication/create", app.handleShowCreatePublicationPage)
		r.Post("/publication/create", app.handleCreatePublication)
		r.Get("/publication", app.handleShowMyPublicationsPage)
	})

	r.Route("/{publicationSlug:[a-z-]+}", func(r chi.Router) {
		r.Use(dynamic...)
		r.Get("/", app.handleShowPublicationPage)
		r.Get("/{articleSlug:[a-z0-9-]+}", app.handleShowArticlePage)
	})

	return r
}
