package server

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
)

func TestStack(t *testing.T) {
	is := is.New(t)
	_, err := loadimage("Female.png")
	is.NoErr(err)
	_, err := loadimage("Beard.png")
	is.NoErr(err)
}

func loadimage(name string) (image.Image, error) {
	var img image.Image
	r, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		return img, err
	}
	defer r.Close()
	theimage, _, err := image.Decode(r)
	if err != nil {
		return img, err
	}
	return theimage, nil
}
