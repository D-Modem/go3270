// This file is part of https://github.com/racingmars/go3270/
// Copyright 2026 by the go3270-sec contributors, licensed under the MIT
// license. See LICENSE in the project root for license information.

package go3270

import (
	"bytes"
	"net"
	"testing"
	"time"
)

// a Character Sets query reply as PCOMM-style terminals send it: GE,
// Load PS (extended), formats 1 and 3, 9x16 slots, the base set, the GE
// set and two loadable sets
var charsetReply = []byte{0x81, 0x85,
	0xb2,                   // ALT, LOADABLE, EXT, GF
	0x00,                   // slot size match required
	9, 16,                  // SDW, SDH
	0x50, 0x00, 0x00, 0x00, // FORM: types 1 and 3
	7,                                  // DL
	0x00, 0x00, 0x00, 0x02, 0xb9, 0x04, 0x17, // base: CGCSGID 697/1047
	0x01, 0x00, 0xf1, 0x03, 0xc3, 0x01, 0x36, // GE set
	0x02, 0x80, 0xff, 0x00, 0x00, 0x00, 0x00, // loadable RWS 2
	0x03, 0x80, 0xff, 0x00, 0x00, 0x00, 0x00, // loadable RWS 3
}

func TestSymbolSets(t *testing.T) {
	s := getSymbolSets(charsetReply)
	if s == nil || !s.GraphicEscape || !s.Loadable || !s.Extended || !s.SizeMatch {
		t.Fatalf("flags: %+v", s)
	}
	if s.SlotWidth != 9 || s.SlotHeight != 16 {
		t.Errorf("slot %dx%d", s.SlotWidth, s.SlotHeight)
	}
	if !s.SupportsFormat(1) || s.SupportsFormat(2) || !s.SupportsFormat(3) {
		t.Errorf("formats %08x", s.Formats)
	}
	if len(s.Sets) != 4 || s.Sets[0].CGCSGID != 0x02b90417 || s.Sets[1].LCID != 0xf1 {
		t.Errorf("sets %+v", s.Sets)
	}
	if l := s.LoadableSets(); len(l) != 2 || l[0].RWS != 2 || l[1].RWS != 3 {
		t.Errorf("loadable %+v", l)
	}
	// short and odd replies do not panic
	for i := 0; i < len(charsetReply); i++ {
		getSymbolSets(charsetReply[:i])
	}
	if getSymbolSets([]byte{0x81, 0x85, 0, 0, 0, 0, 0, 0, 0, 0, 0}) == nil {
		t.Error("a reply without descriptors")
	}
}

func TestLoadPSFormat1(t *testing.T) {
	sym := Symbol{
		"#########",
		"#.......#",
	}
	sf, err := LoadPS{RWS: 2, LCID: 0x41, First: 0x41, Format: 1, Symbols: []Symbol{sym}}.StructuredField()
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x06, 0x01, 0x41, 0x41, 0x02,
		0xc0, 0x00, // column 0: rows 0 and 1
		0xff, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if !bytes.Equal(sf, want) {
		t.Errorf("Load PS\n%x\nwant\n%x", sf, want)
	}
}

func TestLoadPSFormat3(t *testing.T) {
	sym := Symbol{"#.......", "........", "#######."}
	sf, err := LoadPS{RWS: 2, LCID: 0x42, First: 0x50, Format: 3, Width: 8, Height: 3,
		Extended: true, Clear: true, Symbols: []Symbol{sym, sym}}.StructuredField()
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x06, 0xc3, 0x42, 0x50, 0x02, 4, 0, 8, 3,
		0x80, 0x00, 0xfe, 0x80, 0x00, 0xfe}
	if !bytes.Equal(sf, want) {
		t.Errorf("Load PS\n%x\nwant\n%x", sf, want)
	}
	// 9 wide: the rows' bits run on, each symbol padded to a byte
	sf, _ = LoadPS{RWS: 2, LCID: 0x42, First: 0x50, Format: 3, Width: 9, Height: 2,
		Symbols: []Symbol{{"#........", "........#"}}}.StructuredField()
	if !bytes.Equal(sf[5:], []byte{0x80, 0x00, 0x40, 0x00}[:3]) {
		t.Errorf("9 wide: %x", sf[5:])
	}
}

func TestLoadPSErrors(t *testing.T) {
	for _, l := range []LoadPS{
		{LCID: 0x30, First: 0x41, Format: 1},
		{LCID: 0x41, First: 0x40, Format: 1},
		{LCID: 0x41, First: 0xfe, Format: 1, Symbols: []Symbol{{}, {}}},
		{LCID: 0x41, First: 0x41, Format: 2},
		{LCID: 0x41, First: 0x41, Format: 3},
	} {
		if _, err := l.StructuredField(); err == nil {
			t.Errorf("took %+v", l)
		}
	}
}

