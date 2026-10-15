// Command gfxtest is a TN3270 server for debugging 3270 graphics on an
// emulator: it shows what the terminal reported (its query replies), draws
// vector bitmaps (Object Picture structured fields, GDDM-style image
// orders) in several ways, sends structured fields typed in hex, and logs
// every byte each way, the graphics orders and the terminal's structured
// field replies decoded.
//
//	go run ./cmd/gfxtest -addr :3270 -image photo.png -log gfxtest.log
//
// Connect an emulator with host graphics on (IBM PCOMM: Session
// Parameters > Graphics), to the address, and choose a test.
package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/racingmars/go3270"
)

var (
	imageFile = flag.String("image", "", "a PNG, JPEG or GIF for the image test")
	logFile   = flag.String("log", "", "the trace's file (default: stderr)")
	verbose   = flag.Bool("v", false, "hex dump every record (not only graphics and replies)")
	preview   = flag.String("preview", "", "a directory to write each picture sent to, as a PNG: what it should look like")
)

var logger = log.New(os.Stderr, "", log.Ltime|log.Lmicroseconds)

func main() {
	addr := flag.String("addr", ":3270", "the address to listen on")
	flag.Parse()
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			log.Fatal(err)
		}
		logger.SetOutput(io.MultiWriter(os.Stderr, f))
	}
	if *imageFile != "" {
		if _, err := loadImage(*imageFile); err != nil {
			log.Fatalf("-image: %v", err)
		}
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	logger.Printf("gfxtest listening on %s", ln.Addr())
	n := 0
	for {
		c, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		n++
		go session(&traceConn{Conn: c, id: n})
	}
}

// --- the session

type settings struct {
	origin string // C: 0,0 the middle, y up; B: bottom left, y up; T: top left, y down
	row    int    // the picture's top left: a character cell
	col    int
	size   string // S, M, L
	mix    string // X: exclusive or (planes make 8 colors), O: overpaint
	order  string // A: the graphics after the screen; B: before it (then the screen without erase)
	erase  bool   // Erase/Reset before the test
}

type sess struct {
	c        *traceConn
	dev      go3270.DevInfo
	rows     int
	cols     int
	cellW    int
	cellH    int
	set      settings
	msg      string
	lastTest string
	hex      string // the hex field, kept
	shots    int    // pictures previewed
}

func session(c *traceConn) {
	defer c.Close()
	logger.Printf("#%d connected from %s", c.id, c.RemoteAddr())
	dev, err := go3270.NegotiateTelnet(c)
	if err != nil {
		logger.Printf("#%d negotiation: %v", c.id, err)
		return
	}
	s := &sess{c: c, dev: dev, set: settings{origin: "C", row: 4, col: 2, size: "M", mix: "X", order: "A"}}
	s.rows, s.cols = dev.AltDimensions()
	if s.rows == 0 {
		s.rows, s.cols = 24, 80
	}
	s.cellW, s.cellH = dev.CellSize()
	if s.cellW == 0 || s.cellH == 0 {
		s.cellW, s.cellH = 9, 16
	}
	logger.Printf("#%d terminal %s, %dx%d, cell %dx%d, GE %v, vector graphics %v, query replies %s",
		c.id, dev.TerminalType(), s.cols, s.rows, s.cellW, s.cellH, dev.SupportsGE(), dev.SupportsVectorGraphics(),
		qcodes(dev))
	for _, q := range dev.QueryCodes() {
		logger.Printf("#%d query reply %02X %s: %s", c.id, q, qname(q), hex.EncodeToString(dev.QueryReply(q)))
	}
	for {
		if err := s.menu(); err != nil {
			logger.Printf("#%d %v", c.id, err)
			return
		}
	}
}

func (s *sess) opts() go3270.ScreenOpts {
	return go3270.ScreenOpts{AltScreen: s.dev, Codepage: s.dev.Codepage()}
}

func text(row, col int, t string, color go3270.Color) go3270.Field {
	return go3270.Field{Row: row, Col: col, Content: t, Color: color}
}

func input(row, col, width int, name string) []go3270.Field {
	return []go3270.Field{
		{Row: row, Col: col, Write: true, Name: name, Color: go3270.Green, Highlighting: go3270.Underscore},
		{Row: row, Col: col + width + 1, Autoskip: true},
	}
}

