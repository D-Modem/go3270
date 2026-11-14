<!-- Made by tools/apiref from the package's doc comments: change the code, not this page. -->

# API reference

The exported API of package `go3270`, from its doc comments (the same text that `go doc -all` and [pkg.go.dev](https://pkg.go.dev/github.com/D-Modem/go3270) show).

    import "github.com/D-Modem/go3270"

## Constants

```go
const (
	PlaneBlue  byte = 1
	PlaneRed   byte = 2
	PlaneGreen byte = 4
)
```

The planes of a triple-plane symbol set, for LoadPS.Plane.

## Variables

```go
var (
	ErrNoDFT       = errors.New("the terminal does not do DFT file transfer")
	ErrDFTProtocol = errors.New("DFT file transfer: unexpected reply")
	ErrDFTAborted  = errors.New("the transfer was cancelled at the terminal")
)
```

Errors of DFT transfers.

```go
var Debug io.Writer
```

Enable go3270 library debugging by setting Debug to an io.Writer. Disable debugging by setting it to nil (the default value).

```go
var ErrNo3270 = errors.New("couldn't negotiate telnet options for tn3270")
```

ErrNo3270 indicates that the telnet client did not respond properly to the options negotiation that are expected for a tn3270 client.

```go
var ErrTelnetError = errors.New("telnet or 3270 protocol error")
```

ErrTelnetError indicates an unexpected response was encountered in the telnet protocol.

```go
var ErrUnknownTerminal = errors.New("unknown terminal type")
```

ErrUnknownTerminal indicates the client did not identify itself as an IBM-3277, 3278, 3279, or IBM-DYNAMIC model. All modern tn3270 clients should report as IBM-3278 models or IBM-DYNAMIC.

## Functions

### AIDtoString

```go
func AIDtoString(aid AID) string
```

AIDtoString returns a string representation of an AID key name.

### DFTBufferSize

```go
func DFTBufferSize(dev DevInfo) int
```

DFTBufferSize is the most the client takes in one DFT transmission, from its Distributed Data Management query reply; 0 if it has no DFT.

### DFTMessage

```go
func DFTMessage(conn net.Conn, timeout time.Duration, msg string) error
```

DFTMessage ends a transfer with IND$FILE's message to the user: "TRANS03 File transfer complete" for success (the emulator shows its own), or e.g. "TRANS13 Error writing file to host: file transfer canceled".  The message goes as ASCII, up to 80 characters.

### ImagePicture

```go
func ImagePicture(x, y int, planes ...Bitmap) ([]byte, error)
```

ImagePicture is an Object Picture structured field (GA23-0059 5.38), for WriteStructuredFields, that draws bitmaps with their top left corner at x, y: graphics coordinates, 0, 0 the middle of the screen, y upwards. Planes are combined by exclusive or, so a blue and a red plane make pink where both are on.

The orders inside (Begin Segment, Set Mix, Set Current Position, Set Color, Begin/Image Data/End Image) are 3270 native vector graphics as GDDM sends them; experimental: check DevInfo.SupportsVectorGraphics, and try it on your emulator.

### ImagePictures

```go
func ImagePictures(x, y int, o PictureOptions, planes ...Bitmap) ([][]byte, int, error)
```

ImagePictures is Object Picture structured fields, for WriteStructuredFields, that draw bitmaps with their top left corner at x, y (graphics coordinates, 0, 0 the middle of the screen, y upwards), and the number of the next segment, for pictures sent with these.

Unlike ImagePicture, each bitmap is cropped to its pels that are on, and a picture too large for one segment (64K) is split into several, each in its own structured field, a bitmap into bands of rows if need be.

### ReadRecord

```go
func ReadRecord(conn net.Conn, timeout time.Duration) ([]byte, error)
```

ReadRecord reads one inbound 3270 record: the bytes up to IAC EOR, IAC IAC undone, other telnet commands dropped, waiting up to timeout.

### ReceiveFileDFT

```go
func ReceiveFileDFT(conn net.Conn, dev DevInfo, w io.Writer, maxBytes int64, timeout time.Duration) (int64, error)
```

ReceiveFileDFT receives the client's file (an IND$FILE PUT) into w, taking at most max bytes.  It returns the bytes taken; the transfer is closed, but its closing message (DFTMessage) is the caller's to send.

### RunTransactions

```go
func RunTransactions(conn net.Conn, dev DevInfo, initial Tx,
	data any) error
```

RunTransactions begins running transaction functions, starting with the initial transaction, until a transaction eventually returns nil for the next transaction, or until a transaction function returns a non-nil error value. data (which may be nil, if the initial transaction does not require data) is passed in as the data to the initial transaction.

dev is the DevInfo of the connected client, as obtained from NegotiateTelnet(). It is safe to pass a nil DevInfo, in which case all transactions will only be able to operate with the default 24x80 screen size.

### SendFileDFT

```go
func SendFileDFT(conn net.Conn, dev DevInfo, r io.Reader, timeout time.Duration) (int64, error)
```

SendFileDFT sends r to the client (an IND$FILE GET).  It returns the bytes sent; the closing message (DFTMessage) is the caller's to send.

### SetCodepage

```go
func SetCodepage(cs Codepage)
```

SetCodepage sets the codepage/character set that go3270 uses. This is a global setting, so if you're expecting clients to be configured to use a character set other than go3270's default, cp1047, you should probably set this during your application initialization and then leave it unchanged after. This is \_not\_ a per-connection setting.

For per-client codepage, set the ScreenOpts.Codepage field in the calls to ShowScreenOpts() or the codepage argument to HandleScreen() and HandleScreenAlt().

### ShowScreenNoResponse

```go
func ShowScreenNoResponse(screen Screen, values map[string]string,
	crow, ccol int, conn net.Conn) error
```

Deprecated: use ShowScreenOpts with ScreenOpts.NoResponse = true.

NOTE: this deprecated function is NOT codepage-aware. The global code page set by SetCodepage will always be used.

### UnNegotiateTelnet

```go
func UnNegotiateTelnet(conn net.Conn, timeout time.Duration) error
```

UnNegotiateTelnet will naively (e.g. not checking client responses) attempt to restore the telnet options state to what it was before NegotiateTelnet() was called.

### WriteStructuredFields

```go
func WriteStructuredFields(conn net.Conn, fields ...[]byte) error
```

WriteStructuredFields sends a Write Structured Field command with the given structured fields. Each field is given from its SFID on; its length is put in front of it.

## Types

### AID

```go
type AID byte
```

AID is an Action ID character.

```go
const (
	AIDNone  AID = 0x60
	AIDEnter AID = 0x7D
	AIDPF1   AID = 0xF1
	AIDPF2   AID = 0xF2
	AIDPF3   AID = 0xF3
	AIDPF4   AID = 0xF4
	AIDPF5   AID = 0xF5
	AIDPF6   AID = 0xF6
	AIDPF7   AID = 0xF7
	AIDPF8   AID = 0xF8
	AIDPF9   AID = 0xF9
	AIDPF10  AID = 0x7A
	AIDPF11  AID = 0x7B
	AIDPF12  AID = 0x7C
	AIDPF13  AID = 0xC1
	AIDPF14  AID = 0xC2
	AIDPF15  AID = 0xC3
	AIDPF16  AID = 0xC4
	AIDPF17  AID = 0xC5
	AIDPF18  AID = 0xC6
	AIDPF19  AID = 0xC7
	AIDPF20  AID = 0xC8
	AIDPF21  AID = 0xC9
	AIDPF22  AID = 0x4A
	AIDPF23  AID = 0x4B
	AIDPF24  AID = 0x4C
	AIDPA1   AID = 0x6C
	AIDPA2   AID = 0x6E
	AIDPA3   AID = 0x6B
	AIDClear AID = 0x6D
)
```

### Bitmap

```go
type Bitmap struct {
	Color Color // Blue, Red, Pink, Green, Turquoise, Yellow or White
	Rows  []string
}
```

Bitmap is one plane of an image for ImagePicture: Rows of '#' (on) and anything else (off), all of the same length, drawn in Color.

#### DitherBitmaps

```go
func DitherBitmaps(img image.Image, p Palette) []Bitmap
```

DitherBitmaps is img in the palette's colors (Floyd-Steinberg error diffusion), a Bitmap a color, each pel in one at most, black pels in none (the background shows): for ImagePictures with MixOverpaint, which draws them a pel for a pixel.

### Codepage

```go
type Codepage interface {
	// Decode converts a slice of EBCDIC bytes into a UTF-8 string.
	Decode(e []byte) string

	// Encode converts a UTF-8 string into a slice of EBCDIC bytes.
	Encode(s string) []byte

	// ID returns the name of this codepage. Usually a numeric string like
	// "037" or "1047", but could also be a name such as "bracket" if IBM has
	// not assigned a number to the particular codepage.
	ID() string
}
```

Implementations of Codepage provide EBCDIC\<->UTF-8 translation. By default, go3270 is configured to use CP 1047. You may alternatively set a different codepage using the SetCodepage() function during your application initialization.

#### Codepage037

```go
func Codepage037() Codepage
```

Codepage037 returns the IBM CP 037 code page (United States/Canada).

#### Codepage1026

```go
func Codepage1026() Codepage
```

Codepage1026 returns the IBM CP 1026 code page (Turkey).

#### Codepage1047

```go
func Codepage1047() Codepage
```

Codepage1047 returns the IBM CP 1047 code page (Latin-1/Open Systems), go3270's default.

#### Codepage1140

```go
func Codepage1140() Codepage
```

Codepage1140 returns the IBM CP 1140 code page: CP 037 (United States/Canada) with the euro sign.

#### Codepage1141

```go
func Codepage1141() Codepage
```

Codepage1141 returns the IBM CP 1141 code page: CP 273 (Germany/Austria) with the euro sign.

#### Codepage1142

```go
func Codepage1142() Codepage
```

Codepage1142 returns the IBM CP 1142 code page: CP 277 (Denmark/Norway) with the euro sign.

#### Codepage1143

```go
func Codepage1143() Codepage
```

Codepage1143 returns the IBM CP 1143 code page: CP 278 (Finland/Sweden) with the euro sign.

#### Codepage1144

```go
func Codepage1144() Codepage
```

Codepage1144 returns the IBM CP 1144 code page: CP 280 (Italy) with the euro sign.

#### Codepage1145

```go
func Codepage1145() Codepage
```

Codepage1145 returns the IBM CP 1145 code page: CP 284 (Spain/Latin America) with the euro sign.

#### Codepage1146

```go
func Codepage1146() Codepage
```

Codepage1146 returns the IBM CP 1146 code page: CP 285 (United Kingdom) with the euro sign.

#### Codepage1147

```go
func Codepage1147() Codepage
```

Codepage1147 returns the IBM CP 1147 code page: CP 297 (France) with the euro sign.

#### Codepage1148

```go
func Codepage1148() Codepage
```

Codepage1148 returns the IBM CP 1148 code page: CP 500 (International) with the euro sign.

#### Codepage1149

```go
func Codepage1149() Codepage
```

Codepage1149 returns the IBM CP 1149 code page: CP 871 (Iceland) with the euro sign.

#### Codepage1160

```go
func Codepage1160() Codepage
```

Codepage1160 returns the IBM CP 1160 code page (Thai, with the euro sign).

#### Codepage273

```go
func Codepage273() Codepage
```

Codepage273 returns the IBM CP 273 code page (Germany/Austria).

#### Codepage275

```go
func Codepage275() Codepage
```

Codepage275 returns the IBM CP 275 code page (Brazil).

#### Codepage277

```go
func Codepage277() Codepage
```

Codepage277 returns the IBM CP 277 code page (Denmark/Norway).

#### Codepage278

```go
func Codepage278() Codepage
```

Codepage278 returns the IBM CP 278 code page (Finland/Sweden).

#### Codepage280

```go
func Codepage280() Codepage
```

Codepage280 returns the IBM CP 280 code page (Italy).

#### Codepage284

```go
func Codepage284() Codepage
```

Codepage284 returns the IBM CP 284 code page (Spain/Latin America).

#### Codepage285

```go
func Codepage285() Codepage
```

Codepage285 returns the IBM CP 285 code page (United Kingdom).

#### Codepage297

```go
func Codepage297() Codepage
```

Codepage297 returns the IBM CP 297 code page (France).

#### Codepage424

```go
func Codepage424() Codepage
```

Codepage424 returns the IBM CP 424 code page (Hebrew).

#### Codepage500

```go
func Codepage500() Codepage
```

Codepage500 returns the IBM CP 500 code page (International).

#### Codepage803

```go
func Codepage803() Codepage
```

Codepage803 returns the IBM CP 803 code page (Hebrew, old code).

#### Codepage870

```go
func Codepage870() Codepage
```

Codepage870 returns the IBM CP 870 code page (Latin-2 multilingual).

#### Codepage871

```go
func Codepage871() Codepage
```

Codepage871 returns the IBM CP 871 code page (Iceland).

#### Codepage875

```go
func Codepage875() Codepage
```

Codepage875 returns the IBM CP 875 code page (Greek).

#### Codepage880

```go
func Codepage880() Codepage
```

Codepage880 returns the IBM CP 880 code page (Cyrillic multilingual).

#### Codepage924

```go
func Codepage924() Codepage
```

Codepage924 returns the IBM CP 924 code page (Latin-9).

#### CodepageBracket

```go
func CodepageBracket() Codepage
```

CodepageBracket returns the "bracket" code page, c/x3270's default: CP 037 with \[, ], Ý and ¨ where CP 1047 has them.

### Color

```go
type Color byte
```

Color is a 3270 extended field attribute color value

```go
const (
	DefaultColor Color = 0
	Blue         Color = 0xf1
	Red          Color = 0xf2
	Pink         Color = 0xf3
	Green        Color = 0xf4
	Turquoise    Color = 0xf5
	Yellow       Color = 0xf6
	White        Color = 0xf7
)
```

The valid 3270 colors

### DevInfo

```go
type DevInfo interface {
	// AltDimensions returns the number or rows and columns on the alternate
	// screen size.
	AltDimensions() (rows, cols int)

	// TerminalType reports the terminal-provided identification string. All
	// modern tn3270 clients will report one of the IBM-3278 models (-2, -3,
	// -4, or -5), or IBM-DYNAMIC if the alternate screen size isn't one of
	// the fixed sizes of the 3278 models. This string is purely
	// informational; the actual size of the alternate screen is available
	// from AltDimensions().
	TerminalType() string

	// Codepage is the Codepage interface that implements the EBCDIC
	// translation for the detected code page for the terminal, if supported.
	// This may be nil if the client code page is unknown. Whenever calling
	// the screen functions, always pass the value returned by this Codepage()
	// function in the ScreenOpts (nil is allowed to default to the global
	// default codepage).
	Codepage() Codepage

	// SupportsGE returns true if the client indicated it supports graphic
	// escape to code page 310.
	SupportsGE() bool

	// QueryReply returns the client's query reply with the given QCODE
	// (0x81 Usable Area, 0x85 Character Sets, 0xB0 Segment, ...), from the
	// QCODE byte on, or nil if the client sent none. The slice is a copy.
	QueryReply(qcode byte) []byte

	// QueryCodes lists the QCODEs of the query replies the client sent, in
	// ascending order.
	QueryCodes() []byte

	// CellSize is the size of the default character cell in pels, from the
	// Usable Area query reply (0, 0 if the client did not say).
	CellSize() (width, height int)

	// SymbolSets describes the client's character sets and its support for
	// loading programmed symbols (see LoadPS), from the Character Sets query
	// reply; nil if the client sent none.
	SymbolSets() *SymbolSets

	// SupportsVectorGraphics reports whether the client answered the query
	// with the Segment query reply (0xB0), or lists it in its Summary
	// reply, which means it takes graphics segments: vector graphics in
	// Object Picture structured fields.
	SupportsVectorGraphics() bool
	// contains filtered or unexported methods
}
```

DevInfo provides information about the terminal that is connected.

3270 terminals operate at a default screen size of 24 rows that are 80 columns wide. The normal "Write/Erase" datastream command always writes to the default 24x80 buffer. But some terminals support more rows and/or columns, and the alternate sized buffer may be written to with the "Write/Erase Alternate" command.

#### NegotiateTelnet

```go
func NegotiateTelnet(conn net.Conn) (DevInfo, error)
```

NegotiateTelnet will negotiate the options necessary for tn3270 on a new telnet connection, conn.

### Field

```go
type Field struct {
	// Row is the row, 0-based, that the field attribute character should
	// begin at. When using the standard screen of 24 rows, Row must be 0-23.
	// When writing to the alternate screen, Row may be up to 1 less than the
	// number of rows on the screen.
	Row int

	// Col is the column, 0-based, that the field attribute character should
	// begin at. When using the standard screen of 80 columns, Col must be
	// 0-79. When writing to the alternate screen, Col may be up to 1 less
	// than the number of columns on the screen.
	Col int

	// Text is the content of the field to display.
	Content string

	// PositionOnly will use a Set Buffer Address (SBA) command to move the
	// cursor to the position, but not insert a new attribute byte. The only
	// properties of the Field that will apply if PositionOnly is true are:
	// Row, Col, Content, Name. Others will be silently ignored.
	PositionOnly bool

	// Write allows the user to edit the value of the field.
	Write bool

	// Autoskip causes protected (Write = false) fields to automatically be
	// skipped and the cursor should move to the next field upon encountering
	// this field. Autoskip is ignored on fields with Write = true.
	Autoskip bool

	// Intense indicates this field should be displayed with high intensity.
	Intense bool

	// Hidden indicates the field content should not be displayed (e.g. a
	// password input field).
	Hidden bool

	// NumericOnly indicates that only numbers may be entered into the field.
	// Very fiew 3270 clients support this, so you must always still validate
	// the input on the server side.
	NumericOnly bool

	// Color is the field color. The default value is the default color.
	Color Color

	// Highlighting is the highlight attribute for the field. The default value
	// is the default (i.e. no) highlighting.
	Highlighting Highlight

	// AttributeOnly will cause this "field" (which won't really be a new
	// field) to use the SA (Set Attribute) 3270 command to change the color
	// and highlighting of the text WITHOUT starting a new field. You can
	// create a field at Row 5, Column 10, with AttributeOnly true, and the
	// text will start in Roe 5, Column 10 and not skip a space to column 11
	// like a new field would when AttributeOnly is false.
	//
	// If AttributeOnly is true, the Write, Autoskip, Intense, Hidden, and
	// NumericOnly properties will have no effect. Only Highlighting and Color
	// will affect the output, and will both default to Default (e.g. default
	// highlighting and default color) if not specificed in this field.
	//
	// After using AttributeOnly to change color and highlighting, you may
	// need to explicitly use another AttributeOnly "field" to reset to
	// defaults, a regular (AttributeOnly = false) field may not end the
	// attributes. (See example1.)
	AttributeOnly bool

	// Name is the name of this field, which is used to get the user-entered
	// data. All writeable fields on a screen must have a unique name.
	// Protected fields may also have a name to populate them dynamically when
	// the screen is sent.
	Name string

	// CharSet is the local ID of the character set the field is shown in:
	// 0 for the default; a programmed symbol set's LCID (see LoadPS). It is
	// sent as an extended field attribute (SFE type 0x43), so it ends with
	// the field; with AttributeOnly, as a Set Attribute (SA type 0x43),
	// which lasts until another changes it (see DefaultCharSet).
	CharSet byte

	// DefaultCharSet, with AttributeOnly, sets the character set back to
	// the default (SA type 0x43, 0) for the characters after it.
	DefaultCharSet bool

	// Raw, if not nil, is sent as the field's content as it is, instead of
	// Content encoded in the code page: the code points of programmed
	// symbols (0x41 to 0xFE) in a CharSet field. Values in the values map do
	// not replace it.
	Raw []byte

	// KeepSpaces will prevent the strings.TrimSpace() function from being
	// called on the field value. Generally you want leading and trailing
	// spaces trimmed from fields in 3270 before processing, but if you are
	// building a whitespace-sensitive application, you can ask for the
	// original, un-trimmed value for a field by setting this to true.
	KeepSpaces bool
}
```

Field is a field on the 3270 screen.

### FieldRules

```go
type FieldRules struct {
	// MustChange, when true, indicates that the value of the field MUST be
	// altered by the user -- if applied to a field with no starting value,
	// this makes the field a required field. If true on a field with a
	// starting value (either in the field's Content attribute, or with an
	// override in the initial values map), then the user must change
	// the value from the default.
	MustChange bool

	// ErrorText is the text displayed with the MustChange validation fails.
	// If ErrorText is the empty string, but MustValidation fails, an error
	// string will be constructed from the field name: "Please enter a valid
	// value for <fieldName>."
	ErrorText string

	// Validator is a function to validate the value the user input into the
	// field. It may be nil if no validation is required. The Validator
	// function is called *after* the MustChange logic, so if you wish to
	// fully handle validation, ensure MustChange is set to false.
	Validator Validator

	// Reset indicates that if the screen fails validation, this field should
	// always be reset to its original/default value, regardless of what the
	// user entered.
	Reset bool
}
```

FieldRules provides the validation rules for a particular field.

### Highlight

```go
type Highlight byte
```

Highlight is a 3270 extended field attribute highlighting method

```go
const (
	DefaultHighlight Highlight = 0
	Blink            Highlight = 0xf1
	ReverseVideo     Highlight = 0xf2
	Underscore       Highlight = 0xf4
)
```

The valid 3270 highlights

### IndFile

```go
type IndFile struct {
	Get     bool     // GET: the host sends the file; PUT: the host receives it
	Name    string   // the host file name, as typed
	ASCII   bool     // ASCII (text) rather than binary
	CRLF    bool     // CR LF line ends
	Append  bool     // PUT appends to the host file
	Options []string // all the options, upper case
}
```

IndFile is an IND$FILE command.

#### ParseIndFile

```go
func ParseIndFile(s string) (*IndFile, bool)
```

ParseIndFile parses "IND$FILE GET|PUT name \[(] options" (TSO puts the options after the name; VM/CMS after a "("), case aside.  A "(" within a name (a TSO member, 'USER.DATA(MEM)') stays in it.

### LoadPS

```go
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
```

LoadPS is the Load Programmed Symbols structured field (GA23-0059 5.13): Symbols loaded into the client's symbol store RWS, from code point First on, to be shown through the character set ID LCID.

#### LoadPS.StructuredField

```go
func (l LoadPS) StructuredField() ([]byte, error)
```

StructuredField is the Load PS structured field, from its SFID on, for WriteStructuredFields.

### Mix

```go
type Mix byte
```

Mix is how a picture's pels combine with what is already drawn.

```go
const (
	MixOverpaint Mix = 2 // the color drawn replaces what was there
	MixXor       Mix = 4 // exclusive or: a blue and a red plane make pink
)
```

### Palette

```go
type Palette map[int]color.RGBA
```

Palette is a terminal's graphics colors: the color of each index (1 blue, 2 red, ... 7 white, and on some 8 to 15), 0 being the background.

#### GraphicColors

```go
func GraphicColors(d DevInfo) Palette
```

GraphicColors is the terminal's graphics colors: from its Graphic Color query reply (0xB4) if it sent one (IBM Personal Communications has 16: orange, purple, dark blue, green and turquoise, mustard and grey besides the 8), else the 8 base colors.

### PictureOptions

```go
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
```

PictureOptions are ImagePictures' options.

### Response

```go
type Response struct {
	// Which Action ID key did the user press?
	AID AID

	// Row the cursor was on (0-based).
	Row int

	// Column the cursor was on (0-based).
	Col int

	// Field values.
	Values map[string]string
}
```

Response encapsulates data received from a 3270 client in response to the previously sent screen.

#### HandleScreen

```go
func HandleScreen(screen Screen, rules Rules, values map[string]string,
	pfkeys, exitkeys []AID, errorField string, crow, ccol int,
	conn net.Conn, codepage ...Codepage) (Response, error)
```

HandleScreen is a higher-level interface to the ShowScreen() function. HandleScreen will loop until all validation rules are satisfied, and only return when an expected AID (i.e. PF) key is pressed.

  - screen is the Screen to display (see ShowScreen()).
  - rules are the Rules to enforce: each key in the Rules map corresponds to a Field.Name in the screen array.
  - values are field values you wish to override (see ShowScreen()).
  - pfkeys and exitkeys are the AID keys that you wish to accept (that is, perform validation and return if successful) and treat as exit keys (unconditionally return).
  - errorField is the name of a field in the screen array that you wish error messages to be written in when HandleScreen loops waiting for a valid user submission.
  - crow and ccol are the initial cursor position.
  - conn is the network connection to the 3270 client.
  - codepage is an optional argument (implemented this way as a varargs argument as a hack to add this feature without breaking API backward compatability) for the codepage to use. Typically you should pass in the return value from DevInfo.Codepage() each time to get the correct codepage that was detected when the client connected. If nil, the global default code page (default 1047, but changed with the SetCodepage() function) will be used.

HandleScreen will return when the user: 1) presses a key in pfkeys AND all fields pass validation, OR 2) the user presses a key in exitkeys. In all other cases, HandleScreen will re-present the screen to the user again, possibly with an error message set in the errorField field.

