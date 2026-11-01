package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPagesAvailableFromModuleRoot(t *testing.T) {
	// App Engine builds ./gae but starts the binary from the module root.
	t.Chdir("..")
	response := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Gopherize.me") {
		t.Fatalf("homepage: status %d", response.Code)
	}
	for _, path := range []string{"/static/app.js", "/static/app.css", "/favicon.ico"} {
		response := httptest.NewRecorder()
		http.DefaultServeMux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || response.Body.Len() == 0 {
			t.Fatalf("%s: status %d", path, response.Code)
		}
	}
	for _, page := range []string{"gopher", "grid", "branding"} {
		if _, err := template.ParseFS(pageFiles, "pages/_layout.html", "pages/"+page+".html"); err != nil {
			t.Fatalf("%s: %v", page, err)
		}
	}
}