func (s *sess) menu() error {
	d := s.dev
	ss := d.SymbolSets()
	psets := "none"
	if ss != nil {
		var l []string
		for _, set := range ss.LoadableSets() {
			t := ""
			if set.Triple {
				t = " triple"
			}
			l = append(l, fmt.Sprintf("%dx%d%s", set.SlotWidth, set.SlotHeight, t))
		}
		if len(l) > 0 {
			psets = fmt.Sprintf("%d loadable (%s)", len(l), strings.Join(l, ", "))
		}
	}
	scr := go3270.Screen{
		text(0, 1, "gfxtest: 3270 vector bitmaps", go3270.Turquoise),
		text(1, 1, fmt.Sprintf("Terminal %s  %dx%d  cell %dx%d pels  GE %v", d.TerminalType(), s.cols, s.rows,
			s.cellW, s.cellH, d.SupportsGE()), go3270.White),
		text(2, 1, fmt.Sprintf("Vector graphics (Segment reply B0): %v   Programmed symbols: %s",
			d.SupportsVectorGraphics(), psets), go3270.White),
		text(3, 1, "Query replies: "+trunc(qcodes(d), s.cols-17), go3270.Blue),
		text(5, 1, "1 Test pattern, one plane    2 Color bars, 3 planes    3 Color bars, 7 bitmaps", go3270.Green),
		text(6, 1, "4 Corner markers (origin)    5 Image file", go3270.Green),
		text(7, 1, "6 Erase/Reset (clears graphics)   7 Send the hex below   8 Query replies, hex", go3270.Green),
		text(9, 1, "Origin", go3270.Turquoise), text(9, 26, "C middle, y up; B bottom left; T top left, y down", go3270.Blue),
		text(10, 1, "At row, col", go3270.Turquoise), text(10, 30, "the picture's top left, in character cells", go3270.Blue),
		text(11, 1, "Size", go3270.Turquoise), text(11, 26, "S small, M medium, L to the screen's edge", go3270.Blue),
		text(12, 1, "Mix", go3270.Turquoise), text(12, 26, "X exclusive or (planes make 8 colors), O overpaint", go3270.Blue),
		text(13, 1, "Graphics", go3270.Turquoise), text(13, 26, "A after the screen, B before it (then no erase)", go3270.Blue),
		text(14, 1, "Erase/Reset first", go3270.Turquoise), text(14, 26, "Y or N", go3270.Blue),
		text(16, 1, "Hex: structured fields from the SFID on, without length; several split by /", go3270.Turquoise),
		text(s.rows-3, 1, "Choice ===>", go3270.White),
		text(s.rows-1, 1, "Enter=run  PF3=quit   Every byte both ways is in the log", go3270.Blue),
	}
	if s.msg != "" {
		scr = append(scr, text(s.rows-2, 1, trunc(s.msg, s.cols-2), go3270.Yellow))
	}
	scr = append(scr, input(s.rows-3, 13, 2, "choice")...)
	scr = append(scr, input(9, 20, 1, "origin")...)
	scr = append(scr, input(10, 20, 2, "row")...)
	scr = append(scr, input(10, 24, 3, "col")...)
	scr = append(scr, input(11, 20, 1, "size")...)
	scr = append(scr, input(12, 20, 1, "mix")...)
	scr = append(scr, input(13, 20, 1, "order")...)
	scr = append(scr, input(14, 20, 1, "erase")...)
	// the hex: three rows, one field
	scr = append(scr, go3270.Field{Row: 17, Col: 0, Write: true, Name: "hex", Color: go3270.Green,
		Highlighting: go3270.Underscore}, go3270.Field{Row: 20, Col: 0, Autoskip: true})
	vals := map[string]string{"origin": s.set.origin, "row": strconv.Itoa(s.set.row), "col": strconv.Itoa(s.set.col),
		"size": s.set.size, "mix": s.set.mix, "order": s.set.order, "erase": yn(s.set.erase), "choice": s.lastTest, "hex": s.hex}
	o := s.opts()
	o.CursorRow, o.CursorCol = s.rows-3, 14
	resp, err := go3270.ShowScreenOpts(scr, vals, s.c, o)
	if err != nil {
		return err
	}
	s.msg = ""
	if resp.AID == go3270.AIDPF3 {
		return io.EOF
	}
	v := func(k string) string { return strings.ToUpper(strings.TrimSpace(strings.Trim(resp.Values[k], "\x00"))) }
	if o := v("origin"); strings.Contains("CBT", o) && o != "" {
		s.set.origin = o
	}
	if n, err := strconv.Atoi(v("row")); err == nil && n >= 0 && n < s.rows {
		s.set.row = n
	}
	if n, err := strconv.Atoi(v("col")); err == nil && n >= 0 && n < s.cols {
		s.set.col = n
	}
	if x := v("size"); x == "S" || x == "M" || x == "L" {
		s.set.size = x
	}
	if x := v("mix"); x == "X" || x == "O" {
		s.set.mix = x
	}
	if x := v("order"); x == "A" || x == "B" {
		s.set.order = x
	}
	s.set.erase = v("erase") == "Y"
	choice := v("choice")
	s.lastTest = choice
	s.hex = strings.TrimRight(strings.Trim(resp.Values["hex"], "\x00"), " ")
	switch choice {
	case "1", "2", "3", "4", "5":
		return s.picture(choice)
	case "6":
		s.c.mark("Erase/Reset")
		if err := go3270.WriteStructuredFields(s.c, []byte{0x03, 0x00}); err != nil {
			return err
		}
		s.msg = "Erase/Reset sent (03 00): the graphics should be gone"
	case "7":
		return s.sendHex(s.hex)
	case "8":
		return s.replies()
	case "":
	default:
		s.msg = "No such test: " + choice
	}
	return nil
}

