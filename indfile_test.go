// This file is part of https://github.com/racingmars/go3270/
// Copyright 2026 by the go3270-sec contributors, licensed under the MIT
// license. See LICENSE in the project root for license information.

package go3270

import (
	"net"
	"strings"
	"testing"
	"time"
)

// TestParseIndFile: GET or PUT, the name, and the options after it (TSO) or
// after "(" (CMS); other commands are refused.
func TestParseIndFile(t *testing.T) {
	cases := []struct {
		in          string
		ok, get     bool
		name        string
		ascii, crlf bool
	}{
		{"IND$FILE PUT fttext ASCII CRLF", true, false, "fttext", true, true},
		{"IND$FILE PUT ft1 text (ASCII CRLF", true, false, "ft1 text", true, true},
		{"ind$file get readme.txt", true, true, "readme.txt", false, false},
		{"IND$FILE GET my file.zip (", true, true, "my file.zip", false, false},
		{"IND$FILE GET 'USER.DATA(MEM)' ASCII", true, true, "'USER.DATA(MEM)'", true, false},
		{"IND$FILE PUT USER.DATA(MEM) ASCII CRLF", true, false, "USER.DATA(MEM)", true, true},
		{"IND$FILE PUT 'X.LIB(A)' RECFM(V) LRECL(80)", true, false, "'X.LIB(A)'", false, false},
		{"IND$FILE PUT ft1 text a(ASCII CRLF", true, false, "ft1 text a", true, true},
		{"IND$FILE DELETE x", false, false, "", false, false},
		{"IND$FILE PUT", false, false, "", false, false},
		{"1", false, false, "", false, false},
	}
	for _, c := range cases {
		f, ok := ParseIndFile(c.in)
		if ok != c.ok {
			t.Errorf("%q: ok %v", c.in, ok)
			continue
		}
		if !ok {
			continue
		}
		if f.Get != c.get || f.Name != c.name || f.ASCII != c.ascii || f.CRLF != c.crlf {
			t.Errorf("%q: %+v", c.in, f)
		}
	}
	if f, _ := ParseIndFile("IND$FILE PUT 'X.LIB(A)' RECFM(V) LRECL(80) (APPEND)"); strings.Join(f.Options, " ") != "RECFM(V) LRECL(80) APPEND" || !f.Append {
		t.Errorf("options %q", f.Options)
	}
}

// TestDFTBufferSize: the smaller of the DDM reply's LIMIN and LIMOUT; 0 without
// the reply.
func TestDFTBufferSize(t *testing.T) {
	d := &deviceInfo{replies: map[byte][]byte{0x95: {0x95, 0, 0, 0x40, 0, 0x20, 0, 1, 1, 0}}}
	if n := DFTBufferSize(d); n != 0x2000 {
		t.Errorf("size %d", n)
	}
	if DFTBufferSize(&deviceInfo{}) != 0 {
		t.Error("DFT without the reply")
	}
}

// TestReadRecord: IAC IAC undone, telnet commands dropped, and nothing past IAC
// EOR read.
func TestReadRecord(t *testing.T) {
	srv, cli := net.Pipe()
	go func() {
		// IAC IAC, a DO, a subnegotiation, then IAC EOR; and the next
		// record's first byte, which must stay unread
		cli.Write([]byte{0x88, 0xff, 0xff, 0xff, 0xfd, 0x18, 0xff, 0xfa, 0x18, 0x01, 0xff, 0xf0, 0x01, 0xff, 0xef, 0x7d})
	}()
	rec, err := ReadRecord(srv, time.Second)
	if err != nil || string(rec) != "\x88\xff\x01" {
		t.Errorf("record %x, %v", rec, err)
	}
	b := make([]byte, 1)
	srv.Read(b)
	if b[0] != 0x7d {
		t.Errorf("next byte %x", b[0])
	}
}
