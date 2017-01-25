package server

import (
	"image"
	"image/draw"
	"image/png"
	"net/http"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/file"
	"google.golang.org/appengine/log"
)

type stack []image.Image

func (s server) renderHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	images := strings.Split(q.Get("images"), "|")
	if len(images) == 0 {
		http.Error(w, "Must specify at least one image", http.StatusBadRequest)
		return
	}
	ctx := appengine.NewContext(r)
	bucket, err := file.DefaultBucketName(ctx)
	if err != nil {
		s.responderr(ctx, w, r, http.StatusInternalServerError, err)
		return
	}
	client, err := storage.NewClient(ctx)
	if err != nil {
		s.responderr(ctx, w, r, http.StatusInternalServerError, err)
		return
	}
	imgObjects := s.loadimages(ctx, client.Bucket(bucket), images...)
	var first image.Image
	for _, img := range imgObjects {
		if img == nil {
			continue
		}
		first = img
		break
	}
	if first == nil {
		s.responderr(ctx, w, r, http.StatusInternalServerError, errors.Wrap(err, "Artwork is being updated - please try again later"))
		return
	}
	output := image.NewRGBA(first.Bounds())
	for _, img := range imgObjects {
		if img == nil {
			// skip missing images
			continue
		}
		draw.Draw(output, output.Bounds(), img, image.ZP, draw.Over)
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "attachment; filename=gopherizeme.png;")
	if err := png.Encode(w, output); err != nil {
		log.Errorf(ctx, "PNG encode: %s", err)
		return
	}
}

func (s server) loadimages(ctx context.Context, bucket *storage.BucketHandle, names ...string) []image.Image {
	var wg sync.WaitGroup
	var l sync.Mutex
	images := make(map[string]image.Image)
	errs := make(map[string]error)
	for _, name := range names {
		if len(name) == 0 {
			continue
		}
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			r, err := bucket.Object(name).NewReader(ctx)
			if err != nil {
				l.Lock()
				errs[name] = err
				images[name] = nil
				l.Unlock()
				return
			}
			imageObj, _, err := image.Decode(r)
			if err != nil {
				l.Lock()
				errs[name] = err
				images[name] = nil
				l.Unlock()
				return
			}
			l.Lock()
			images[name] = imageObj
			l.Unlock()
		}(name)
	}
	wg.Wait()
	if len(errs) > 0 {
		log.Warningf(ctx, "processing images: %s", errs)
	}
	imagesList := make([]image.Image, len(names))
	for i, name := range names {
		imagesList[i] = images[name]
	}
	return imagesList
}