func yn(b bool) string {
	if b {
		return "Y"
	}
	return "N"
}

// picture runs a test: its screen (a caption, room below) and its
// picture, as the settings say; Enter goes back.
func (s *sess) picture(test string) error {
	w, h := s.sizePels()
	var planes []go3270.Bitmap
	var what string
	switch test {
	case "1":
		planes, what = []go3270.Bitmap{{Color: go3270.White, Rows: pattern(w, h)}}, "a frame, diagonals and a checkerboard, white"
	case "2":
		planes, what = bars(w, h), "8 bars, black blue red pink green turquoise yellow white: 3 planes (B R G)"
	case "3":
		planes, what = barsEach(w, h), "8 bars as 7 bitmaps, one a color (needs mix O to look right)"
	case "4":
		return s.corners()
	case "5":
		if *imageFile == "" {
			s.msg = "No -image given"
			return nil
		}
		img, err := loadImage(*imageFile)
		if err != nil {
			s.msg = err.Error()
			return nil
		}
		planes = dither(img, w, h)
		what = fmt.Sprintf("%s: %s, drawn %dx%d", filepath.Base(*imageFile), describe(*imageFile, img),
			len(planes[0].Rows[0]), len(planes[0].Rows))
		logger.Printf("#%d image %s: %s, drawn %dx%d", s.c.id, *imageFile, describe(*imageFile, img),
			len(planes[0].Rows[0]), len(planes[0].Rows))
	}
	x, y := s.position(s.set.row, s.set.col)
	return s.show(fmt.Sprintf("Test %s: %s", test, what), x, y, planes)
}

// sizePels is the picture's size for the size setting: within the screen.
func (s *sess) sizePels() (int, int) {
	maxW, maxH := (s.cols-s.set.col-1)*s.cellW, (s.rows-s.set.row-2)*s.cellH
	w, h := 64, 48
	switch s.set.size {
	case "M":
		w, h = 192, 128
	case "L":
		w, h = maxW, maxH
	}
	w, h = imin(w, maxW), imin(h, maxH)
	// three planes in one segment: under 64K
	for 3*((w+7)/8)*h > 60000 {
		w, h = w*9/10, h*9/10
	}
	return imax(w, 8), imax(h, 8)
}

// position is a character cell's top left in graphics coordinates, as the
// origin setting says.
func (s *sess) position(row, col int) (int, int) {
	switch s.set.origin {
	case "B":
		return col * s.cellW, (s.rows - row) * s.cellH
	case "T":
		return col * s.cellW, row * s.cellH
	}
	return col*s.cellW - s.cols*s.cellW/2, s.rows*s.cellH/2 - row*s.cellH
}

// show sends the test's screen and its picture, in the order the settings
// say, and waits for Enter (PF5 sends the picture again).
func (s *sess) show(caption string, x, y int, planes []go3270.Bitmap) error {
	sf, err := picture(x, y, s.set.mix, planes)
	if err != nil {
		s.msg = err.Error()
		return nil
	}
	if *preview != "" {
		s.shots++
		name := filepath.Join(*preview, fmt.Sprintf("gfxtest-%d-%d.png", s.c.id, s.shots))
		if err := writePreview(name, s.set.mix, planes); err != nil {
			logger.Printf("#%d preview: %v", s.c.id, err)
		} else {
			logger.Printf("#%d preview of the picture: %s", s.c.id, name)
		}
	}
	scr := go3270.Screen{
		text(0, 1, trunc(caption, s.cols-2), go3270.Turquoise),
		text(1, 1, fmt.Sprintf("at %d,%d (origin %s, mix %s, graphics %s the screen), %d bytes", x, y, s.set.origin,
			s.set.mix, map[string]string{"A": "after", "B": "before"}[s.set.order], len(sf)), go3270.Blue),
		text(s.rows-1, 1, "Enter=back  PF5=send the picture again  PF6=Erase/Reset", go3270.Blue),
	}
	scr = append(scr, input(s.rows-2, 1, 1, "x")...)
	send := func(any) error {
		if s.set.erase {
			s.c.mark("Erase/Reset")
			if err := go3270.WriteStructuredFields(s.c, []byte{0x03, 0x00}); err != nil {
				return err
			}
		}
		s.c.mark("Object Picture")
		return go3270.WriteStructuredFields(s.c, sf)
	}
	for {
		o := s.opts()
		o.CursorRow, o.CursorCol = s.rows-2, 2
		if s.set.order == "B" {
			if err := send(nil); err != nil {
				return err
			}
			o.NoClear = true // Write, not Erase/Write: the picture may stay
		} else {
			o.PostSendCallback = send
		}
		resp, err := go3270.ShowScreenOpts(scr, nil, s.c, o)
		if err != nil {
			return err
		}
		switch resp.AID {
		case go3270.AIDPF5:
			continue
		case go3270.AIDPF6:
			s.c.mark("Erase/Reset")
			go3270.WriteStructuredFields(s.c, []byte{0x03, 0x00})
			continue
		}
		return nil
	}
}

