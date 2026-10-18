package go3270

import (
	"encoding/hex"
	"image"
	"image/color"
	"testing"
)

// pcommColors is IBM Personal Communications' Graphic Color query reply.
const pcommColors = "b4000400ffff00100010000000000000000100ff00000002ff0000000003ffff000000040000ff00000500ffff000006ff00ff000007ffffff00000800000000000900a00000000aff00a000000ba0a00000000c0000a000000d00a0a000000ea000a000000f60606000"

// TestGraphicColors: PCOMM's Graphic Color reply gives 16 colors by index; a
// short reply, the 8 base colors.
func TestGraphicColors(t *testing.T) {
	r, _ := hex.DecodeString(pcommColors)
	p := GraphicColors(&deviceInfo{replies: map[byte][]byte{0xb4: r}})
	if len(p) != 16 {
		t.Fatalf("%d colors", len(p))
	}
	for i, want := range map[int]color.RGBA{1: {0, 0, 0xff, 0xff}, 2: {0xff, 0, 0, 0xff}, 3: {0xff, 0, 0xff, 0xff},
		4: {0, 0xff, 0, 0xff}, 6: {0xff, 0xff, 0, 0xff}, 7: {0xff, 0xff, 0xff, 0xff}, 10: {0xff, 0xa0, 0, 0xff},
		11: {0xa0, 0, 0xa0, 0xff}, 15: {0x60, 0x60, 0x60, 0xff}} {
		if p[i] != want {
			t.Errorf("color %d: %v, want %v", i, p[i], want)
		}
	}
	if p := GraphicColors(&deviceInfo{replies: map[byte][]byte{0xb4: {0xb4, 0, 4}}}); len(p) != 8 {
		t.Errorf("a short reply: %d colors", len(p))
	}
}

// TestDitherBitmaps: each pel in one bitmap at most, black in none, a
// palette color to itself.
func TestDitherBitmaps(t *testing.T) {
	r, _ := hex.DecodeString(pcommColors)
	p, _ := parseGraphicColors(r)
	img := image.NewRGBA(image.Rect(0, 0, 60, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 60; x++ {
			c := []color.RGBA{{0x60, 0x60, 0x60, 0xff}, {0xff, 0xa0, 0, 0xff}, {0, 0, 0, 0xff}}[x/20]
			img.Set(x, y, c)
		}
	}
	got := map[int]int{}
	seen := map[[2]int]bool{}
	for _, b := range DitherBitmaps(img, p) {
		for y, row := range b.Rows {
			for x := range row {
				if row[x] == '#' {
					if seen[[2]int{x, y}] {
						t.Fatalf("pel %d,%d twice", x, y)
					}
					seen[[2]int{x, y}] = true
					got[int(b.Color)&0x0f]++
				}
			}
		}
	}
	if got[15] != 400 || got[10] != 400 || len(got) != 2 {
		t.Errorf("by color: %v", got)
	}
}
