package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/matryer/gopherize.me/server"
)

func init() {
	mux := mux.NewRouter()
	mux.PathPrefix("/api/").Handler(server.New())
	mux.Handle("/save", handleSave())
	mux.Handle("/gopher/{gopherhash}", handleGopher())
	mux.Handle("/", server.FileServer("pages/index.html"))
	http.Handle("/", mux)
}
