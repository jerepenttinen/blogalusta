package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
)

func (app *application) routes() *chi.Mux {
	r := chi.NewRouter()

	r.Use(app.recoverPanic, app.logRequest, secureHeaders)

	dynamic := []func(http.Handler) http.Handler{app.session.Enable, noSurf, app.authenticate}

	r.With(dynamic...).Get("/", app.handleShowHomePage)
	r.With(dynamic...).With(app.requireAuthenticatedUser).Post("/render", app.handleRender)

	r.Get("/img/{imageID:[0-9]+}.jpg", app.handleGetImage)
	r.Get("/img/0.jpg", app.handleGetDefaultImage)

	r.Route("/user", func(r chi.Router) {
		r.Use(dynamic...)
		r.Get("/signup", app.handleShowSignupPage)
		r.Post("/signup", app.handleSignup)
		r.Get("/login", app.handleShowLoginPage)
		r.Post("/login", app.handleLogin)
		r.Route("/", func(r chi.Router) {
			r.Use(app.requireAuthenticatedUser)
			r.Post("/logout", app.handleLogout)
			r.Get("/publication/create", app.handleShowCreatePublicationPage)
			r.Post("/publication/create", app.handleCreatePublication)
			r.Post("/publication/delete", app.handleDeletePublication)
			r.Get("/article", app.handleShowChoosePublicationPage)
			r.Get("/settings", app.handleShowUserSettingsPage)
			r.Post("/image", app.handleChangeUserProfilePicture)
			r.With(app.addProfileToContext).Get("/{profileSlug:[a-z0-9-]+-[0-9]+}", app.handleShowProfilePage)
		})
	})

	r.Route("/{publicationSlug:[a-z-]+}", func(r chi.Router) {
		r.Use(app.addPublicationToContext)
		r.Use(dynamic...)
		r.Get("/", app.handleShowPublicationPage)
		r.Get("/about", app.handleShowPublicationAboutPage)
		r.With(app.addArticleToContext).Get("/{articleSlug:[a-z0-9-]+-[0-9]+}", app.handleShowArticlePage)
		r.Route("/", func(r chi.Router) {
			r.Use(app.requireAuthenticatedUser)
			r.Get("/article", app.handleShowCreateArticlePage)
			r.Post("/article", app.handleCreateArticle)
		})
	})

	FileServer(r, "/static", http.Dir("./ui/static/"))

	return r
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		ctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(ctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
