package main

import (
	"embed"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/matryer/gopherize.me/server"
	"github.com/rs/cors"
	"google.golang.org/appengine/v2"
)

//go:embed pages
var pageFiles embed.FS

func init() {
	mux := mux.NewRouter()
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
