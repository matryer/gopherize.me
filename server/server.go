package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/fasterness/cors"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
)

// New makes a new server.
func New() http.Handler {
	return cors.New(&server{})
}

type server struct{}

func (s server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/artwork") {
		s.artworkHandler(w, r)
		return
	}
	if r.URL.Path == "/api/render" || r.URL.Path == "/api/render.png" {
		s.renderHandler(w, r)
		return
	}
	http.NotFound(w, r)
}

func (s server) respond(ctx context.Context, w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Errorf(ctx, "encode response: %s", err)
	}
}
func (s server) responderr(ctx context.Context, w http.ResponseWriter, r *http.Request, status int, err error) {
	w.WriteHeader(status)
	var data struct {
		Error string `json:"error"`
	}
	if err != nil {
		data.Error = err.Error()
	} else {
		data.Error = "Something went wrong"
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Errorf(ctx, "encode response: %s", err)
	}
}

type FileServer string

func (f FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, string(f))
}

func ErrHandler(err error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, err.Error())
	})
}
