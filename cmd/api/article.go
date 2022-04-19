package main

import (
	"net/http"
)

func (app *application) handleShowArticlePage(w http.ResponseWriter, r *http.Request) {
	td := &templateData{}
	var err error
	article := app.article(r)
	user := app.authenticatedUser(r)

	td.Like, err = app.models.Articles.Likes(article, user)
	if err != nil {
		app.serverError(w, err)
		return
	}

	td.Comments, err = app.models.Comments.Retrieve(article)
	if err != nil {
		app.serverError(w, err)
		return
	}

	td.UserMap, err = app.models.Comments.Commenters(td.Comments, nil)
	if err != nil {
		app.serverError(w, err)
		return
	}

	td.LikeMap, err = app.models.Comments.LikesMany(td.Comments, user)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "article.page.gohtml", td)
}

func (app *application) handleLikeArticle(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)
	publication := app.publication(r)
	article := app.article(r)

	err := app.likeArticle(w, user, article)
	if err != nil {
		return
	}

	http.Redirect(w, r, publication.GetArticleURL(article), http.StatusSeeOther)
}

func (app *application) handleUnlikeArticle(w http.ResponseWriter, r *http.Request) {
	user := app.authenticatedUser(r)
	publication := app.publication(r)
	article := app.article(r)

	err := app.unlikeArticle(w, user, article)
	if err != nil {
		return
	}

	http.Redirect(w, r, publication.GetArticleURL(article), http.StatusSeeOther)
}

func (app *application) handleCreateComment(w http.ResponseWriter, r *http.Request) {
	article := app.article(r)
	publication := app.publication(r)

	http.Redirect(w, r, publication.GetArticleURL(article), http.StatusSeeOther)
}

func (app *application) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	article := app.article(r)
	publication := app.publication(r)

	http.Redirect(w, r, publication.GetArticleURL(article), http.StatusSeeOther)
}

func (app *application) handleLikeComment(w http.ResponseWriter, r *http.Request) {
	article := app.article(r)
	publication := app.publication(r)
	user := app.authenticatedUser(r)
	comment := app.comment(r)

	hasLiked, err := app.models.Comments.UserHasLiked(comment, user)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	if hasLiked {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = app.models.Users.LikeComment(user, comment)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, publication.GetArticleURL(article)+"#comments", http.StatusSeeOther)
}

func (app *application) handleUnlikeComment(w http.ResponseWriter, r *http.Request) {
	article := app.article(r)
	publication := app.publication(r)
	user := app.authenticatedUser(r)
	comment := app.comment(r)

	hasLiked, err := app.models.Comments.UserHasLiked(comment, user)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if !hasLiked {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = app.models.Users.UnlikeComment(user, comment)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, publication.GetArticleURL(article)+"#comments", http.StatusSeeOther)
}
