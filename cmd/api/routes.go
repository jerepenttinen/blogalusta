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
	r.With(dynamic...).With(app.requireAuthenticatedUser).Post("/render", app.handleRender)

	r.Route("/user", func(r chi.Router) {
		r.Use(dynamic...)
		r.Get("/signup", app.handleShowSignupPage)
		r.Post("/signup", app.handleSignup)
		r.Get("/login", app.handleShowLoginPage)
		r.Post("/login", app.handleLogin)
		r.With(app.requireAuthenticatedUser).Post("/logout", app.handleLogout)
		r.With(app.requireAuthenticatedUser).Get("/publication/create", app.handleShowCreatePublicationPage)
		r.With(app.requireAuthenticatedUser).Post("/publication/create", app.handleCreatePublication)
		r.With(app.requireAuthenticatedUser).Get("/publication", app.handleShowMyPublicationsPage)
		r.With(app.requireAuthenticatedUser).Post("/publication/delete", app.handleDeletePublication)
	})

	r.Route("/{publicationSlug:[a-z-]+}", func(r chi.Router) {
		r.Use(app.addPublicationToContext)
		r.Use(dynamic...)
		r.Get("/", app.handleShowPublicationPage)
		r.With(app.requireAuthenticatedUser).Get("/article", app.handleShowCreateArticlePage)
		r.With(app.requireAuthenticatedUser).Post("/article", app.handleCreateArticle)
		r.With(app.addArticleToContext).Get("/{articleSlug:[a-z0-9-]+-[0-9]+}", app.handleShowArticlePage)
	})

	return r
}
