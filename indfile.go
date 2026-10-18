// This file is part of https://github.com/racingmars/go3270/
// Copyright 2026 by the go3270-sec contributors, licensed under the MIT
// license. See LICENSE in the project root for license information.

package go3270

// IND$FILE file transfer, the host's side, in DFT mode (Distributed
// Function Terminal: Data Transfer structured fields, ID 0xD0).
//
// The terminal's user starts a transfer in their emulator (x3270's
// Transfer, PCOMM's Send/Receive), which types "IND$FILE GET name ..." or
// "IND$FILE PUT name ..." into the field at the cursor and presses Enter.
// The application sees that as a field value (ParseIndFile), then runs
// the transfer: SendFileDFT for a GET (host to PC), ReceiveFileDFT for a
// PUT, and the closing message, DFTMessage.
//
// The byte layouts are a real TSO IND$FILE's (x3270's test trace
// s3270/Test/ft_dft.trc) and x3270's ft_dft.c, the PC's side.

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// IndFile is an IND$FILE command.
type IndFile struct {
	Get     bool     // GET: the host sends the file; PUT: the host receives it
	Name    string   // the host file name, as typed
	ASCII   bool     // ASCII (text) rather than binary
	CRLF    bool     // CR LF line ends
	Append  bool     // PUT appends to the host file
	Options []string // all the options, upper case
}

// ParseIndFile parses "IND$FILE GET|PUT name [(] options" (TSO puts the
// options after the name; VM/CMS after a "("), case aside.  A "(" within
// a name (a TSO member, 'USER.DATA(MEM)') stays in it.
func ParseIndFile(s string) (*IndFile, bool) {
	f := strings.Fields(s)
	if len(f) < 3 || !strings.EqualFold(f[0], "IND$FILE") {
		return nil, false
	}
	ind := &IndFile{}
	switch strings.ToUpper(f[1]) {
	case "GET":
		ind.Get = true
	case "PUT":
	default:
		return nil, false
	}
	var name, opts []string
	i := 2
	for ; i < len(f); i++ {
		tok := f[i]
		if strings.HasPrefix(tok, "(") {
			opts = append(opts, tok[1:]) // CMS: the options from here
			i++
			break
		}
		if len(name) > 0 && isIndOption(strings.ToUpper(tok)) {
			break // TSO: options after the name
		}
		// CMS written close up, "fm(ASCII": the "(" begins options when
		// one follows it (a TSO member, "DATA(MEM)", does not)
		if j := strings.IndexByte(tok, '('); j > 0 && !strings.HasPrefix(tok, "'") &&
			isIndOption(strings.ToUpper(strings.TrimSuffix(tok[j+1:], ")"))) {
			name = append(name, tok[:j])
			opts = append(opts, tok[j+1:])
			i++
			break
		}
		name = append(name, tok)
	}
	if len(name) == 0 {
		return nil, false
	}
	ind.Name = strings.Join(name, " ")
	for _, o := range append(opts, f[i:]...) {
		// "(ASCII", "CRLF)": the CMS parentheses off; "RECFM(V)" kept whole
		u := strings.ToUpper(strings.TrimPrefix(o, "("))
		if !strings.Contains(u, "(") {
			u = strings.TrimSuffix(u, ")")
		}
		if u == "" {
			continue
		}
		ind.Options = append(ind.Options, u)
		switch u {
		case "ASCII":
			ind.ASCII = true
		case "CRLF":
			ind.CRLF = true
		case "APPEND":
			ind.Append = true
		}
	}
	return ind, true
}

// isIndOption reports whether u (upper case) is an IND$FILE option, with or
// without a parenthesized value: where a TSO name ends.
func isIndOption(u string) bool {
	for _, o := range []string{"ASCII", "BINARY", "CRLF", "NOCRLF", "APPEND", "RECFM", "LRECL",
		"BLKSIZE", "SPACE", "TRACKS", "CYLINDERS", "AVBLOCK"} {
		if u == o || strings.HasPrefix(u, o+"(") {
			return true
		}
	}
	return false
}

// DFTBufferSize is the most the client takes in one DFT transmission, from
// its Distributed Data Management query reply; 0 if it has no DFT.
func DFTBufferSize(dev DevInfo) int {
	r := dev.QueryReply(0x95)
	// 95 flags(2) LIMIN(2) LIMOUT(2) ...
	if len(r) < 7 {
		return 0
	}
	in := int(r[3])<<8 | int(r[4])
	out := int(r[5])<<8 | int(r[6])
	n := in
	if out < n {
		n = out
	}
	if n == 0 {
		n = in + out // one of them
	}
	return n
}

// Errors of DFT transfers.
var (
	ErrNoDFT       = errors.New("the terminal does not do DFT file transfer")
	ErrDFTProtocol = errors.New("DFT file transfer: unexpected reply")
	ErrDFTAborted  = errors.New("the transfer was cancelled at the terminal")
)

// the Outbound 3270DS structured field the host sends ahead of each
// request: Write, WCC reset and restore
var dftWrite = []byte{0x40, 0x00, 0xf1, 0xc2}

