package pairs

import (
	"io"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

func init() {
	http.HandleFunc("/gophers.json", handleGophers)
}

func handleGophers(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)
	// client.Transport = &http.Transport{
	// 	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// }
	res, err := client.Get("https://gopherize.me/gophers/recent/json?" + r.URL.RawQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	if _, err = io.Copy(w, res.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