// corners: small squares at the screen's four corners and the middle, and
// one where row, col says, in one picture each, to see the coordinates.
func (s *sess) corners() error {
	sq := func(c go3270.Color) []go3270.Bitmap {
		rows := make([]string, s.cellH)
		for i := range rows {
			rows[i] = strings.Repeat("#", s.cellW)
		}
		return []go3270.Bitmap{{Color: c, Rows: rows}}
	}
	type mark struct {
		row, col int
		color    go3270.Color
		name     string
	}
	marks := []mark{{0, 0, go3270.Red, "top left (red)"}, {0, s.cols - 1, go3270.Green, "top right (green)"},
		{s.rows - 1, 0, go3270.Blue, "bottom left (blue)"}, {s.rows - 1, s.cols - 1, go3270.Yellow, "bottom right (yellow)"},
		{s.rows / 2, s.cols / 2, go3270.White, "middle (white)"}, {s.set.row, s.set.col, go3270.Pink, "row,col (pink)"}}
	var sfs [][]byte
	var names []string
	for _, m := range marks {
		x, y := s.position(m.row, m.col)
		sf, err := picture(x, y, s.set.mix, sq(m.color))
		if err != nil {
			return err
		}
		sfs = append(sfs, sf)
		names = append(names, fmt.Sprintf("%s %d,%d", m.name, x, y))
	}
	scr := go3270.Screen{
		text(2, 1, "Test 4: a cell-sized square in each corner and the middle; are they there?", go3270.Turquoise),
		text(3, 1, "origin "+s.set.origin+": "+trunc(strings.Join(names[:3], "; "), s.cols-12), go3270.Blue),
		text(4, 1, trunc(strings.Join(names[3:], "; "), s.cols-2), go3270.Blue),
		text(s.rows-2, 1, "Enter=back", go3270.Blue),
	}
	o := s.opts()
	o.PostSendCallback = func(any) error {
		s.c.mark("Object Pictures: corners")
		return go3270.WriteStructuredFields(s.c, sfs...)
	}
	_, err := go3270.ShowScreenOpts(scr, nil, s.c, o)
	return err
}

// sendHex sends structured fields typed in hex (split by /), each from its
// SFID on.
func (s *sess) sendHex(text string) error {
	text = strings.Trim(text, "\x00")
	var sfs [][]byte
	for _, part := range strings.Split(text, "/") {
		h := strings.NewReplacer(" ", "", "\n", "", ",", "").Replace(part)
		if h == "" {
			continue
		}
		b, err := hex.DecodeString(h)
		if err != nil {
			s.msg = "Not hex: " + err.Error()
			return nil
		}
		sfs = append(sfs, b)
	}
	if len(sfs) == 0 {
		s.msg = "No hex to send"
		return nil
	}
	s.c.mark("structured fields from the screen")
	if err := go3270.WriteStructuredFields(s.c, sfs...); err != nil {
		return err
	}
	s.msg = fmt.Sprintf("Sent %d structured fields: see the log for what came back", len(sfs))
	return nil
}

// replies shows the query replies in hex, a page at a time.
func (s *sess) replies() error {
	var lines []string
	for _, q := range s.dev.QueryCodes() {
		h := hex.EncodeToString(s.dev.QueryReply(q))
		lines = append(lines, fmt.Sprintf("%02X %s:", q, qname(q)))
		for len(h) > 0 {
			n := imin(len(h), s.cols-6)
			lines = append(lines, "    "+h[:n])
			h = h[n:]
		}
	}
	per := s.rows - 3
	for top := 0; top < len(lines); top += per {
		scr := go3270.Screen{text(0, 1, "Query replies (from the QCODE on)", go3270.Turquoise),
			text(s.rows-1, 1, "Enter=next page  PF3=back", go3270.Blue)}
		for i := 0; i < per && top+i < len(lines); i++ {
			scr = append(scr, text(1+i, 1, lines[top+i], go3270.Green))
		}
		resp, err := go3270.ShowScreenOpts(scr, nil, s.c, s.opts())
		if err != nil || resp.AID == go3270.AIDPF3 {
			return err
		}
	}
	return nil
}