For alternate screen support (larger than 24x80), use HandleScreenAlt().

#### HandleScreenAlt

```go
func HandleScreenAlt(screen Screen, rules Rules, values map[string]string,
	pfkeys, exitkeys []AID, errorField string, crow, ccol int,
	conn net.Conn, dev DevInfo, codepage ...Codepage) (Response, error)
```

HandleScreenAlt is identical to HandleScreen, but writes to the "alternate" screen size provided by dev. To write a non-24-by-80 screen, use this HandleScreenAlt function with a non-nil dev. If dev is nil, the behavior is identical to HandleScreen, which is limited to 24x80 and will set larger terminals to the default 24x80 mode.

#### ShowScreen

```go
func ShowScreen(screen Screen, values map[string]string, crow, ccol int,
	conn net.Conn) (Response, error)
```

Deprecated: use ShowScreenOpts with default/empty ScreenOpts.

NOTE: this deprecated function is NOT codepage-aware. The global code page set by SetCodepage will always be used.

#### ShowScreenOpts

```go
func ShowScreenOpts(screen Screen,
	values map[string]string, conn net.Conn,
	opts ScreenOpts) (Response, error)
```

ShowScreenOpts writes the 3270 datastream for the screen, with the provided ScreenOpts, to a connection.

Fields that aren't valid (e.g. outside of the screen size) are silently ignored. If a named field has an entry in the values map, the content of the field from the values map is used INSTEAD OF the Field struct's Content field. The values map may be nil if no overrides are needed.

