package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	humanize "github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/matryer/gopherize.me/server"
	"github.com/pkg/errors"
	"google.golang.org/appengine"
	"google.golang.org/appengine/blobstore"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/file"
	"google.golang.org/appengine/image"
	"google.golang.org/appengine/log"
)

const (
	gopherKind = "Gopher"
)

type Gopher struct {
	ID           string    `datastore:"-" json:"id,omitempty"`
	Images       []string  `datastore:",noindex" json:"images"`
	OriginalURL  string    `datastore:",noindex" json:"original_url"`
	URL          string    `datastore:",noindex" json:"url"`
	ThumbnailURL string    `datastore:",noindex" json:"thumbnail_url"`
	CTime        time.Time `json:"ctime"`
}

var ageMagnitudes = []humanize.RelTimeMagnitude{
	{time.Second, "born just now", time.Second},
	{2 * time.Second, "1 second %s", 1},
	{time.Minute, "%d seconds %s", time.Second},
	{2 * time.Minute, "1 minute %s", 1},
	{time.Hour, "%d minutes %s", time.Minute},
	{2 * time.Hour, "1 hour %s", 1},
	{humanize.Day, "%d hours %s", time.Hour},
	{2 * humanize.Day, "1 day %s", 1},
	{humanize.Week, "%d days %s", humanize.Day},
	{2 * humanize.Week, "1 week %s", 1},
	{humanize.Month, "%d weeks %s", humanize.Week},
	{2 * humanize.Month, "1 month %s", 1},
	{humanize.Year, "%d months %s", humanize.Month},
	{18 * humanize.Month, "1 year %s", 1},
	{2 * humanize.Year, "2 years %s", 1},
	{humanize.LongTime, "%d years %s", humanize.Year},
	{math.MaxInt64, "a long while %s", 1},
}

func (g Gopher) Age() string {
	return humanize.CustomRelTime(g.CTime, time.Now(), "old", "", ageMagnitudes)
}

