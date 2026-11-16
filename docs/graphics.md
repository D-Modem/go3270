# Graphics, query replies and IND$FILE

This page tells what this fork adds to [racingmars/go3270](https://github.com/racingmars/go3270).

This fork keeps each query reply that the client sends during
`NegotiateTelnet()`. It also adds 3270 graphics:

 * `DevInfo.QueryReply(qcode)` and `DevInfo.QueryCodes()` return the raw
   query replies. `DevInfo.CellSize()` is the character cell in pels.
   `DevInfo.SymbolSets()` describes the character sets of the client and
   its support for programmed symbols. `DevInfo.SupportsVectorGraphics()`
   reports the Segment query reply (0xB0), if the client sent it or listed
   it in the Summary. If the Summary lists replies that did not come with
   the Query, the library asks for them with a Query List. IBM Personal
   Communications sends its graphics replies (Segment, Graphic Color and
   the other replies) only after a Query List.
 * `LoadPS` builds the Load Programmed Symbols structured field (formats 1
   and 3) from `Symbol` bitmaps. `WriteStructuredFields()` sends it. To
   show the symbols, use a `Field` with `CharSet` set to the LCID of the
   symbol set and `Raw` set to the code points of the symbols.
 * `ImagePicture()` builds an Object Picture structured field that draws
   bitmaps with native 3270 vector graphics (experimental).
   `ImagePictures()` crops each bitmap to the area that it draws. It splits
   a picture larger than 64K into many segments. It also accepts the mix.
   Use `MixOverpaint` to send an image as one bitmap for each colour. This
   mix operates correctly on PCOMM (the exclusive or of PCOMM does not
   combine blue with the other colours).
   `GraphicColors()` is the graphics palette of the terminal (its Graphic
   Color reply, or the 8 base colours). `DitherBitmaps()` changes an image
   into one bitmap for each of its colours, error-diffused, for
   `MixOverpaint`.

The x3270 family does not support these functions. IBM Personal
Communications, HostExplorer and Quick3270 support them. The formats are from
[GA23-0059-07][ga23-07] (5.13, 5.38, 6.12, 6.51) and Greg Price's
[3270 Graphics][grfx].

`cmd/gfxtest` is a TN3270 server to try the graphics on an emulator:

    go run ./cmd/gfxtest -addr :3270 -image photo.png -log gfxtest.log

It shows the data that the terminal reported (its size, cell size, GE, the
Segment reply, programmed symbols, each query reply in hex). It draws these
test pictures as Object Pictures:

 * A one-plane test pattern.
 * Colour bars as three XOR planes, or as seven bitmaps.
 * Corner markers, to check the origin.
 * The `-image` file, dithered to 8 colours.

The menu has these settings: the origin, position, size, mix, Erase/Reset,
and the order of the graphics and the screen (before or after). The server
also sends structured fields that you type in hex.

With `-preview DIR`, the server also writes each picture to DIR as a PNG.
The PNG shows the expected result on the terminal. Compare it with the
result that the terminal shows. With mix O, the image goes as one bitmap
for each colour, overpainted, without exclusive or. It is diffused
(Floyd-Steinberg) to the palette of the terminal, from its Graphic Color
query reply. The palette of PCOMM has 15 colours and black (orange, purple,
the dark shades, grey). A terminal without the reply gets the 8 base
colours.
Test 9 sends a Query List for each reply that the Summary lists but the
terminal did not send. (PCOMM lists its graphics replies, but does not send
them for a plain Query.) The
image test reports the size and format of the file, as decoded.

Each record in each direction goes to the log, with the graphics orders and
the inbound structured fields of the terminal decoded. If an emulator draws
nothing, start with a trace from it.

 * IND$FILE file transfer, the side of the host, in DFT mode: `ParseIndFile()`
   reads the `IND$FILE GET|PUT name options` command that an emulator
   types. `SendFileDFT()` and `ReceiveFileDFT()` move the file.
   `DFTMessage()` ends with the message of IND$FILE
   (`TRANS03 File transfer complete`). `DFTBufferSize()` tells if the
   terminal supports DFT. `ReadRecord()` reads one inbound record, with its
   structured fields. The layouts are the layouts of a real TSO IND$FILE,
   from the test traces of x3270.

Fixes:

 * A query reply that is too short (length 0 to 3) does not cause a panic
   in `NegotiateTelnet()` now.
 * The query replies that a client can send are limited to 16 KB.
 * The library telnet-escapes encoded field text.

[ga23-07]: https://bitsavers.org/pdf/ibm/3174/GA23-0059-07_3270_Data_Stream_Programmers_Reference_199206.pdf
[grfx]: https://www.prycroft6.com.au/misc/3270grfx.html
