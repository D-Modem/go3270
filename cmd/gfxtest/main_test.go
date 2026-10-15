package main

import (
	"encoding/hex"
	"image"
	"image/color"
	"os"
	"testing"
)

// pcommColors is IBM Personal Communications' Graphic Color query reply.
const pcommColors = "b4000400ffff00100010000000000000000100ff00000002ff0000000003ffff000000040000ff00000500ffff000006ff00ff000007ffffff00000800000000000900a00000000aff00a000000ba0a00000000c0000a000000d00a0a000000ea000a000000f60606000"

func TestGraphicColors(t *testing.T) {
	r, _ := hex.DecodeString(pcommColors)
	p, ok := graphicColors(r)
	if !ok || len(p.colors) != 16 {
		t.Fatalf("%v %v", ok, p)
	}
	for i, want := range map[int]color.RGBA{1: {0, 0, 0xff, 0xff}, 2: {0xff, 0, 0, 0xff}, 3: {0xff, 0, 0xff, 0xff},
		4: {0, 0xff, 0, 0xff}, 6: {0xff, 0xff, 0, 0xff}, 7: {0xff, 0xff, 0xff, 0xff}, 10: {0xff, 0xa0, 0, 0xff},
		11: {0xa0, 0, 0xa0, 0xff}, 15: {0x60, 0x60, 0x60, 0xff}} {
		if p.colors[i] != want {
			t.Errorf("color %d: %v, want %v", i, p.colors[i], want)
		}
	}
	if _, ok := graphicColors([]byte{0xb4, 0, 4}); ok {
		t.Error("a short reply taken")
	}
}

// TestQuantize: each pel in one bitmap at most, none black, gray to gray.
func TestQuantize(t *testing.T) {
	r, _ := hex.DecodeString(pcommColors)
	pal, _ := graphicColors(r)
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			if x < 20 {
				img.Set(x, y, color.RGBA{0x60, 0x60, 0x60, 0xff})
			} else {
				img.Set(x, y, color.RGBA{0xff, 0xa0, 0, 0xff})
			}
		}
	}
	bms := quantize(img, 40, 20, pal)
	seen := map[[2]int]bool{}
	got := map[int]int{}
	for _, b := range bms {
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
		for name, p := range map[string]palette{"16": pal, "8": (&sess{dev: nil}).basePalette()} {
			if err := writePreview(os.Getenv("PREVIEW_DIR")+"/q"+name+".png", "O", p, quantize(img, 233, 233, p)); err != nil {
				t.Fatal(err)
			}
		}
	}
}
