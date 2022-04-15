package main

import (
	"blogalusta/internal/data"
	"bytes"
	"errors"
	"fmt"
	"github.com/justinas/nosurf"
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
	td.AuthenticatedUser = app.authenticatedUser(r)
	td.Publication = app.publication(r)
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

func (app *application) getArticleSlugAndId(url string) (string, int, error) {
	i := strings.LastIndex(url, "-")
	if i == -1 {
		return "", 0, errors.New("invalid article url")
	}

	slug := url[:i]
	id, _ := strconv.Atoi(url[i+1:])

	return slug, id, nil
}