// --- pictures

// picture is an Object Picture of the planes at x, y: go3270's, with
// overpaint in place of exclusive or when mix says O.
func picture(x, y int, mix string, planes []go3270.Bitmap) ([]byte, error) {
	sf, err := go3270.ImagePicture(x, y, planes...)
	if err != nil {
		return nil, err
	}
	if mix == "O" {
		// the Set Mix order (0C 04) after Begin Segment's 14 bytes
		if i := 5 + 14; len(sf) > i+1 && sf[i] == 0x0c {
			sf[i+1] = 0x02 // overpaint
		}
	}
	return sf, nil
}

func blank(w, h int) [][]byte {
	rows := make([][]byte, h)
	for i := range rows {
		rows[i] = bytes.Repeat([]byte{'.'}, w)
	}
	return rows
}

func strs(rows [][]byte) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = string(r)
	}
	return out
}

// pattern is a frame, both diagonals and a checkerboard of 8-pel squares in
// the middle third.
func pattern(w, h int) []string {
	r := blank(w, h)
	for x := 0; x < w; x++ {
		r[0][x], r[h-1][x] = '#', '#'
	}
	for y := 0; y < h; y++ {
		r[y][0], r[y][w-1] = '#', '#'
		x := y * (w - 1) / imax(h-1, 1)
		r[y][x], r[y][w-1-x] = '#', '#'
	}
	for y := h / 3; y < 2*h/3; y++ {
		for x := w / 3; x < 2*w/3; x++ {
			if (x/8+y/8)%2 == 0 {
				r[y][x] = '#'
			}
		}
	}
	return strs(r)
}

// colors in their graphics index order: black, blue 1, red 2, pink 3,
// green 4, turquoise 5, yellow 6, white 7
var barColors = []go3270.Color{go3270.Blue, go3270.Red, go3270.Pink, go3270.Green, go3270.Turquoise, go3270.Yellow, go3270.White}

// bars: 8 vertical bars, colors 0 to 7, as three planes (blue 1, red 2,
// green 4) that exclusive or makes the rest of.
func bars(w, h int) []go3270.Bitmap {
	planes := [3][][]byte{blank(w, h), blank(w, h), blank(w, h)}
	for x := 0; x < w; x++ {
		c := x * 8 / w
		for bit := 0; bit < 3; bit++ {
			if c&(1<<bit) != 0 {
				for y := 0; y < h; y++ {
					planes[bit][y][x] = '#'
				}
			}
		}
	}
	return []go3270.Bitmap{{Color: go3270.Blue, Rows: strs(planes[0])}, {Color: go3270.Red, Rows: strs(planes[1])},
		{Color: go3270.Green, Rows: strs(planes[2])}}
}

// barsEach: the same bars, a bitmap for each color.
func barsEach(w, h int) []go3270.Bitmap {
	var out []go3270.Bitmap
	for c := 1; c <= 7; c++ {
		r := blank(w, h)
		for x := 0; x < w; x++ {
			if x*8/w == c {
				for y := 0; y < h; y++ {
					r[y][x] = '#'
				}
			}
		}
		out = append(out, go3270.Bitmap{Color: barColors[c-1], Rows: strs(r)})
	}
	return out
}

// describe is what the image file decoded as: its size and format, and
// how long the file is (to tell one file from another).
func describe(path string, img image.Image) string {
	format, size := "?", int64(0)
	if f, err := os.Open(path); err == nil {
		_, format, _ = image.DecodeConfig(f)
		if fi, err := f.Stat(); err == nil {
			size = fi.Size()
		}
		f.Close()
	}
	b := img.Bounds()
	return fmt.Sprintf("%dx%d %s (%s, %d bytes)", b.Dx(), b.Dy(), format, strings.TrimPrefix(fmt.Sprintf("%T", img), "*image."), size)
}

// gocaRGB is the 3270 graphics colors 0 to 7, as a preview shows them.
var gocaRGB = []color.RGBA{{0, 0, 0, 255}, {0, 0, 255, 255}, {255, 0, 0, 255}, {255, 0, 255, 255},
	{0, 255, 0, 255}, {0, 255, 255, 255}, {255, 255, 0, 255}, {255, 255, 255, 255}}

