package main

import (
	"embed"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/matryer/gopherize.me/server"
	"github.com/rs/cors"
	"google.golang.org/appengine/v2"
)

//go:embed pages/*.html static
var pageFiles embed.FS

func init() {
	mux := mux.NewRouter()
	mux.PathPrefix("/static/").Handler(http.FileServer(http.FS(pageFiles)))
	for _, path := range []string{"/favicon.ico", "/favicon"} {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, pageFiles, "static/favicon.ico")
		})
	}
	mux.Handle("/gopher/{gopherhash}/json", handleGopherAPI())
	mux.Handle("/gophers/recent/json", handleRecentGophers())
	mux.PathPrefix("/api/").Handler(server.New())
	mux.Handle("/branding", brandingHandler())
	mux.Handle("/save", handleSave())
	mux.Handle("/gopher/{gopherhash}", handleGopher())
	mux.Handle("/gophers/count", handleGophersCount())
	mux.Handle("/grid", handleGrid())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, pageFiles, "pages/index.html")
	})
	http.Handle("/", cors.Default().Handler(mux))
}

func main() {
	appengine.Main()
}