If opts.NoClear is false, the client screen will be cleared before writing the new screen, and the cursor will be repositioned to the values in opts.CursorRow and opts.CursorCol. If opts.NoClear is true, the screen will NOT be cleared, the cursor will NOT be repositioned, and the new screen will be overlayed over the current state of the client screen.

If opts.NoResponse is false, ShowScreenOpts will block before returning, waiting for data from the client and returning the Response. If opts.NoResponse is true, ShowScreenOpts will immediately return after sending the datastream and the Response will be empty.

If using from multiple threads -- one to block and wait for a response, and another to send screens with NoResponse and/or NoClear, be aware that if you change the input fields on screen after the initial blocking call is made, the response fields will not line up correctly and end up being invalid. That is to say, while waiting for a response, don't perform other actions from another thread that could layout the user input fields differently.

### Rules

```go
type Rules map[string]FieldRules
```

Rules is a map of field names (strings) to FieldRules structs. Each field for which you wish validation to occur must appear in the map. Fields not in the map will not have any input validation performed.

### Screen

```go
type Screen []Field
```

Screen is an array of Fields which compose a complete 3270 screen. No checking is performed for lack of overlapping fields, unique field names, fields out of bounds of the screen size (these will simply be omitted from the datastream), etc.

### ScreenOpts