// writePreview draws the planes as the terminal should (each pel's
// colors exclusive-or'd, or the last on top for mix O) to a PNG, each pel
// 2x2.
func writePreview(name, mix string, planes []go3270.Bitmap) error {
	w, h := 0, 0
	for _, p := range planes {
		h = imax(h, len(p.Rows))
		for _, r := range p.Rows {
			w = imax(w, len(r))
		}
	}
	out := image.NewRGBA(image.Rect(0, 0, 2*w, 2*h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := 0
			for _, p := range planes {
				if y >= len(p.Rows) || x >= len(p.Rows[y]) || !strings.ContainsRune("#Xx1*", rune(p.Rows[y][x])) {
					continue
				}
				if pc := int(p.Color) & 7; mix == "O" {
					c = pc
				} else {
					c ^= pc
				}
			}
			for d := 0; d < 4; d++ {
				out.SetRGBA(2*x+d%2, 2*y+d/2, gocaRGB[c])
			}
		}
	}
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	if err := png.Encode(f, out); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// bayer is a 4x4 ordered dither's thresholds, 0 to 15.
var bayer = [4][4]int{{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5}}

// dither scales img to w x h (keeping its shape) and dithers each of red,
// green and blue to on or off: the three planes.
func dither(img image.Image, w, h int) []go3270.Bitmap {
	b := img.Bounds()
	if b.Dx()*h > b.Dy()*w {
		h = imax(w*b.Dy()/b.Dx(), 1)
	} else {
		w = imax(h*b.Dx()/b.Dy(), 1)
	}
	planes := [3][][]byte{blank(w, h), blank(w, h), blank(w, h)} // blue, red, green
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x*b.Dx()/w, b.Min.Y+y*b.Dy()/h).RGBA()
			t := uint32(bayer[y%4][x%4]*4096 + 2048)
			if bl > t {
				planes[0][y][x] = '#'
			}
			if r > t {
				planes[1][y][x] = '#'
			}
			if g > t {
				planes[2][y][x] = '#'
			}
		}
	}
	return []go3270.Bitmap{{Color: go3270.Blue, Rows: strs(planes[0])}, {Color: go3270.Red, Rows: strs(planes[1])},
		{Color: go3270.Green, Rows: strs(planes[2])}}
}

// --- names

var qcodeNames = map[byte]string{
	0x80: "Summary", 0x81: "Usable Area", 0x82: "Image", 0x83: "Text Partitions", 0x84: "Alphanumeric Partitions",
	0x85: "Character Sets", 0x86: "Color", 0x87: "Highlight", 0x88: "Reply Modes", 0x8A: "Field Validation",
	0x8B: "MSR Control", 0x8C: "Field Outlining", 0x8E: "Partition Characteristics", 0x8F: "OEM Auxiliary Device",
	0x90: "Format Presentation", 0x91: "DBCS-Asia", 0x92: "Save/Restore Format", 0x94: "Format Storage Auxiliary Device",
	0x95: "Distributed Data Management", 0x96: "Storage Pools", 0x97: "Document Interchange Architecture",
	0x98: "Data Chaining", 0x99: "Auxiliary Device", 0x9A: "3270 IPDS", 0x9C: "Product Defined Data Stream",
	0x9D: "Anomaly Implementation", 0x9E: "IBM Auxiliary Device", 0x9F: "Begin/End of File", 0xA0: "Device Characteristics",
	0xA1: "RPQ Names", 0xA2: "Data Streams", 0xA6: "Implicit Partition", 0xA7: "Paper Feed Techniques",
	0xA8: "Transparency", 0xA9: "Settable Printer Characteristics", 0xAA: "IOCA Auxiliary Device",
	0xAB: "Cooperative Processing Requestor", 0xB0: "Segment", 0xB1: "Procedure", 0xB2: "Line Type",
	0xB3: "Port", 0xB4: "Graphic Color", 0xB5: "Extended Drawing Routine", 0xB6: "Graphic Symbol Sets",
	0xFF: "Null",
}

func qname(q byte) string {
	if n, ok := qcodeNames[q]; ok {
		return n
	}
	return "?"
}

func qcodes(d go3270.DevInfo) string {
	var l []string
	for _, q := range d.QueryCodes() {
		l = append(l, fmt.Sprintf("%02X", q))
	}
	return strings.Join(l, " ")
}

func trunc(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) > n {
		return s[:n]
	}
	return s
}

// --- the trace

// traceConn logs every record each way: commands and structured fields by
// name, graphics orders decoded, the rest in hex with -v.
type traceConn struct {
	net.Conn
	id int

	mu   sync.Mutex
	in   []byte // a record coming in, so far
	note string // what the next record out is (for the log)
}

func (c *traceConn) mark(note string) {
	c.mu.Lock()
	c.note = note
	c.mu.Unlock()
}

