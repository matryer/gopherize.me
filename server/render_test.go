package server

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestRenderLayersPreservesOrderAndTransparency(t *testing.T) {
	bottom := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	bottom.Set(0, 0, color.NRGBA{R: 255, A: 255})
	bottom.Set(1, 0, color.NRGBA{R: 255, A: 255})
	top := image.NewNRGBA(bottom.Bounds())
	top.Set(0, 0, color.NRGBA{B: 255, A: 255})
	layers := map[string]image.Image{"bottom": bottom, "top": top}
	var out bytes.Buffer
	err := renderLayers(&out, []string{"bottom", "", "top"}, func(name string) (image.Image, error) { return layers[name], nil })
	if err != nil {
		t.Fatal(err)
	}
	got, err := png.Decode(&out)
	if err != nil {
		t.Fatal(err)
	}
	if c := color.NRGBAModel.Convert(got.At(0, 0)); c != (color.NRGBA{B: 255, A: 255}) {
		t.Fatalf("top pixel = %v", c)
	}
	if c := color.NRGBAModel.Convert(got.At(1, 0)); c != (color.NRGBA{R: 255, A: 255}) {
		t.Fatalf("transparent pixel = %v", c)
	}
}

func TestRenderLayersRejectsMissingArtwork(t *testing.T) {
	want := errors.New("missing artwork")
	var out bytes.Buffer
	if err := renderLayers(&out, []string{"missing"}, func(string) (image.Image, error) { return nil, want }); !errors.Is(err, want) {
		t.Fatalf("got %v", err)
	}
	if out.Len() != 0 {
		t.Fatal("wrote a partial PNG")
	}
	if err := renderLayers(&out, []string{""}, nil); err == nil {
		t.Fatal("accepted an empty selection")
	}
}