```go
type ScreenOpts struct {
	// If AltScreen is non-nil, the screen will be written to the "alternate"
	// screen size, which is the non-default (24x80) screen dimensions that
	// the terminal supports (although for many terminals, the alternate
	// screen is still just 24x80). If AltScreen is nil, the default (24x80)
	// mode will be used. Never switch between AltScreen and normal screen
	// (e.g. AltScreen = nil) unless NoClear is false. (That is, switching
	// between default and alternate screen size or back requires a screen
	// clear at the same time.) When AltScreen is nil, field positions in the
	// screen must be within the 24x80 screen (so rows 0-23 and cols 0-79),
	// when AltScreen is present, the field positions must be within the
	// dimensions of the DevInfo.AltDimensions() values.
	AltScreen DevInfo

	// Codepage is the Codepage implementation to use when sending text to the
	// client and translating incoming field text from the client. Typically
	// you should pass in the return value from DevInfo.Codepage() each time
	// to get the correct codepage that was detected when the client
	// connected. If nil, the global default code page (default 1047, but
	// changed with the SetCodepage() function) will be used. NOTE: providing
	// a DevInfo to ScreenOpts.AltScreen does _not_ automatically set this
	// value, you must set it explicitly on every call that accepts
	// ScreenOpts.
	Codepage Codepage

	// NoResponse will draw the screen and immediately return, without
	// waiting for any input data from the remote client.
	NoResponse bool

	// NoClear will send the data stream to the remote client without
	// clearing the screen first. Existing data will be overlayed with
	// the current screen.
	NoClear bool

	// CursorRow sets the row (0-indexed) to position the cursor after sending
	// the screen, when NoClear is false. When AltScreen is nil, maximum value
	// is 23; otherwise, maximum is 1 less than the number of rows in
	// AltScreen.
	CursorRow int

	// CursorCol sets the column (0-indexed) to position the cursor after
	// sending the screen, when NoClear is false. When AltScreen is nil,
	// maximum value is 79; otherwise, maximum is 1 less than the number of
	// columns in AltScreen.
	CursorCol int

	// PostSendCallback is a function that, if non-nil, will be called after
	// go3270 sends the datastream to the client, but before it blocks to
	// read the response (if NoResponse is true, the callback will still be
	// called before returning). If the function returns an error, then
	// the ShowScreenOpts() function will return the error instead of a
	// response. The value of CallbackData will be passed as the argument to
	// the function.
	PostSendCallback func(any) error

	// CallbackData is passed as the argument to the PostSendCallback
	// function.
	CallbackData any
}
```

