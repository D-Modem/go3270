// This file is part of https://github.com/racingmars/go3270/
// Copyright 2026 by the go3270-sec contributors, licensed under the MIT
// license. See LICENSE in the project root for license information.

package go3270

// 3270 graphics: what the client says it can do (its query replies), and
// the structured fields to use it.
//
//   - Programmed symbols: the client's loadable character sets (read/write
//     storage, RWS) are loaded with bitmap symbols by the Load Programmed
//     Symbols structured field (LoadPS), and shown by giving a field the
//     character set's local ID (Field.CharSet) and the symbols' code points
//     (Field.Raw). See GA23-0059-07, 5.13 and 6.12.
//   - Vector graphics: Object Picture structured fields carry graphics
//     segments (ImagePicture). GA23-0059 defines the structured field; the
//     drawing orders inside it are the graphics products' (GDDM, GOCA).
//
// Neither is supported by the x3270 family; IBM Personal Communications,
// HostExplorer and Quick3270 support both. Always check DevInfo first.

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"sort"
)

// SymbolSets is what a client's Character Sets query reply says about its
// character sets.
type SymbolSets struct {
	// GraphicEscape: the alternate character set is reachable by GE.
	GraphicEscape bool

	// Loadable: the client takes the Load Programmed Symbols structured
	// field. It can only be used if Sets has a Loadable set, too.
	Loadable bool

	// Extended: the extended form of Load PS (with the cell size) is
	// supported.
	Extended bool

	// SizeMatch: a Load PS cell size must match the slot size exactly.
	SizeMatch bool

	// SlotWidth, SlotHeight: the default character slot, in pels.
	SlotWidth, SlotHeight int

	// Formats: bit i (counting from the most significant bit) set when Load
	// PS format type i is supported. See SupportsFormat.
	Formats uint32

	// Sets are the character sets the client has, in the order it gave.
	Sets []SymbolSet
}

// SymbolSet is one character set of a client.
type SymbolSet struct {
	RWS      byte // device specific ID: LoadPS.RWS for a loadable set
	LCID     byte // the local ID it has now (0xFF: none)
	Loadable bool
	Triple   bool // triple plane (color) set
	DBCS     bool
	// slot size, if the client gives one per set (0: SlotWidth/Height)
	SlotWidth, SlotHeight int
	CGCSGID               uint32 // character set and code page
}

// SupportsFormat reports whether Load PS format type t is supported.
func (s *SymbolSets) SupportsFormat(t int) bool {
	if s == nil || t < 0 || t > 31 {
		return false
	}
	return s.Formats&(1<<(31-t)) != 0
}

// LoadableSets lists the sets that programmed symbols can be loaded into.
func (s *SymbolSets) LoadableSets() []SymbolSet {
	if s == nil || !s.Loadable {
		return nil
	}
	var out []SymbolSet
	for _, set := range s.Sets {
		if set.Loadable {
			out = append(out, set)
		}
	}
	return out
}

// getSymbolSets parses a Character Sets query reply. buf begins with
// {0x81, 0x85}: offsets are 2 less than in GA23-0059's table.
func getSymbolSets(buf []byte) *SymbolSets {
	if len(buf) < 11 || buf[0] != 0x81 || buf[1] != 0x85 {
		return nil
	}
	f1, f2 := buf[2], buf[3]
	s := &SymbolSets{
		GraphicEscape: f1&0x80 != 0,
		Loadable:      f1&0x20 != 0,
		Extended:      f1&0x10 != 0,
		SizeMatch:     f2&0x40 == 0,
		SlotWidth:     int(buf[4]),
		SlotHeight:    int(buf[5]),
		Formats: uint32(buf[6])<<24 | uint32(buf[7])<<16 |
			uint32(buf[8])<<8 | uint32(buf[9]),
	}
	ms, ch2, gf := f1&0x08 != 0, f1&0x04 != 0, f1&0x02 != 0
	dl := int(buf[10])
	if dl < 3 {
		return s
	}
	for pos := 11; pos+dl <= len(buf) && len(s.Sets) < 64; pos += dl {
		d := buf[pos : pos+dl]
		set := SymbolSet{
			RWS:      d[0],
			Loadable: d[1]&0x80 != 0,
			Triple:   d[1]&0x40 != 0,
			DBCS:     d[1]&0x20 != 0,
			LCID:     d[2],
		}
		i := 3
		if ms && i+2 <= len(d) {
			set.SlotWidth, set.SlotHeight = int(d[i]), int(d[i+1])
			i += 2
		}
		if ch2 {
			i += 2
		}
		if gf && i+4 <= len(d) {
			set.CGCSGID = uint32(d[i])<<24 | uint32(d[i+1])<<16 |
				uint32(d[i+2])<<8 | uint32(d[i+3])
		}
		s.Sets = append(s.Sets, set)
	}
	return s
}

