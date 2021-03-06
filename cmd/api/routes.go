package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
)

func (app *application) routes() *chi.Mux {
	r := chi.NewRouter()

	r.Use(app.recoverPanic, app.logRequest, app.secureHeaders)

	dynamic := []func(http.Handler) http.Handler{app.session.Enable, noSurf, app.authenticate}

	r.With(dynamic...).Get("/", app.handleShowHomePage)

	r.With(dynamic...).With(app.requireAuthenticatedUser).Post("/{articleID:[0-9]+}/like", app.handleLikeArticleHome)
	r.With(dynamic...).With(app.requireAuthenticatedUser).Post("/{articleID:[0-9]+}/unlike", app.handleUnlikeArticleHome)

	r.Get("/img/{imageID:[0-9]+}.jpg", app.handleGetImage)
	r.Get("/img/0.jpg", app.handleGetDefaultImage)

	r.Route("/user", func(r chi.Router) {
		r.Use(dynamic...)
		r.Get("/signup", app.handleShowSignupPage)
		r.Post("/signup", app.handleSignup)
		r.Get("/login", app.handleShowLoginPage)
		r.Post("/login", app.handleLogin)
		r.With(app.addProfileToContext).Get("/{profileSlug:[a-z0-9-]+-[0-9]+}", app.handleShowProfilePage)

		r.Route("/", func(r chi.Router) {
			r.Use(app.requireAuthenticatedUser)
			r.Post("/logout", app.handleLogout)
			r.Get("/publication/create", app.handleShowCreatePublicationPage)
			r.Post("/publication/create", app.handleCreatePublication)
			r.Post("/publication/{id:[0-9]+}/leave", app.handleLeavePublication)
			r.Get("/publication/list", app.handleShowPublicationListPage)
			r.Get("/article", app.handleShowChoosePublicationPage)
			r.Get("/invitations", app.handleShowUserInvitationsPage)
			r.Post("/invitations/{id:[0-9]+}/accept", app.handleAcceptInvitation)
			r.Post("/invitations/{id:[0-9]+}/decline", app.handleDeclineInvitation)

			r.Route("/settings", func(r chi.Router) {
				r.Get("/", app.handleShowUserSettingsPage)
				r.Post("/picture", app.handleChangeUserProfilePicture)
				r.Post("/name", app.handleChangeUserName)
				r.Post("/password", app.handleChangeUserPassword)
			})
		})
	})

	r.Route("/{publicationSlug:[a-z-]+}", func(r chi.Router) {
		r.Use(app.addPublicationToContext)
		r.Use(dynamic...)
		r.Get("/", app.handleShowPublicationPage)
		r.Get("/about", app.handleShowPublicationAboutPage)

		r.Route("/", func(r chi.Router) {
			r.Use(app.requireAuthenticatedUser)
			r.Post("/subscribe", app.handleSubscribe)
			r.Post("/unsubscribe", app.handleUnsubscribe)
			r.Post("/{articleID:[0-9]+}/like", app.handleLikeArticlePublication)
			r.Post("/{articleID:[0-9]+}/unlike", app.handleUnlikeArticlePublication)

			r.Route("/", func(r chi.Router) {
				r.Use(app.requireUserIsWriter)
				r.Get("/article", app.handleShowCreateArticlePage)
				r.Post("/article", app.handleCreateArticle)

				r.Route("/", func(r chi.Router) {
					r.Use(app.requireUserIsOwner)
					r.Get("/settings", app.handleShowPublicationSettingsPage)
					r.Post("/invite", app.handleInviteWriter)
					r.Post("/{userID:[0-9]+}/withdraw", app.handleWithdrawInvitation)
					r.Post("/{userID:[0-9]+}/kick", app.handleKickWriter)
					r.Post("/delete", app.handleDeletePublication)
				})
			})
		})
		r.Route("/{articleSlug:[a-z0-9-]+-[0-9]+}", func(r chi.Router) {
			r.Use(app.addArticleToContext)
			r.Get("/", app.handleShowArticlePage)
			r.Route("/", func(r chi.Router) {
				r.Use(app.requireAuthenticatedUser)
				r.Post("/like", app.handleLikeArticle)
				r.Post("/unlike", app.handleUnlikeArticle)
				r.Post("/comment", app.handleCreateComment)
				r.Route("/{commentID:[0-9]+}", func(r chi.Router) {
					r.Use(app.addCommentToContext)
					r.Post("/delete", app.handleDeleteComment)
					r.Post("/like", app.handleLikeComment)
					r.Post("/unlike", app.handleUnlikeComment)
				})
			})
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