ScreenOpts are the options that callers may set when sending a screen to the 3270 client.

### Symbol

```go
type Symbol []string
```

Symbol is the picture of one programmed symbol: a row of text for each row of pels, top to bottom, '#' (or 'X', '1', '\*') for a pel that is on, anything else for one that is off.

### SymbolSet

```go
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
```

SymbolSet is one character set of a client.

### SymbolSets

```go
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
```

SymbolSets is what a client's Character Sets query reply says about its character sets.

#### SymbolSets.LoadableSets

```go
func (s *SymbolSets) LoadableSets() []SymbolSet
```

LoadableSets lists the sets that programmed symbols can be loaded into.

#### SymbolSets.SupportsFormat

```go
func (s *SymbolSets) SupportsFormat(t int) bool
```

SupportsFormat reports whether Load PS format type t is supported.

### Tx

```go
type Tx func(conn net.Conn, dev DevInfo, data any) (
	next Tx, newdata any, err error)
```

Tx is a function that serves as one transaction in a go3270 application. The Tx function is called with the network connection to the client, the DevInfo for use with alternate screen writes, and a "data" value provided by the previous transaction. Tx functions return the next transaction to run (or nil to indicate the RunTransactions() function should terminate), the data to pass into the next transaction, and any error. If the error is non-nil, the RunTransactions() function will terminate and return the err. A non-nil error is \_not\_ passed between transactions, it terminates transaction processing.

### Validator

```go
type Validator func(input string) bool
```

Validator is a type that represents a function which can perform field input validation. The function is passed a string, input, and returns true if the input is valid or false if the not.

```go
var IsInteger Validator = func(input string) bool {
	input = strings.TrimSpace(input)
	return isIntegerRegexp.MatchString(input)
}
```

IsInteger is a Validator that returns true if, after spaces are trimmed from the beginning and end if the string, the value is an integer (including negative numbers and 0).

```go
var NonBlank Validator = func(input string) bool {
	return !(strings.TrimSpace(input) == "")
}
```

NonBlank is a Validator that returns true if, after spaces are trimmed from the beginning and end of the string, the value is not empty.