func (c *traceConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	note := c.note
	c.note = ""
	c.mu.Unlock()
	for _, rec := range records(p) {
		logger.Printf("#%d >>> %s", c.id, describeOut(rec, note))
	}
	return c.Conn.Write(p)
}

func (c *traceConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.mu.Lock()
		c.in = append(c.in, p[:n]...)
		var done [][]byte
		for {
			i := eor(c.in)
			if i < 0 {
				break
			}
			done = append(done, unescape(c.in[:i]))
			c.in = c.in[i+2:]
		}
		c.mu.Unlock()
		for _, rec := range done {
			logger.Printf("#%d <<< %s", c.id, describeIn(rec))
		}
	}
	return n, err
}

// eor is where IAC EOR is in b (not an escaped IAC's), -1 if not.
func eor(b []byte) int {
	for i := 0; i+1 < len(b); i++ {
		if b[i] == 0xff {
			if b[i+1] == 0xef {
				return i
			}
			i++ // IAC IAC, or another command
		}
	}
	return -1
}

func unescape(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte{0xff, 0xff}, []byte{0xff})
}

// records splits what is written into 3270 records (telnet negotiation is
// left as one piece).
func records(p []byte) [][]byte {
	var out [][]byte
	for {
		i := eor(p)
		if i < 0 {
			if len(p) > 0 {
				out = append(out, p)
			}
			return out
		}
		out = append(out, unescape(p[:i]))
		p = p[i+2:]
	}
}

var commands = map[byte]string{0xf1: "Write", 0xf5: "Erase/Write", 0x7e: "Erase/Write Alternate", 0x6f: "Erase All Unprotected",
	0xf3: "Write Structured Field", 0xf2: "Read Buffer", 0xf6: "Read Modified", 0x6e: "Read Modified All",
	0x01: "Write", 0x05: "Erase/Write", 0x0d: "Erase/Write Alternate", 0x11: "Write Structured Field"}

var sfNames = map[byte]string{0x00: "Read Partition", 0x01: "Read Partition (Query)", 0x02: "Set Reply Mode", 0x03: "Erase/Reset",
	0x06: "Load Programmed Symbols", 0x09: "Set Window Origin", 0x0b: "Set Partition Characteristics", 0x0c: "Create Partition",
	0x0d: "Destroy Partition", 0x0e: "Activate Partition", 0x40: "Outbound 3270DS", 0x80: "Outbound Text Header",
	0x81: "Query Reply", 0x82: "Inbound 3270DS"}

func describeOut(rec []byte, note string) string {
	if len(rec) == 0 {
		return "(empty)"
	}
	if rec[0] == 0xff {
		return "telnet " + hex.EncodeToString(rec)
	}
	name := commands[rec[0]]
	if name == "" {
		name = fmt.Sprintf("command %02X", rec[0])
	}
	if note != "" {
		name += " [" + note + "]"
	}
	s := fmt.Sprintf("%s, %d bytes", name, len(rec))
	if rec[0] == 0xf3 || rec[0] == 0x11 {
		s += "\n" + structuredFields(rec[1:], true)
	} else if *verbose {
		s += "\n" + indent(hex.Dump(rec))
	}
	return s
}

func describeIn(rec []byte) string {
	if len(rec) == 0 {
		return "(empty)"
	}
	if rec[0] == 0x88 {
		return fmt.Sprintf("Structured Field (inbound), %d bytes\n%s", len(rec), structuredFields(rec[1:], false))
	}
	s := fmt.Sprintf("AID %02X, %d bytes", rec[0], len(rec))
	if *verbose {
		s += "\n" + indent(hex.Dump(rec))
	}
	return s
}

// structuredFields lists structured fields (each from its length on),
// Object Pictures' orders decoded.
func structuredFields(b []byte, out bool) string {
	var lines []string
	for len(b) >= 3 {
		l := int(b[0])<<8 | int(b[1])
		if l == 0 {
			l = len(b) // to the end
		}
		if l < 3 || l > len(b) {
			lines = append(lines, fmt.Sprintf("  a bad length %d with %d bytes left: %s", l, len(b), hex.EncodeToString(b)))
			break
		}
		sf := b[2:l]
		b = b[l:]
		id := sf[0]
		switch {
		case id == 0x0f && len(sf) > 1:
			lines = append(lines, fmt.Sprintf("  SF 0F%02X %s, %d bytes", sf[1], sf0F(sf[1], out), l))
			if sf[1] == 0x10 && len(sf) >= 5 {
				lines = append(lines, fmt.Sprintf("    partition %02X, flags %02X (%s)", sf[2], sf[3], picFlags(sf[3])))
				lines = append(lines, goca(sf[5:])...)
			} else {
				lines = append(lines, indent(hex.Dump(sf)))
			}
		case !out && id == 0x81 && len(sf) > 1:
			lines = append(lines, fmt.Sprintf("  Query Reply %02X %s, %d bytes: %s", sf[1], qname(sf[1]), l, hex.EncodeToString(sf[1:])))
		default:
			name := sfNames[id]
			if name == "" {
				name = "?"
			}
			lines = append(lines, fmt.Sprintf("  SF %02X %s, %d bytes: %s", id, name, l, hex.EncodeToString(sf)))
		}
	}
	if len(b) > 0 {
		lines = append(lines, "  left over: "+hex.EncodeToString(b))
	}
	return strings.Join(lines, "\n")
}