// getCellSize returns the default cell size (AW, AH) from a Usable Area
// query reply. buf begins with {0x81, 0x81}.
func getCellSize(buf []byte) (w, h int) {
	if len(buf) < 19 {
		return 0, 0
	}
	return int(buf[17]), int(buf[18])
}

// QueryReply returns a copy of the reply with QCODE qcode; nil if none came.
func (d *deviceInfo) QueryReply(qcode byte) []byte {
	r, ok := d.replies[qcode]
	if !ok {
		return nil
	}
	return append([]byte(nil), r...)
}

// QueryCodes lists the QCODEs of the replies kept, ascending.
func (d *deviceInfo) QueryCodes() []byte {
	var out []byte
	for k := range d.replies {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// CellSize is the default cell in pels, from the Usable Area reply.
func (d *deviceInfo) CellSize() (int, int) { return d.cellW, d.cellH }

// SymbolSets is the Character Sets reply, parsed; nil if none came.
func (d *deviceInfo) SymbolSets() *SymbolSets { return d.symbols }

// SupportsVectorGraphics reports whether the Segment reply (0xB0) came or the
// Summary lists it.
func (d *deviceInfo) SupportsVectorGraphics() bool {
	if _, ok := d.replies[0xb0]; ok {
		return true
	}
	// IBM Personal Communications lists the graphics replies in its
	// Summary but does not send them for a Query
	return bytes.IndexByte(d.summary(), 0xb0) >= 0
}

// summary is the QCODEs the Summary query reply lists: the replies the
// client says it has.
func (d *deviceInfo) summary() []byte {
	if r := d.replies[0x80]; len(r) > 1 {
		return r[1:]
	}
	return nil
}

// telnetEscape doubles every 0xFF (IAC) in b.
func telnetEscape(b []byte) []byte {
	if bytes.IndexByte(b, 0xff) < 0 {
		return b
	}
	out := make([]byte, 0, len(b)+8)
	for _, c := range b {
		out = append(out, c)
		if c == 0xff {
			out = append(out, 0xff)
		}
	}
	return out
}

// WriteStructuredFields sends a Write Structured Field command with the
// given structured fields. Each field is given from its SFID on; its length
// is put in front of it.
func WriteStructuredFields(conn net.Conn, fields ...[]byte) error {
	var b bytes.Buffer
	b.WriteByte(0xf3) // WSF
	for _, f := range fields {
		if len(f)+2 > 0xffff {
			return errors.New("go3270: structured field too long")
		}
		l := len(f) + 2
		b.Write([]byte{byte(l >> 8), byte(l)})
		b.Write(f)
	}
	out := append(telnetEscape(b.Bytes()), 0xff, 0xef) // IAC EOR
	_, err := conn.Write(out)
	return err
}

// Symbol is the picture of one programmed symbol: a row of text for each
// row of pels, top to bottom, '#' (or 'X', '1', '*') for a pel that is on,
// anything else for one that is off.
type Symbol []string

// on reports whether the pel at row, col is on ('#', 'X', 'x', '1' or '*'); off
// past the symbol's edge.
func (s Symbol) on(row, col int) bool {
	if row >= len(s) || col >= len(s[row]) {
		return false
	}
	switch s[row][col] {
	case '#', 'X', 'x', '1', '*':
		return true
	}
	return false
}

// LoadPS is the Load Programmed Symbols structured field (GA23-0059 5.13):
// Symbols loaded into the client's symbol store RWS, from code point First
// on, to be shown through the character set ID LCID.
type LoadPS struct {
	RWS   byte // the store: a loadable SymbolSet's RWS
	LCID  byte // the local ID to give it, 0x40 to 0xEF
	First byte // the first symbol's code point, 0x41 to 0xFE
	Clear bool // clear the store's other symbols

	// Format is the data format: 1 (a 9 by 16 cell: 2 bytes of the left
	// column, then 16 bytes of the other 8 columns, a byte a row) or 3
	// (rows of Width pels, top to bottom). Check SymbolSets.SupportsFormat.
	Format int

	// Width and Height of a format 3 cell: the client's slot size, unless
	// Extended.
	Width, Height int

	// Extended sends the extended form of Load PS, which tells the client
	// Width and Height (SymbolSets.Extended must be true).
	Extended bool

	// Plane is the plane of a triple-plane (color) set the symbols are
	// loaded into: PlaneBlue, PlaneRed or PlaneGreen; 0, all three (or a
	// single-plane set). A plane needs the extended form, which it implies.
	// A cell shows its planes' colors, added (blue and red make pink...),
	// where its color attribute is White (neutral, X'F7'), GA23-0059 4.5.
	Plane byte

	Symbols []Symbol
}

// The planes of a triple-plane symbol set, for LoadPS.Plane.
const (
	PlaneBlue  byte = 1
	PlaneRed   byte = 2
	PlaneGreen byte = 4
)

// StructuredField is the Load PS structured field, from its SFID on, for
// WriteStructuredFields.
func (l LoadPS) StructuredField() ([]byte, error) {
	if l.LCID < 0x40 || l.LCID > 0xef {
		return nil, fmt.Errorf("go3270: Load PS LCID %#x is not 0x40 to 0xEF", l.LCID)
	}
	if l.First < 0x41 || l.First > 0xfe {
		return nil, fmt.Errorf("go3270: Load PS first code point %#x is not 0x41 to 0xFE", l.First)
	}
	if int(l.First)+len(l.Symbols)-1 > 0xfe {
		return nil, errors.New("go3270: Load PS: more symbols than code points")
	}
	switch l.Plane {
	case 0, PlaneBlue, PlaneRed, PlaneGreen:
	default:
		return nil, fmt.Errorf("go3270: Load PS plane %#x is not blue, red or green", l.Plane)
	}
	flags := byte(0)
	if l.Clear {
		flags |= 0x40
	}
	var data bytes.Buffer
	var ext []byte
	switch l.Format {
	case 1:
		flags |= 1
		for _, s := range l.Symbols {
			var col0 uint16
			for r := 0; r < 16; r++ {
				if s.on(r, 0) {
					col0 |= 1 << (15 - r)
				}
			}
			data.Write([]byte{byte(col0 >> 8), byte(col0)})
			for r := 0; r < 16; r++ {
				var row byte
				for c := 1; c <= 8; c++ {
					if s.on(r, c) {
						row |= 1 << (8 - c)
					}
				}
				data.WriteByte(row)
			}
		}
	case 3:
		flags |= 3
		w, h := l.Width, l.Height
		if w <= 0 || h <= 0 || w > 255 || h > 255 {
			return nil, errors.New("go3270: Load PS format 3 needs the cell's Width and Height")
		}
		// GA23-0059 5.13.3: the rows' bits follow one another; each
		// symbol is padded to a whole byte
		for _, s := range l.Symbols {
			var acc, n uint
			for r := 0; r < h; r++ {
				for c := 0; c < w; c++ {
					acc <<= 1
					if s.on(r, c) {
						acc |= 1
					}
					if n++; n == 8 {
						data.WriteByte(byte(acc))
						acc, n = 0, 0
					}
				}
			}
			if n > 0 {
				data.WriteByte(byte(acc << (8 - n)))
			}
		}
		if l.Extended || l.Plane != 0 {
			// parameters' length (itself included), flags, LW, LH
			ext = []byte{4, 0, byte(w), byte(h)}
		}
	default:
		return nil, fmt.Errorf("go3270: Load PS format %d is not supported here", l.Format)
	}
	if l.Plane != 0 {
		// GA23-0059 5.13.2: through the color planes (byte 12), after the
		// subsection ID (one-byte codes); format 1's cell is 9 by 16
		if ext == nil {
			ext = []byte{4, 0, 9, 16}
		}
		ext = append(ext, 0, l.Plane)
		ext[0] = byte(len(ext))
	}
	if ext != nil {
		flags |= 0x80
	}
	sf := []byte{0x06, flags, l.LCID, l.First, l.RWS}
	sf = append(sf, ext...)
	return append(sf, data.Bytes()...), nil
}

// Bitmap is one plane of an image for ImagePicture: Rows of '#' (on) and
// anything else (off), all of the same length, drawn in Color.
type Bitmap struct {
	Color Color // Blue, Red, Pink, Green, Turquoise, Yellow or White
	Rows  []string
}

// gocaColor maps a field color to the graphics color index.
func gocaColor(c Color) byte {
	if c == DefaultColor {
		return 7
	}
	return byte(c) & 0x0f // 0xf1 blue .. 0xf7 white: 1 .. 7
}

// ImagePicture is an Object Picture structured field (GA23-0059 5.38), for
// WriteStructuredFields, that draws bitmaps with their top left corner at
// x, y: graphics coordinates, 0, 0 the middle of the screen, y upwards.
// Planes are combined by exclusive or, so a blue and a red plane make pink
// where both are on.
//
// The orders inside (Begin Segment, Set Mix, Set Current Position, Set
// Color, Begin/Image Data/End Image) are 3270 native vector graphics as
// GDDM sends them; experimental: check DevInfo.SupportsVectorGraphics, and
// try it on your emulator.
func ImagePicture(x, y int, planes ...Bitmap) ([]byte, error) {
	var seg bytes.Buffer
	seg.Write([]byte{0x0c, 4}) // Set Mix: exclusive or
	for _, p := range planes {
		if len(p.Rows) == 0 {
			continue
		}
		w, h := len(p.Rows[0]), len(p.Rows)
		if w > 2040 || h > 2040 {
			return nil, errors.New("go3270: image too large")
		}
		bpr := (w + 7) / 8
		seg.Write([]byte{0x21, 4, byte(x >> 8), byte(x), byte(y >> 8), byte(y)})       // Set Current Position
		seg.Write([]byte{0x0a, gocaColor(p.Color)})                                    // Set Color
		seg.Write([]byte{0x91, 6, 0, 0, byte(w >> 8), byte(w), byte(h >> 8), byte(h)}) // Begin Image
		for _, row := range p.Rows {
			if len(row) != w {
				return nil, errors.New("go3270: image rows of different lengths")
			}
			line := make([]byte, bpr)
			for c := 0; c < w; c++ {
				switch row[c] {
				case '#', 'X', 'x', '1', '*':
					line[c/8] |= 0x80 >> (c % 8)
				}
			}
			seg.Write([]byte{0x92, byte(bpr)}) // Image Data
			seg.Write(line)
		}
		seg.Write([]byte{0x93, 2, 0, 0}) // End Image
	}
	segLen := seg.Len()
	if segLen > 0xffff-64 {
		return nil, errors.New("go3270: image too large")
	}
	sf := []byte{0x0f, 0x10, 0x00, 0xc0, 0x00} // Object Picture, PID 0, first and last, immediate
	// Begin Segment: name "BBS1" (EBCDIC), visible, no highlight, new segment
	sf = append(sf, 0x70, 12, 0xc2, 0xc2, 0xe2, 0xf1, 0x74, 0x68,
		byte(segLen>>8), byte(segLen), 0, 0, 0, 0)
	return append(sf, seg.Bytes()...), nil
}

// Mix is how a picture's pels combine with what is already drawn.
type Mix byte

const (
	MixOverpaint Mix = 2 // the color drawn replaces what was there
	MixXor       Mix = 4 // exclusive or: a blue and a red plane make pink
)

// maxSegment is the most a graphics segment holds here: its length is two
// bytes, and the structured field around it has its own header.
const maxSegment = 0xffff - 64

// PictureOptions are ImagePictures' options.
type PictureOptions struct {
	// Mix is how pels combine with what is drawn: MixOverpaint for an
	// image with a bitmap a color, each pel in one (exclusive or does not
	// combine blue with the other colors on IBM Personal Communications);
	// MixXor if 0.
	Mix Mix
	// Segment is the number of the first graphics segment (named "B001"
	// on): pictures sent together need segments of their own. 1 if 0.
	Segment int
}

// ImagePictures is Object Picture structured fields, for
// WriteStructuredFields, that draw bitmaps with their top left corner at
// x, y (graphics coordinates, 0, 0 the middle of the screen, y upwards),
// and the number of the next segment, for pictures sent with these.
//
// Unlike ImagePicture, each bitmap is cropped to its pels that are on,
// and a picture too large for one segment (64K) is split into several,
// each in its own structured field, a bitmap into bands of rows if need
// be.
func ImagePictures(x, y int, o PictureOptions, planes ...Bitmap) ([][]byte, int, error) {
	mix, first := o.Mix, o.Segment
	if mix == 0 {
		mix = MixXor
	}
	if first < 1 {
		first = 1
	}
	var parts [][]byte // images: Set Current Position, Set Color, Begin Image .. End Image
	for _, p := range planes {
		if len(p.Rows) == 0 {
			continue
		}
		w := len(p.Rows[0])
		on := func(c byte) bool { return c == '#' || c == 'X' || c == 'x' || c == '1' || c == '*' }
		top, bottom, left, right := -1, -1, w, -1
		for r, row := range p.Rows {
			if len(row) != w {
				return nil, 0, errors.New("go3270: image rows of different lengths")
			}
			for c := 0; c < w; c++ {
				if on(row[c]) {
					if top < 0 {
						top = r
					}
					bottom = r
					left, right = min2(left, c), max2(right, c)
				}
			}
		}
		if top < 0 {
			continue // nothing to draw
		}
		cw := right - left + 1
		if cw > 2040 || bottom-top+1 > 2040 {
			return nil, 0, errors.New("go3270: image too large")
		}
		bpr := (cw + 7) / 8
		// rows a band, each band an image of its own, under a segment
		band := (maxSegment - 2 - 24) / (2 + bpr)
		for r0 := top; r0 <= bottom; r0 += band {
			r1 := min2(r0+band-1, bottom)
			var b bytes.Buffer
			px, py := x+left, y-r0
			b.Write([]byte{0x21, 4, byte(px >> 8), byte(px), byte(py >> 8), byte(py)}) // Set Current Position
			b.Write([]byte{0x0a, gocaColor(p.Color)})                                  // Set Color
			h := r1 - r0 + 1
			b.Write([]byte{0x91, 6, 0, 0, byte(cw >> 8), byte(cw), byte(h >> 8), byte(h)}) // Begin Image
			for _, row := range p.Rows[r0 : r1+1] {
				line := make([]byte, bpr)
				for c := 0; c < cw; c++ {
					if on(row[left+c]) {
						line[c/8] |= 0x80 >> (c % 8)
					}
				}
				b.Write([]byte{0x92, byte(bpr)}) // Image Data
				b.Write(line)
			}
			b.Write([]byte{0x93, 2, 0, 0}) // End Image
			parts = append(parts, b.Bytes())
		}
	}
	// into segments: each its Set Mix and as many images as fit
	var segs [][]byte
	var seg []byte
	for _, part := range parts {
		if len(seg) > 0 && len(seg)+len(part) > maxSegment {
			segs = append(segs, seg)
			seg = nil
		}
		if len(seg) == 0 {
			seg = []byte{0x0c, byte(mix)} // Set Mix
		}
		seg = append(seg, part...)
	}
	if len(seg) > 0 {
		segs = append(segs, seg)
	}
	if first+len(segs) > 1000 {
		return nil, 0, errors.New("go3270: too many segments")
	}
	var out [][]byte
	for i, seg := range segs {
		n := first + i
		sf := []byte{0x0f, 0x10, 0x00, 0xc0, 0x00} // Object Picture, PID 0, first and last, immediate
		// Begin Segment: name "B001".. (EBCDIC), visible, no highlight, new segment
		sf = append(sf, 0x70, 12, 0xc2, 0xf0+byte(n/100), 0xf0+byte(n/10%10), 0xf0+byte(n%10), 0x74, 0x68,
			byte(len(seg)>>8), byte(len(seg)), 0, 0, 0, 0)
		out = append(out, append(sf, seg...))
	}
	return out, first + len(segs), nil
}

// min2 is the smaller of a and b.
func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max2 is the larger of a and b.
func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}