// Open requests, from a TSO IND$FILE: for the host to read (PUT: with
// the record size) and to write (GET, and messages).
var (
	dftOpenRead = []byte{0xd0, 0x00, 0x12, 0x01, 0x06, 0x01, 0x01, 0x04, 0x03, 0x0a, 0x0a,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x50, 0x05, 0x52, 0x03, 0xf0,
		0x08, 0x06, 0x27, 0x04, 0x3f, 0xef, 0x03, 0x09}
	dftOpenWrite = []byte{0xd0, 0x00, 0x12, 0x01, 0x06, 0x01, 0x01, 0x04, 0x03, 0x0a, 0x0a,
		0x00, 0x00, 0x00, 0x00, 0x11, 0x01, 0x01, 0x00, 0x50, 0x05, 0x52, 0x03, 0xf0,
		0x03, 0x09}
	dftSetCursor = []byte{0xd0, 0x45, 0x11, 0x01, 0x05, 0x00, 0x06, 0x00, 0x09, 0x05, 0x01, 0x03, 0x00}
	dftGet       = []byte{0xd0, 0x46, 0x11, 0x01, 0x04, 0x00, 0x80}
	dftInsert    = []byte{0xd0, 0x47, 0x11, 0x01, 0x05, 0x00, 0x80, 0x00, 0x00}
	dftClose     = []byte{0xd0, 0x41, 0x12}
)

// maxRecord is the most we read of one inbound record.
const maxRecord = 70 * 1024

// ReadRecord reads one inbound 3270 record: the bytes up to IAC EOR, IAC
// IAC undone, other telnet commands dropped, waiting up to timeout.
func ReadRecord(conn net.Conn, timeout time.Duration) ([]byte, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))
	defer conn.SetReadDeadline(time.Time{})
	var rec []byte
	r := bufio.NewReaderSize(oneByte{conn}, 1)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b != 0xff {
			if len(rec) >= maxRecord {
				return nil, errors.New("3270: record too long")
			}
			rec = append(rec, b)
			continue
		}
		c, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		switch c {
		case 0xff:
			rec = append(rec, 0xff)
		case 0xef: // EOR
			return rec, nil
		case 0xfb, 0xfc, 0xfd, 0xfe: // WILL WONT DO DONT: an option byte
			if _, err := r.ReadByte(); err != nil {
				return nil, err
			}
		case 0xfa: // SB ... IAC SE
			for {
				x, err := r.ReadByte()
				if err != nil {
					return nil, err
				}
				if x == 0xff {
					y, err := r.ReadByte()
					if err != nil {
						return nil, err
					}
					if y == 0xf0 {
						break
					}
				}
			}
		}
	}
}

// oneByte reads a byte at a time: nothing past the record is taken from
// the connection.
type oneByte struct{ c net.Conn }

// Read reads one byte at most into p.
func (o oneByte) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return o.c.Read(p)
}

// dftReply reads the client's reply: AID 0x88 and a Data Transfer
// structured field; its request/reply type and the field's bytes after it.
func dftReply(conn net.Conn, timeout time.Duration) (uint16, []byte, error) {
	for {
		rec, err := ReadRecord(conn, timeout)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				return 0, nil, fmt.Errorf("DFT file transfer: no reply: %w", err)
			}
			return 0, nil, err
		}
		if len(rec) == 0 || rec[0] != 0x88 {
			// a key pressed meanwhile (an attention): not ours
			if len(rec) > 0 && (rec[0] == 0x6d || rec[0] == 0x6c || rec[0] == 0xf3) {
				return 0, nil, ErrDFTAborted
			}
			continue
		}
		// 88, then SF: length(2) D0 type(2) ...
		if len(rec) < 6 || rec[3] != 0xd0 {
			return 0, nil, ErrDFTProtocol
		}
		l := int(rec[1])<<8 | int(rec[2])
		if l < 5 || 1+l > len(rec) {
			return 0, nil, ErrDFTProtocol
		}
		typ := uint16(rec[4])<<8 | uint16(rec[5])
		return typ, rec[6 : 1+l], nil
	}
}

// dftRequest sends structured fields, the Write ahead of them.
func dftRequest(conn net.Conn, fields ...[]byte) error {
	return WriteStructuredFields(conn, append([][]byte{dftWrite}, fields...)...)
}

// dftExchange sends fields and checks the reply's type.
func dftExchange(conn net.Conn, timeout time.Duration, want uint16, fields ...[]byte) ([]byte, error) {
	if err := dftRequest(conn, fields...); err != nil {
		return nil, err
	}
	typ, data, err := dftReply(conn, timeout)
	if err != nil {
		return nil, err
	}
	if typ&0xff == 0x08 {
		return nil, fmt.Errorf("%w (error reply %04x)", ErrDFTAborted, typ)
	}
	if typ != want {
		return nil, fmt.Errorf("%w: %04x, expected %04x", ErrDFTProtocol, typ, want)
	}
	return data, nil
}