func sf0F(id byte, out bool) string {
	names := map[byte]string{0x10: "Object Picture", 0x11: "Object Data", 0x1f: "Exception/Status (?)",
		0x21: "Exception/Status", 0x22: "Inbound Object Picture"}
	if n, ok := names[id]; ok {
		return n
	}
	return "?"
}

func picFlags(f byte) string {
	var l []string
	if f&0x80 != 0 {
		l = append(l, "first")
	}
	if f&0x40 != 0 {
		l = append(l, "last")
	}
	if len(l) == 0 {
		l = append(l, "middle")
	}
	return strings.Join(l, ", ")
}

// goca decodes the drawing orders (GOCA, GDDM's) inside an Object Picture.
func goca(b []byte) []string {
	var out []string
	for len(b) > 0 {
		op := b[0]
		switch op {
		case 0x00:
			out = append(out, "    00 No-op")
			b = b[1:]
			continue
		case 0x0a, 0x0c, 0x08, 0x18, 0x19, 0x26, 0x28, 0x29, 0x38, 0x39:
			// fixed: one byte of parameter
			if len(b) < 2 {
				out = append(out, "    "+hex.EncodeToString(b)+" (short)")
				return out
			}
			n := map[byte]string{0x0a: "Set Color", 0x0c: "Set Mix", 0x08: "Set Pattern Set", 0x18: "Set Line Type",
				0x19: "Set Line Width", 0x26: "Set Background Mix", 0x28: "Set Pattern Symbol",
				0x29: "Set Marker Symbol", 0x38: "Set Character Set", 0x39: "Set Marker Set"}[op]
			extra := ""
			if op == 0x0c || op == 0x26 {
				extra = map[byte]string{0: " (default)", 1: " (or)", 2: " (overpaint)", 3: " (?)", 4: " (exclusive or)", 5: " (leave alone)"}[b[1]]
			}
			out = append(out, fmt.Sprintf("    %02X %02X %s%s", op, b[1], n, extra))
			b = b[2:]
			continue
		}
		if len(b) < 2 || int(b[1])+2 > len(b) {
			out = append(out, fmt.Sprintf("    %s (an order cut short)", hex.EncodeToString(b)))
			return out
		}
		n := int(b[1])
		data := b[2 : 2+n]
		name := map[byte]string{0x70: "Begin Segment", 0x71: "End Segment", 0x21: "Set Current Position",
			0x91: "Begin Image", 0x92: "Image Data", 0x93: "End Image", 0xc1: "Line", 0x81: "Line at Current Position",
			0xc3: "Character String", 0x68: "Begin Area", 0x60: "End Area"}[op]
		if name == "" {
			name = "?"
		}
		detail := hex.EncodeToString(data)
		switch op {
		case 0x21:
			if n == 4 {
				detail = fmt.Sprintf("x %d, y %d", int16(uint16(data[0])<<8|uint16(data[1])), int16(uint16(data[2])<<8|uint16(data[3])))
			}
		case 0x91:
			if n == 6 {
				detail = fmt.Sprintf("format %02X%02X, %d x %d", data[0], data[1], int(data[2])<<8|int(data[3]), int(data[4])<<8|int(data[5]))
			}
		case 0x70:
			if n >= 12 {
				detail = fmt.Sprintf("name %s, flags %02X%02X, segment %d bytes, %s", hex.EncodeToString(data[0:4]), data[4], data[5],
					int(data[6])<<8|int(data[7]), hex.EncodeToString(data[8:]))
			}
		}
		if op == 0x92 {
			// many: one line for a run of them
			rows, bytes := 0, 0
			for len(b) >= 2 && b[0] == 0x92 && int(b[1])+2 <= len(b) {
				rows++
				bytes += int(b[1])
				b = b[2+int(b[1]):]
			}
			out = append(out, fmt.Sprintf("    92 Image Data x %d rows, %d bytes", rows, bytes))
			continue
		}
		out = append(out, fmt.Sprintf("    %02X %s: %s", op, name, detail))
		b = b[2+n:]
	}
	return out
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    ")
}

// keep the time import for the log's flags' sake

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
