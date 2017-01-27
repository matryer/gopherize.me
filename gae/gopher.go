package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/storage"

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
	Images       []string `datastore:",noindex"`
	OriginalURL  string   `datastore:",noindex"`
	URL          string   `datastore:",noindex"`
	ThumbnailURL string   `datastore:",noindex"`
	CTime        time.Time
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
		if err == datastore.ErrNoSuchEntity {
			// gopher doesn't exist - create it

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

func hash(s string) string {
	hash := sha1.New()
	hash.Write([]byte(s))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
