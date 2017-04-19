package server

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/appengine"
	"google.golang.org/appengine/blobstore"
	"google.golang.org/appengine/file"
	"google.golang.org/appengine/image"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
)

type artworkResponse struct {
	Categories        []Category `json:"categories"`
	TotalCombinations int        `json:"total_combinations"`
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

	var res artworkResponse

	if len(r.URL.Query().Get("nocache")) == 0 {
		_, err := memcache.Gob.Get(ctx, "artwork", &res)
		if err == nil {
			// exit early - from cache
			log.Debugf(ctx, "cache hit")
			s.respond(ctx, w, r, http.StatusOK, res)
			return
		}
		log.Debugf(ctx, "cache miss - generating artwork data")
	} else {
		log.Debugf(ctx, "skipping cache - generating artwork data")
	}

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
	bucketlist := client.Bucket(bucket).Objects(ctx, q)
	for {
		obj, err := bucketlist.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			s.responderr(ctx, w, r, http.StatusInternalServerError, err)
			return
		}
		objects = append(objects, obj)
	}
	var categorykeys []string
	categories := make(map[string]*Category)
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

		// get thumbnail URL
		thumbURL := publicURL
		if blobkey, err := blobstore.BlobKeyForFile(ctx, "/gs/"+object.Bucket+"/"+object.Name); err == nil {
			serveURL, err := image.ServingURL(ctx, blobkey, &image.ServingURLOptions{
				Secure: true,
				Size:   71,
			})
			if err == nil {
				thumbURL = serveURL.String()
			} else {
				log.Warningf(ctx, "image.ServingURL: %s", err)
			}
		} else {
			log.Warningf(ctx, "blobstore.BlobKeyForFile: %s", err)
		}

		category.Images = append(category.Images, Image{
			ID:            object.Name,
			Name:          imageName,
			Href:          publicURL,
			ThumbnailHref: thumbURL,
		})
	}
	var orderedCats []Category
	for _, cat := range categorykeys {
		orderedCats = append(orderedCats, *categories[cat])
	}
	res = artworkResponse{
		Categories: orderedCats,
	}

	// calculate total number of combinations
	res.TotalCombinations = 1
	for _, cat := range res.Categories {
		res.TotalCombinations *= len(cat.Images) + 1
	}

	cacheItem := &memcache.Item{
		Key:        "artwork",
		Object:     res,
		Expiration: 24 * time.Hour,
	}
	if err := memcache.Gob.Set(ctx, cacheItem); err != nil {
		log.Warningf(ctx, "memcache set: %s", err)
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