func handleSave() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		imageList := r.URL.Query().Get("images")
		images := strings.Split(imageList, "|")
		if len(images) == 0 {
			http.Error(w, "missing images", http.StatusBadRequest)
			return
		}
		imageList = strings.Join(images, "|")
		imagesHash := hash(imageList)
		gopherKey := datastore.NewKey(ctx, gopherKind, imagesHash, 0, nil)
		var gopher Gopher
		err := datastore.Get(ctx, gopherKey, &gopher)
		if err != datastore.ErrNoSuchEntity && err != nil {
			err = errors.Wrap(err, "read Gopher")
			log.Errorf(ctx, "%s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err == nil {
			log.Debugf(ctx, "already rendered - skipping: %s", images)
		}
		if err == datastore.ErrNoSuchEntity {
			// gopher doesn't exist - create it
			log.Debugf(ctx, "rendering: %s", images)
			var buf bytes.Buffer
			if err := server.Render(ctx, &buf, images); err != nil {
				err = errors.Wrap(err, "rendering")
				log.Errorf(ctx, "%s", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			bucket, err := file.DefaultBucketName(ctx)
			if err != nil {
				err = errors.Wrap(err, "DefaultBucketName")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			client, err := storage.NewClient(ctx)
			if err != nil {
				err = errors.Wrap(err, "storage.NewClient")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			objpath := "gophers/" + imagesHash + ".png"
			object := client.Bucket(bucket).Object(objpath)
			objW := object.NewWriter(ctx)
			objW.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
			objW.CacheControl = "public, max-age=31536000"
			if err := server.Render(ctx, objW, images); err != nil {
				objW.Close()
				err = errors.Wrap(err, "Render")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := objW.Close(); err != nil {
				err = errors.Wrap(err, "Close writer")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			blobkey, err := blobstore.BlobKeyForFile(ctx, fmt.Sprintf("/gs/%s/%s", bucket, objpath))
			if err != nil {
				err = errors.Wrap(err, "BlobKeyForFile")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			absURL, err := image.ServingURL(ctx, blobkey, &image.ServingURLOptions{Secure: true})
			if err != nil {
				err = errors.Wrap(err, "ServingURL (abs)")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			absURLStr := absURL.String()
			thumbURL, err := image.ServingURL(ctx, blobkey, &image.ServingURLOptions{Secure: true, Size: 70})
			if err != nil {
				err = errors.Wrap(err, "ServingURL (thumb)")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			thumbURLStr := thumbURL.String()

			gopher = Gopher{
				Images:       images,
				CTime:        time.Now(),
				URL:          absURLStr,
				ThumbnailURL: thumbURLStr,
				OriginalURL:  fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, objpath),
			}
			gopherKey, err = datastore.Put(ctx, gopherKey, &gopher)
			if err != nil {
				err = errors.Wrap(err, "save Gopher")
				log.Errorf(ctx, "%s", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		http.Redirect(w, r, "/gopher/"+gopherKey.StringID(), 308) // StatusPermanentRedirect
	})
}

func handleGopher() http.Handler {
	tpl, err := template.ParseFiles("pages/_layout.html", "pages/gopher.html")
	if err != nil {
		return server.ErrHandler(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)

		gopherHash := mux.Vars(r)["gopherhash"]

		if appengine.IsDevAppServer() {
			pageInfo := struct {
				PageURL     string
				Gopher      Gopher
				GopherHash  string
				CacheBuster string
			}{
				PageURL:     "https://gopherize.me/gopher/" + gopherHash,
				GopherHash:  gopherHash,
				CacheBuster: time.Now().String(),
				Gopher: Gopher{
					CTime:        time.Now(),
					URL:          "https://lh3.googleusercontent.com/6VExrE4MS9Z7FbK-Os9pYdnoXl0etzyCganMXyHv3Rd8eqdiDwmLxP8FdaRD07zUweE7yFq1jaWl9Em1Jssrxbs",
					ThumbnailURL: "https://lh3.googleusercontent.com/6VExrE4MS9Z7FbK-Os9pYdnoXl0etzyCganMXyHv3Rd8eqdiDwmLxP8FdaRD07zUweE7yFq1jaWl9Em1Jssrxbs=s70",
					OriginalURL:  "https://storage.googleapis.com/gopherizeme.appspot.com/gophers/b15efa9350d901705427ce0df2dbc3861d458a76.png",
				},
			}
			w.Header().Set("Content-Type", "text/html")
			if err := tpl.ExecuteTemplate(w, "layout", pageInfo); err != nil {
				log.Errorf(ctx, "template execute: %s", err)
				server.ErrHandler(err).ServeHTTP(w, r)
			}
			return
		}

		var gopher Gopher
		gopherKey := datastore.NewKey(ctx, gopherKind, gopherHash, 0, nil)
		err := datastore.Get(ctx, gopherKey, &gopher)
		if err == datastore.ErrNoSuchEntity {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			err = errors.Wrap(err, "load gopher")
			log.Errorf(ctx, "%s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		pageInfo := struct {
			PageURL     string
			Gopher      Gopher
			GopherHash  string
			CacheBuster string
		}{
			PageURL:     "https://gopherize.me/gopher/" + gopherHash,
			Gopher:      gopher,
			GopherHash:  gopherHash,
			CacheBuster: appengine.VersionID(ctx),
		}
		w.Header().Set("Content-Type", "text/html")
		if err := tpl.ExecuteTemplate(w, "layout", pageInfo); err != nil {
			log.Errorf(ctx, "template execute: %s", err)
			server.ErrHandler(err).ServeHTTP(w, r)
		}
	})
}

func handleGopherAPI() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		gopherHash := mux.Vars(r)["gopherhash"]
		var gopher Gopher
		gopherKey := datastore.NewKey(ctx, gopherKind, gopherHash, 0, nil)
		err := datastore.Get(ctx, gopherKey, &gopher)
		if err == datastore.ErrNoSuchEntity {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			err = errors.Wrap(err, "load gopher")
			log.Errorf(ctx, "%s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		imageList := strings.Join(gopher.Images, "|")
		gopher.ID = hash(imageList)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(gopher); err != nil {
			err = errors.Wrap(err, "encode gopher")
			log.Errorf(ctx, "%s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func handleRecentGophers() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var response struct {
			Gophers []Gopher `json:"gophers"`
		}
		limitStr := r.URL.Query().Get("limit")
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			limit = 100
		}
		if limit > 1000 {
			limit = 1000
		}
		ctx := appengine.NewContext(r)
		keys, err := datastore.NewQuery(gopherKind).Limit(limit).Order("-CTime").GetAll(ctx, &response.Gophers)
		if err != nil {
			err = errors.Wrap(err, "load gophers")
			log.Errorf(ctx, "%s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for i := range keys {
			imageList := strings.Join(response.Gophers[i].Images, "|")
			response.Gophers[i].ID = hash(imageList)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			err = errors.Wrap(err, "encode gopher")
			log.Errorf(ctx, "%s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func handleGophersCount() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		n, err := datastore.NewQuery(gopherKind).Count(ctx)
		if err != nil {
			err = errors.Wrap(err, "counting gophers is hard")
			log.Errorf(ctx, "%s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var response struct {
			N int `json:"gophers_count"`
		}
		response.N = n
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			err = errors.Wrap(err, "encode gopher")
			log.Errorf(ctx, "%s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func hash(s string) string {
	hash := sha1.New()
	hash.Write([]byte(s))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
