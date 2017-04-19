package server

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/appengine"
	"google.golang.org/appengine/blobstore"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/file"
	"google.golang.org/appengine/image"
	"google.golang.org/appengine/log"
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

func (s server) getArtwork(ctx context.Context) (*artworkResponse, error) {
	artworkKey := datastore.NewKey(ctx, "Artwork", "latest", 0, nil)
	var artwork artworkResponse
	err := datastore.Get(ctx, artworkKey, &artwork)
	if err != nil {
		return nil, err
	}
	return &artwork, nil
}

func (s server) generateArtwork(ctx context.Context) (*artworkResponse, error) {
	bucket, err := file.DefaultBucketName(ctx)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	q := &storage.Query{
		Prefix: "artwork",
	}
	var objects []*storage.ObjectAttrs
	ctxObjList, _ := context.WithTimeout(ctx, 10*time.Second)
	bucketlist := client.Bucket(bucket).Objects(ctxObjList, q)
	for {
		obj, err := bucketlist.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		objects = append(objects, obj)
	}
	var categorykeys []string
	categories := make(map[string]*Category)
	for _, object := range objects {
		if object.ContentType != "image/png" {
			continue
		}
		log.Debugf(ctx, "parsing %s", object.Name)
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
		ctxThis, _ := context.WithTimeout(ctx, 10*time.Second)
		if blobkey, err := blobstore.BlobKeyForFile(ctxThis, "/gs/"+object.Bucket+"/"+object.Name); err == nil {
			serveURL, err := image.ServingURL(ctxThis, blobkey, &image.ServingURLOptions{
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
		break // TODO: remove me
	}
	var orderedCats []Category
	for _, cat := range categorykeys {
		orderedCats = append(orderedCats, *categories[cat])
	}
	res := artworkResponse{
		Categories: orderedCats,
	}
	// calculate total number of combinations
	res.TotalCombinations = 1
	for _, cat := range res.Categories {
		res.TotalCombinations *= len(cat.Images) + 1
	}
	log.Debugf(ctx, "found %d categories", res.Categories)
	log.Debugf(ctx, "%d total combinations", res.TotalCombinations)
	artworkKey := datastore.NewKey(ctx, "Artwork", "latest", 0, nil)
	_, err = datastore.Put(ctx, artworkKey, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s server) artworkHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	artwork, err := s.getArtwork(ctx)
	if err != nil {
		s.responderr(ctx, w, r, http.StatusInternalServerError, err)
		return
	}
	s.respond(ctx, w, r, http.StatusOK, artwork)
}

func (s server) generateArtworkHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	log.Debugf(ctx, "Generating artwork...")
	_, err := s.generateArtwork(ctx)
	if err != nil {
		log.Errorf(ctx, "Generating artwork failed: %s", err)
		s.responderr(ctx, w, r, http.StatusInternalServerError, err)
		return
	}
	log.Debugf(ctx, "Generating artwork complete")
	s.respond(ctx, w, r, http.StatusOK, nil)
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
