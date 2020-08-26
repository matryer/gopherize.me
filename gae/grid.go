package main

import (
	"html/template"
	"net/http"

	"github.com/matryer/gopherize.me/server"
	"github.com/pkg/errors"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

func handleGrid() http.Handler {
	tpl, err := template.ParseFiles("pages/_layout.html", "pages/grid.html")
	if err != nil {
		return server.ErrHandler(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		pageInfo := struct {
			PageURL     string
			CacheBuster string
		}{
			PageURL:     "https://gopherize.me/grid",
			CacheBuster: appengine.VersionID(ctx),
		}
		if err := tpl.ExecuteTemplate(w, "layout", pageInfo); err != nil {
			err = errors.Wrap(err, "rendering template")
			log.Errorf(ctx, "%s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
