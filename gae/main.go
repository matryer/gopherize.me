package main

import (
	"net/http"

	"github.com/gopherize/app/server"
)

func init() {
	http.Handle("/api/", server.New())
	http.Handle("/", server.FileServer("pages/index.html"))
}
