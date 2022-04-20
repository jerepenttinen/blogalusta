package main

import (
	"blogalusta/internal/data"
	"bytes"
	"errors"
	"fmt"
	"github.com/gomarkdown/markdown"
	"github.com/justinas/nosurf"
	"golang.org/x/image/draw"
	"html/template"
	"image"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if value == "true" {
			return true
		} else if value == "false" {
			return false
		} else {
			return fallback
		}
	}
	return fallback
}

func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errorLog.Output(2, trace)

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

func (app *application) addDefaultData(td *templateData, r *http.Request) *templateData {
	if td == nil {
		td = &templateData{}
	}
	td.CSRFToken = nosurf.Token(r)
	td.CurrentYear = time.Now().Year()
	td.Flash = app.session.PopString(r, "flash")
	td.AuthenticatedUser = app.authenticatedUser(r)
	td.Publication = app.publication(r)
	td.Article = app.article(r)
	if td.Article != nil {
		td.HTML = template.HTML(app.markdownToHTML(td.Article.Content))
		td.Article.Writer, _ = app.models.Users.Get(int(td.Article.WriterID))
	}
	td.ProfileUser = app.profileUser(r)
	td.Writers = app.writers(r)
	td.IsSubscribed, _ = app.models.Publications.UserIsSubscribed(td.Publication, td.AuthenticatedUser)
	td.InvitedWriters = app.pending(r)
	return td
}

func (app *application) render(w http.ResponseWriter, r *http.Request, name string, td *templateData) {
	ts, ok := app.templateCache[name]
	if !ok {
		app.serverError(w, fmt.Errorf("the template %s does not exist", name))
		return
	}

	buf := new(bytes.Buffer)

	err := ts.Execute(buf, app.addDefaultData(td, r))
	if err != nil {
		app.serverError(w, err)
		return
	}

	buf.WriteTo(w)
}

func (app *application) authenticatedUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(contextKeyUser).(*data.User)
	if !ok {
		return nil
	}
	return user
}

func (app *application) publication(r *http.Request) *data.Publication {
	publication, ok := r.Context().Value(contextKeyPublication).(*data.Publication)
	if !ok {
		return nil
	}
	return publication
}

func (app *application) getSlugAndId(url string) (string, int, error) {
	i := strings.LastIndex(url, "-")
	if i == -1 {
		return "", 0, errors.New("invalid slug")
	}

	slug := url[:i]
	id, _ := strconv.Atoi(url[i+1:])

	return slug, id, nil
}

func (app *application) markdownToHTML(md string) []byte {
	normalized := markdown.NormalizeNewlines([]byte(md))
	unsafeHTML := markdown.ToHTML(normalized, nil, app.markdown.renderer)
	return app.markdown.policy.SanitizeBytes(unsafeHTML)
}

func (app *application) article(r *http.Request) *data.Article {
	article, ok := r.Context().Value(contextKeyArticle).(*data.Article)
	if !ok {
		return nil
	}
	return article
}

func (app *application) profileUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(contextKeyProfile).(*data.User)
	if !ok {
		return nil
	}
	return user
}

func (app *application) writers(r *http.Request) []*data.User {
	writers, ok := r.Context().Value(contextKeyWriters).([]*data.User)
	if !ok {
		return nil
	}
	return writers
}

func (app *application) pending(r *http.Request) []*data.User {
	pending, ok := r.Context().Value(contextKeyPending).([]*data.User)
	if !ok {
		return nil
	}
	return pending
}

func (app *application) comment(r *http.Request) *data.Comment {
	comment, ok := r.Context().Value(contextKeyComment).(*data.Comment)
	if !ok {
		return nil
	}
	return comment
}

func cropImage(img image.Image, crop image.Rectangle) (image.Image, error) {
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	simg, ok := img.(subImager)
	if !ok {
		return nil, errors.New("image dose not support cropping")
	}

	return simg.SubImage(crop), nil
}

func cropCenterResize(img image.Image, sideLength int) (image.Image, error) {
	rect := img.Bounds()

	if rect.Max.X < rect.Max.Y {
		y0 := (rect.Dy() - rect.Dx()) / 2
		y1 := rect.Dx() + y0

		rect = image.Rect(rect.Min.X, y0, rect.Max.X, y1)
	} else if rect.Max.X > rect.Max.Y {
		x0 := (rect.Dx() - rect.Dy()) / 2
		x1 := rect.Dy() + x0

		rect = image.Rect(x0, rect.Min.Y, x1, rect.Max.Y)
	}
	img, err := cropImage(img, rect)
	if err != nil {
		return nil, err
	}
	dst := image.NewRGBA(image.Rect(0, 0, sideLength, sideLength))
	draw.BiLinear.Scale(dst, dst.Rect, img, img.Bounds(), draw.Over, nil)

	return dst, nil
}

func (app *application) likeArticle(w http.ResponseWriter, user *data.User, article *data.Article) error {
	hasLiked, err := app.models.Articles.UserHasLiked(article, user)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return err
	}
	if hasLiked {
		app.clientError(w, http.StatusBadRequest)
		return errors.New("already liked")
	}

	err = app.models.Users.LikeArticle(user, article)
	if err != nil {
		app.serverError(w, err)
		return err
	}

	return nil
}

func (app *application) unlikeArticle(w http.ResponseWriter, user *data.User, article *data.Article) error {
	hasLiked, err := app.models.Articles.UserHasLiked(article, user)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return err
	}

	if !hasLiked {
		app.clientError(w, http.StatusBadRequest)
		return errors.New("not liked")
	}

	err = app.models.Users.UnlikeArticle(user, article)
	if err != nil {
		app.serverError(w, err)
		return err
	}
	return nil
}
