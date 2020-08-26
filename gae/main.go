package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/matryer/gopherize.me/server"
	"github.com/rs/cors"
	"google.golang.org/appengine"
)

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
	mux.Handle("/", server.FileServer("pages/index.html"))
	http.Handle("/", cors.Default().Handler(mux))
}

func main() {
	appengine.Main()
}
