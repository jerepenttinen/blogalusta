package main

import (
	"blogalusta/internal/data"
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/justinas/nosurf"
	"net/http"
)

import (
	"fmt"
)

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("X-Frame-Options", "deny")

		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.infoLog.Printf("%s\t%s %s\t%s", r.RemoteAddr, r.Proto, r.Method, r.URL)

		next.ServeHTTP(w, r)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverError(w, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuthenticatedUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.authenticatedUser(r) == nil {
			http.Redirect(w, r, "/user/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)
	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	})

	return csrfHandler
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exists := app.session.Exists(r, "userID")
		if !exists {
			next.ServeHTTP(w, r)
			return
		}

		user, err := app.models.Users.Get(app.session.GetInt(r, "userID"))
		if err == data.ErrRecordNotFound {
			app.session.Remove(r, "userID")
			next.ServeHTTP(w, r)
			return
		} else if err != nil {
			app.serverError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) addPublicationToContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		publicationSlug := chi.URLParam(r, "publicationSlug")
		publication, err := app.models.Publications.GetBySlug(publicationSlug)
		if err == data.ErrRecordNotFound {
			app.clientError(w, http.StatusNotFound)
			return
		} else if err != nil {
			app.serverError(w, err)
			return
		}

		writers, err := app.models.Users.GetWritersOfPublication(publication)
		if err != nil {
			app.serverError(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), contextKeyPublication, publication)
		ctx = context.WithValue(ctx, contextKeyWriters, writers)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) addArticleToContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url, id, err := app.getSlugAndId(chi.URLParam(r, "articleSlug"))
		if err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}

		article, err := app.models.Articles.Get(id)
		if err == data.ErrRecordNotFound {
			app.clientError(w, http.StatusNotFound)
			return
		} else if err != nil {
			app.serverError(w, err)
			return
		}

		if !article.Matches(url) {
			app.clientError(w, http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyArticle, article)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) addProfileToContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url, id, err := app.getSlugAndId(chi.URLParam(r, "profileSlug"))
		if err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}

		user, err := app.models.Users.Get(id)
		if err == data.ErrRecordNotFound {
			app.clientError(w, http.StatusNotFound)
			return
		} else if err != nil {
			app.serverError(w, err)
			return
		}

		if !user.Matches(url) {
			app.clientError(w, http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyProfile, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
