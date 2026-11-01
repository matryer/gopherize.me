package server

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/appengine/v2"
	"google.golang.org/appengine/v2/log"
	"google.golang.org/appengine/v2/memcache"
)

// Limit expensive renders per instance so simultaneous saves cannot exhaust memory.
var renderSlots = make(chan struct{}, 1)

func Render(ctx context.Context, w io.Writer, images []string) error {
	if len(images) == 0 || len(images) > 32 {
		return errors.New("must specify between 1 and 32 images")
	}
	select {
	case renderSlots <- struct{}{}:
		defer func() { <-renderSlots }()
	case <-ctx.Done():
		return ctx.Err()
	}
	bucket, err := DefaultBucketName()
	if err != nil {
		return err
	}
	client, err := storage.NewClient(ctx)
	if err != nil {
		return errors.Wrap(err, "storage.NewClient")
	}
	defer client.Close()
	return renderLayers(w, images, func(name string) (image.Image, error) {
		r, err := client.Bucket(bucket).Object(name).NewReader(ctx)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		img, _, err := image.Decode(r)
		return img, err
	})
}

// Decode and composite one layer at a time instead of retaining every decoded PNG.
func renderLayers(w io.Writer, names []string, load func(string) (image.Image, error)) error {
	var output *image.RGBA
	for _, name := range names {
		if name == "" {
			continue
		}
		img, err := load(name)
		if err != nil {
			return errors.Wrapf(err, "load image %q", name)
		}
		if output == nil {
			output = image.NewRGBA(img.Bounds())
		}
		draw.Draw(output, output.Bounds(), img, image.Point{}, draw.Over)
	}
	if output == nil {
		return errors.New("must specify at least one image")
	}
	return png.Encode(w, output)
}

func (s server) renderHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	imagesStr := q.Get("images")
	images := strings.Split(imagesStr, "|")
	if len(images) == 0 {
		http.Error(w, "Must specify at least one image", http.StatusBadRequest)
		return
	}
	ctx := appengine.NewContext(r)
	cacheItem, err := memcache.Get(ctx, imagesStr)
	if err == nil {
		// exit early - from cache
		log.Debugf(ctx, "cache hit: %s", imagesStr)
		s.respondWithPng(ctx, w, r, cacheItem.Value)
		return
	}
	log.Debugf(ctx, "cache miss - generating image")
	var buf bytes.Buffer
	if err := Render(ctx, &buf, images); err != nil {
		log.Errorf(ctx, "render: %s", err)
		http.Error(w, "Failed to render image :(", http.StatusInternalServerError)
		return
	}
	// write buffer as response
	s.respondWithPng(ctx, w, r, buf.Bytes())
	// put result in cache
	cacheItem = &memcache.Item{
		Key:   imagesStr,
		Value: buf.Bytes(),
	}
	if err := memcache.Set(ctx, cacheItem); err != nil {
		log.Warningf(ctx, "memcache set: %s", err)
	}
}

func (s server) respondWithPng(ctx context.Context, w http.ResponseWriter, r *http.Request, data []byte) {
	w.Header().Set("Content-Type", "image/png")
	if r.URL.Query().Get("dl") == "0" {
		w.Header().Set("Content-Disposition", "inline")
	} else {
		w.Header().Set("Content-Disposition", "attachment; filename=gopherizeme.png;")
	}
	if _, err := w.Write(data); err != nil {
		log.Warningf(ctx, "write png: %s", err)
	}
}
