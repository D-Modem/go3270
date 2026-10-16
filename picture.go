package go3270

import (
	"image"
	"image/color"
	"math"
	"sort"
)

// Palette is a terminal's graphics colors: the color of each index (1
// blue, 2 red, ... 7 white, and on some 8 to 15), 0 being the background.
type Palette map[int]color.RGBA

// basePalette is the 3270's 8 colors, for a terminal that does not say.
var basePalette = Palette{0: {0, 0, 0, 255}, 1: {0, 0, 255, 255}, 2: {255, 0, 0, 255}, 3: {255, 0, 255, 255},
	4: {0, 255, 0, 255}, 5: {0, 255, 255, 255}, 6: {255, 255, 0, 255}, 7: {255, 255, 255, 255}}

// GraphicColors is the terminal's graphics colors: from its Graphic Color
// query reply (0xB4) if it sent one (IBM Personal Communications has 16:
// orange, purple, dark blue, green and turquoise, mustard and grey besides
// the 8), else the 8 base colors.
func GraphicColors(d DevInfo) Palette {
	if p, ok := parseGraphicColors(d.QueryReply(0xb4)); ok {
		return p
	}
	out := Palette{}
	for i, c := range basePalette {
		out[i] = c
	}
	return out
}

// parseGraphicColors reads a Graphic Color query reply, from the QCODE on:
// QCODE, flags, the length of a color value (n), a byte, 6 more, then an
// entry a color: its index (2 bytes) and n bytes of value, the first
// three red, blue, green (as PCOMM sends them).
func parseGraphicColors(r []byte) (Palette, bool) {
	if len(r) < 10 || r[0] != 0xb4 || r[2] < 3 {
		return nil, false
	}
	n := int(r[2])
	p := Palette{}
	for e := r[10:]; len(e) >= 2+n; e = e[2+n:] {
		if i := int(e[0])<<8 | int(e[1]); i < 16 {
			p[i] = color.RGBA{e[2], e[4], e[3], 255}
		}
	}
	if len(p) < 8 {
		return nil, false
	}
	p[0] = color.RGBA{0, 0, 0, 255} // the background
	return p, true
}

// DitherBitmaps is img in the palette's colors (Floyd-Steinberg error
// diffusion), a Bitmap a color, each pel in one at most, black pels in
// none (the background shows): for ImagePictures with MixOverpaint, which
// draws them a pel for a pixel.
func DitherBitmaps(img image.Image, p Palette) []Bitmap {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	var idx []int
	for i := range p {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	type rgb [3]float64
	cur, next := make([]rgb, w+2), make([]rgb, w+2)
	rows := map[int][][]byte{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, a := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			// transparent is black: the background
			r, g, bl = r*a/0xffff, g*a/0xffff, bl*a/0xffff
			cur[x+1][0] += float64(r >> 8)
			cur[x+1][1] += float64(g >> 8)
			cur[x+1][2] += float64(bl >> 8)
		}
		for x := 0; x < w; x++ {
			want := cur[x+1]
			best, bestD := 0, math.MaxFloat64
			for _, i := range idx {
				c := p[i]
				dr, dg, db := want[0]-float64(c.R), want[1]-float64(c.G), want[2]-float64(c.B)
				// weighted for the eye: green counts most
				if d := 2*dr*dr + 4*dg*dg + 3*db*db; d < bestD {
					best, bestD = i, d
				}
			}
			c := p[best]
			e := rgb{want[0] - float64(c.R), want[1] - float64(c.G), want[2] - float64(c.B)}
			for k := 0; k < 3; k++ {
				cur[x+2][k] += e[k] * 7 / 16
				next[x][k] += e[k] * 3 / 16
				next[x+1][k] += e[k] * 5 / 16
				next[x+2][k] += e[k] * 1 / 16
			}
			if c.R == 0 && c.G == 0 && c.B == 0 {
				continue // black: the background
			}
			if rows[best] == nil {
				rows[best] = make([][]byte, h)
				for i := range rows[best] {
					rows[best][i] = make([]byte, w)
					for j := range rows[best][i] {
						rows[best][i][j] = '.'
					}
				}
			}
			rows[best][y][x] = '#'
		}
		cur, next = next, make([]rgb, w+2)
	}
	var out []Bitmap
	for _, i := range idx {
		if r := rows[i]; r != nil {
			s := make([]string, h)
			for j := range r {
				s[j] = string(r[j])
			}
			out = append(out, Bitmap{Color: Color(0xf0 + i), Rows: s})
		}
	}
	return out
}
