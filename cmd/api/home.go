package main

import (
	"blogalusta/internal/data"
	"blogalusta/internal/forms"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

func (app *application) handleShowHomePage(w http.ResponseWriter, r *http.Request) {
	page := 1
	var err error
	values := r.URL.Query()
	if values.Has("p") {
		page, err = strconv.Atoi(values.Get("p"))
		if err != nil {
			app.clientError(w, http.StatusNotFound)
			return
		}
		if page < 1 {
			app.clientError(w, http.StatusNotFound)
			return
		}
	}

	var filters data.Filters
	filters.Page = page
	filters.PageSize = 10

	articles, metaData, err := app.models.Articles.GetNewestArticles(filters)
	if err == data.ErrRecordNotFound {
		app.clientError(w, http.StatusNotFound)
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}
	pubs, err := app.models.Publications.GetArticlePublications(articles)
	if err != nil {
		app.serverError(w, err)
		return
	}
	writers, err := app.models.Users.GetArticleWriters(articles)
	if err != nil {
		app.serverError(w, err)
		return
	}

	user := app.authenticatedUser(r)
	likeMap, err := app.models.Articles.LikesMany(articles, user)

	app.render(w, r, "home.page.gohtml", &templateData{
		Articles: articles,
		Metadata: metaData,
		PubMap:   pubs,
		UserMap:  writers,
		LikeMap:  likeMap,
	})
}

func (app *application) handleRender(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("content", "title")

	if !form.Valid() {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	w.Write(app.markdownToHTML(form.Get("content")))
}

func (app *application) handleGetImage(w http.ResponseWriter, r *http.Request) {
	imageStr := chi.URLParam(r, "imageID")
	if imageStr == "" {
		app.clientError(w, http.StatusNotFound)
		return
	}

	imageID, err := strconv.Atoi(imageStr)
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	image, err := app.models.Images.Get(imageID)
	if err == data.ErrRecordNotFound {
		app.clientError(w, http.StatusNotFound)
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}

	w.Write(image)
}

func (app *application) handleGetDefaultImage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/static/img/default.png", http.StatusSeeOther)
}

func (app *application) handleLikeArticleHome(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)

	articleID, err := strconv.Atoi(chi.URLParam(r, "articleID"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	err = r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("page")

	if !form.Valid() {
		if form.Errors.Has("page") {
			app.session.Put(r, "flash", form.Errors.Get("page"))
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	page, err := strconv.Atoi(form.Get("page"))
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	article, err := app.models.Articles.Get(articleID)
	if err == data.ErrRecordNotFound {
		app.clientError(w, http.StatusNotFound)
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}

	err = app.likeArticle(w, user, article)
	if err != nil {
		return
	}

	r.URL.Path = "/"
	if page != 1 {
		values := r.URL.Query()
		values.Add("p", strconv.Itoa(page))
		r.URL.RawQuery = values.Encode()
	}

	http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
}

func (app *application) handleUnlikeArticleHome(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)

	articleID, err := strconv.Atoi(chi.URLParam(r, "articleID"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("page")

	if !form.Valid() {
		if form.Errors.Has("page") {
			app.session.Put(r, "flash", form.Errors.Get("page"))
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	page, err := strconv.Atoi(form.Get("page"))
	if err != nil {
		app.clientError(w, http.StatusNotFound)
		return
	}

	article, err := app.models.Articles.Get(articleID)
	if err == data.ErrRecordNotFound {
		app.clientError(w, http.StatusNotFound)
		return
	} else if err != nil {
		app.serverError(w, err)
		return
	}

	err = app.unlikeArticle(w, user, article)
	if err != nil {
		return
	}

	r.URL.Path = "/"
	if page != 1 {
		values := r.URL.Query()
		values.Add("p", strconv.Itoa(page))
		r.URL.RawQuery = values.Encode()
	}

	http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
}
