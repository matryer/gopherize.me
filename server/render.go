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
	imgObjects, err := s.loadimages(ctx, client.Bucket(bucket), images...)
	if err != nil {
		s.responderr(ctx, w, r, http.StatusInternalServerError, errors.Wrap(err, "loading images"))
		return
	}
	first := imgObjects[0]
	output := image.NewRGBA(first.Bounds())
	for _, img := range imgObjects {
		draw.Draw(output, output.Bounds(), img, image.ZP, draw.Over)
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "attachment; filename=gopherizeme.png;")
	if err := png.Encode(w, output); err != nil {
		log.Errorf(ctx, "PNG encode: %s", err)
		return
	}
}

func (s server) loadimages(ctx context.Context, bucket *storage.BucketHandle, names ...string) ([]image.Image, error) {
	var wg sync.WaitGroup
	var l sync.Mutex
	images := make(map[string]image.Image)
	errs := make(map[string]error)
	for _, name := range names {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			r, err := bucket.Object(name).NewReader(ctx)
			if err != nil {
				l.Lock()
				errs[name] = err
				l.Unlock()
				return
			}
			imageObj, _, err := image.Decode(r)
			if err != nil {
				l.Lock()
				errs[name] = err
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
		log.Errorf(ctx, "processing images: %s", errs)
		return nil, errors.New("error processing images")
	}
	imagesList := make([]image.Image, len(images))
	for i, name := range names {
		imagesList[i] = images[name]
	}
	return imagesList, nil
}
