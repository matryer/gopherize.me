package server

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/appengine"
	"google.golang.org/appengine/file"
)

type artworkResponse struct {
	Categories []Category `json:"categories"`
}

type Category struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Images []Image `json:"images"`
}

type Image struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Href          string `json:"href"`
	ThumbnailHref string `json:"thumbnail_href"`
}

func (s server) artworkHandler(w http.ResponseWriter, r *http.Request) {
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
	q := &storage.Query{
		Prefix: "artwork",
	}
	var objects []*storage.ObjectAttrs
	for {
		bucketlist, err := client.Bucket(bucket).List(ctx, q)
		if err != nil {
			s.responderr(ctx, w, r, http.StatusInternalServerError, err)
			return
		}
		for _, obj := range bucketlist.Results {
			objects = append(objects, obj)
		}
		if bucketlist.Next == nil {
			break
		}
		q = bucketlist.Next
	}
	var categorykeys []string
	categories := make(map[string]*Category)
	imagesCount := 0
	for _, object := range objects {
		if object.ContentType != "image/png" {
			continue
		}
		name := strings.TrimPrefix(object.Name, "artwork/")
		imageName := nicename(name)
		publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", object.Bucket, object.Name)
		catsegs := strings.Split(path.Dir(object.Name), "-")
		if len(catsegs) != 2 {
			continue // skip
		}
		cat := catsegs[1]
		category, ok := categories[cat]
		if !ok {
			categorykeys = append(categorykeys, cat)
			category = &Category{
				ID:   path.Dir(object.Name),
				Name: cat,
			}
			categories[cat] = category
		}
		category.Images = append(category.Images, Image{
			ID:            object.Name,
			Name:          imageName,
			Href:          publicURL,
			ThumbnailHref: publicURL,
		})
		imagesCount++
	}
	var orderedCats []Category
	for _, cat := range categorykeys {
		orderedCats = append(orderedCats, *categories[cat])
	}
	res := artworkResponse{
		Categories: orderedCats,
	}
	s.respond(ctx, w, r, http.StatusOK, res)
}

func nicename(s string) string {
	ext := path.Ext(s)
	base := path.Base(s)
	name := strings.TrimSuffix(base, ext)
	name = strings.Replace(name, "_", " ", -1)
	if segs := strings.Split(name, "-"); len(segs) > 1 {
		name = segs[1]
	}
	return name
}