// dftOpen sends the Open request open for the file name (FT:DATA, FT:MSG),
// padded to 7, and waits for its acknowledgment (0x0009).
func dftOpen(conn net.Conn, timeout time.Duration, open []byte, name string) error {
	sf := append(append([]byte(nil), open...), []byte(fmt.Sprintf("%-7s", name))...)
	_, err := dftExchange(conn, timeout, 0x0009, sf)
	return err
}

// ReceiveFileDFT receives the client's file (an IND$FILE PUT) into w,
// taking at most max bytes.  It returns the bytes taken; the transfer is
// closed, but its closing message (DFTMessage) is the caller's to send.
func ReceiveFileDFT(conn net.Conn, dev DevInfo, w io.Writer, maxBytes int64, timeout time.Duration) (int64, error) {
	if DFTBufferSize(dev) == 0 {
		return 0, ErrNoDFT
	}
	if err := dftOpen(conn, timeout, dftOpenRead, "FT:DATA"); err != nil {
		return 0, err
	}
	var n int64
	for {
		if err := dftRequest(conn, dftSetCursor, dftGet); err != nil {
			return n, err
		}
		typ, data, err := dftReply(conn, timeout)
		if err != nil {
			return n, err
		}
		if typ == 0x4608 {
			// EOF (or the client's error): 6904 2200 is the end
			if len(data) >= 4 && data[2] == 0x22 && data[3] == 0x00 {
				break
			}
			return n, ErrDFTAborted
		}
		if typ&0xff == 0x08 {
			return n, ErrDFTAborted
		}
		// 4605: 6306 recnum(4) C080 61 len(2) data
		if typ != 0x4605 || len(data) < 13 || data[0] != 0x63 || data[8] != 0x61 {
			return n, ErrDFTProtocol
		}
		l := (int(data[9])<<8 | int(data[10])) - 5
		if l < 0 || 11+l > len(data) {
			return n, ErrDFTProtocol
		}
		if n+int64(l) > maxBytes {
			dftExchange(conn, timeout, 0x4109, dftClose)
			return n, fmt.Errorf("DFT file transfer: the file is larger than %d bytes", maxBytes)
		}
		if _, err := w.Write(data[11 : 11+l]); err != nil {
			dftExchange(conn, timeout, 0x4109, dftClose)
			return n, err
		}
		n += int64(l)
	}
	_, err := dftExchange(conn, timeout, 0x4109, dftClose)
	return n, err
}

// SendFileDFT sends r to the client (an IND$FILE GET).  It returns the
// bytes sent; the closing message (DFTMessage) is the caller's to send.
func SendFileDFT(conn net.Conn, dev DevInfo, r io.Reader, timeout time.Duration) (int64, error) {
	size := DFTBufferSize(dev)
	if size == 0 {
		return 0, ErrNoDFT
	}
	// room for the WSF, the Write, the Insert and the data's headers
	chunk := size - 64
	if chunk > 32*1024 {
		chunk = 32 * 1024
	}
	if chunk < 128 {
		chunk = 128
	}
	if err := dftOpen(conn, timeout, dftOpenWrite, "FT:DATA"); err != nil {
		return 0, err
	}
	buf := make([]byte, chunk)
	var n int64
	for {
		k, rerr := io.ReadFull(r, buf)
		if k > 0 {
			data := append([]byte{0xd0, 0x47, 0x04, 0xc0, 0x80, 0x61, byte((k + 5) >> 8), byte(k + 5)}, buf[:k]...)
			if _, err := dftExchange(conn, timeout, 0x4705, dftInsert, data); err != nil {
				return n, err
			}
			n += int64(k)
		}
		if rerr == io.EOF || rerr == io.ErrUnexpectedEOF {
			break
		}
		if rerr != nil {
			dftExchange(conn, timeout, 0x4109, dftClose)
			return n, rerr
		}
	}
	_, err := dftExchange(conn, timeout, 0x4109, dftClose)
	return n, err
}

// DFTMessage ends a transfer with IND$FILE's message to the user:
// "TRANS03 File transfer complete" for success (the emulator shows its
// own), or e.g. "TRANS13 Error writing file to host: file transfer
// canceled".  The message goes as ASCII, up to 80 characters.
func DFTMessage(conn net.Conn, timeout time.Duration, msg string) error {
	if err := dftOpen(conn, timeout, dftOpenWrite, "FT:MSG"); err != nil {
		return err
	}
	var b bytes.Buffer
	for _, r := range msg {
		if r < 0x20 || r > 0x7e || r == '$' {
			r = ' '
		}
		b.WriteRune(r)
	}
	text := []byte(fmt.Sprintf("%-80s", strings.TrimSpace(b.String())+"$"))
	if len(text) > 85 {
		text = text[:85]
	}
	data := append([]byte{0xd0, 0x47, 0x04, 0xc0, 0x80, 0x61, byte((len(text) + 5) >> 8), byte(len(text) + 5)}, text...)
	_, err := dftExchange(conn, timeout, 0x4705, dftInsert, data)
	return err
}
