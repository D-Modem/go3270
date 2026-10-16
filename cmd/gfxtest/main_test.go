package main

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/racingmars/go3270"
)

// pcomm is part of IBM Personal Communications' palette.
var pcomm = palette{colors: go3270.Palette{0: {0, 0, 0, 255}, 2: {255, 0, 0, 255}, 7: {255, 255, 255, 255},
	10: {255, 160, 0, 255}, 15: {96, 96, 96, 255}}}

// TestQuantize: scaled, each pel in one bitmap at most, gray to gray.
func TestQuantize(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 80, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 80; x++ {
			if x < 40 {
				img.Set(x, y, color.RGBA{0x60, 0x60, 0x60, 0xff})
			} else {
				img.Set(x, y, color.RGBA{0xff, 0xa0, 0, 0xff})
			}
		}
	}
	bms := quantize(img, 40, 40, pcomm) // to 40x20
	got := map[int]int{}
	seen := map[[2]int]bool{}
	for _, b := range bms {
		if len(b.Rows) != 20 || len(b.Rows[0]) != 40 {
			t.Fatalf("%dx%d", len(b.Rows[0]), len(b.Rows))
		}
		for y, row := range b.Rows {
			for x, c := range row {
				if c == '#' {
					if seen[[2]int{x, y}] {
						t.Fatalf("pel %d,%d twice", x, y)
					}
					seen[[2]int{x, y}] = true
					got[int(b.Color)&0x0f]++
				}
			}
		}
	}
	if got[15] != 400 || got[10] != 400 {
		t.Errorf("by color: %v", got)
	}
	if f := os.Getenv("PREVIEW_IMAGE"); f != "" {
		img, err := loadImage(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := writePreview(os.Getenv("PREVIEW_DIR")+"/q.png", "O", pcomm, quantize(img, 233, 233, pcomm)); err != nil {
			t.Fatal(err)
		}
	}
}
