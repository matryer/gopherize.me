package main

import (
	"html/template"
	"net/http"

	"github.com/matryer/gopherize.me/server"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

func brandingHandler() http.Handler {
	tpl, err := template.ParseFiles("pages/_layout.html", "pages/branding.html")
	if err != nil {
		return server.ErrHandler(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		w.Header().Set("Content-Type", "text/html")
		if err := tpl.ExecuteTemplate(w, "layout", nil); err != nil {
			log.Errorf(ctx, "template execute: %s", err)
			server.ErrHandler(err).ServeHTTP(w, r)
		}
	})
}