func TestWriteStructuredFields(t *testing.T) {
	srv, cli := net.Pipe()
	go func() {
		WriteStructuredFields(srv, []byte{0x06, 0xff, 0x41})
		srv.Close()
	}()
	var got bytes.Buffer
	got.ReadFrom(cli)
	// WSF, length 5 (unescaped), the field with 0xFF doubled, IAC EOR
	want := []byte{0xf3, 0x00, 0x05, 0x06, 0xff, 0xff, 0x41, 0xff, 0xef}
	if !bytes.Equal(got.Bytes(), want) {
		t.Errorf("WSF %x, want %x", got.Bytes(), want)
	}
}

func TestCharSetField(t *testing.T) {
	f := buildField(Field{CharSet: 0x41})
	want := []byte{0x29, 2, 0xc0, sfAttribute(false, false, false, false, false), 0x43, 0x41}
	if !bytes.Equal(f, want) {
		t.Errorf("SFE %x, want %x", f, want)
	}
}

func TestImagePicture(t *testing.T) {
	sf, err := ImagePicture(-60, 60, Bitmap{Color: Blue, Rows: []string{"#.......#", "........."}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(sf, []byte{0x0f, 0x10, 0x00, 0xc0, 0x00, 0x70, 12}) {
		t.Errorf("header %x", sf[:7])
	}
	seg := sf[5+14:]
	if l := int(sf[5+8])<<8 | int(sf[5+9]); l != len(seg) {
		t.Errorf("segment length %d, segment is %d", l, len(seg))
	}
	want := []byte{0x0c, 4, 0x21, 4, 0xff, 0xc4, 0x00, 0x3c, 0x0a, 1,
		0x91, 6, 0, 0, 0, 9, 0, 2, 0x92, 2, 0x80, 0x80, 0x92, 2, 0, 0, 0x93, 2, 0, 0}
	if !bytes.Equal(seg, want) {
		t.Errorf("segment\n%x\nwant\n%x", seg, want)
	}
	if _, err := ImagePicture(0, 0, Bitmap{Rows: []string{"##", "#"}}); err == nil {
		t.Error("took ragged rows")
	}
}

// a client whose query reply has a length too short to be one used to
// crash the server (slice of negative length, index out of range)
func TestShortQueryReply(t *testing.T) {
	for _, l := range []byte{0, 1, 2, 3} {
		srv, cli := net.Pipe()
		done := make(chan error, 1)
		go func() {
			_, err := makeDeviceInfo(srv, "IBM-3278-2-E")
			done <- err
		}()
		go func() {
			buf := make([]byte, 64)
			cli.Read(buf) // EWA
			cli.Read(buf) // the query
			cli.Write([]byte{0x88, 0x00, l, 0x81, 0x81, 0xff, 0xef})
		}()
		select {
		case err := <-done:
			if err == nil {
				t.Errorf("length %d taken", l)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("length %d: no answer", l)
		}
		srv.Close()
		cli.Close()
	}
}

func TestDeviceInfoReplies(t *testing.T) {
	srv, cli := net.Pipe()
	done := make(chan DevInfo, 1)
	go func() {
		d, err := makeDeviceInfo(srv, "IBM-3278-2-E")
		if err != nil {
			t.Error(err)
		}
		done <- d
	}()
	usable := []byte{0x81, 0x81, 0x01, 0x00, 0, 80, 0, 24, 0x01, 0, 1, 0, 0x50, 0, 1, 0, 0x50, 9, 16}
	segment := []byte{0x81, 0xb0, 0x00}
	go func() {
		buf := make([]byte, 64)
		cli.Read(buf)
		cli.Read(buf)
		var r []byte
		r = append(r, 0x88)
		for _, q := range [][]byte{usable, charsetReply, segment} {
			r = append(r, byte((len(q)+2)>>8), byte(len(q)+2))
			r = append(r, q...)
		}
		cli.Write(append(telnetEscape(r), 0xff, 0xef))
	}()
	d := <-done
	if w, h := d.CellSize(); w != 9 || h != 16 {
		t.Errorf("cell %dx%d", w, h)
	}
	if !d.SupportsVectorGraphics() || d.SymbolSets() == nil || !d.SymbolSets().Loadable {
		t.Errorf("vector %v, symbols %+v", d.SupportsVectorGraphics(), d.SymbolSets())
	}
	if codes := d.QueryCodes(); !bytes.Equal(codes, []byte{0x81, 0x85, 0xb0}) {
		t.Errorf("codes %x", codes)
	}
	if r := d.QueryReply(0xb0); !bytes.Equal(r, []byte{0xb0, 0x00}) {
		t.Errorf("segment reply %x", r)
	}
	srv.Close()
	cli.Close()
}
