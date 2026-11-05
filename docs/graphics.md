# Graphics, query replies and IND$FILE

What this fork adds over [racingmars/go3270](https://github.com/racingmars/go3270).

This fork keeps every query reply the client sends during
`NegotiateTelnet()` and adds 3270 graphics:

 * `DevInfo.QueryReply(qcode)` and `DevInfo.QueryCodes()` return the raw
   query replies; `DevInfo.CellSize()` is the character cell in pels;
   `DevInfo.SymbolSets()` describes the client's character sets and
   programmed symbol support; `DevInfo.SupportsVectorGraphics()` reports
   the Segment query reply (0xB0), sent or listed in the Summary. Replies
   the Summary lists that did not come with the Query are asked for with
   a Query List: IBM Personal Communications sends its graphics replies
   (Segment, Graphic Color and the rest) only then.
 * `LoadPS` builds the Load Programmed Symbols structured field (formats 1
   and 3) from `Symbol` bitmaps, and `WriteStructuredFields()` sends it.
   A `Field` with `CharSet` set to the symbol set's LCID and `Raw` set to
   the symbols' code points shows them.
 * `ImagePicture()` builds an Object Picture structured field that draws
   bitmaps with native 3270 vector graphics (experimental).
   `ImagePictures()` crops each bitmap to what it draws, splits a picture
   over 64K into several segments, and takes the mix: `MixOverpaint` for
   an image as a bitmap a colour, which is what works on PCOMM (its
   exclusive or does not combine blue with the other colours).
   `GraphicColors()` is the terminal's graphics palette (its Graphic Color
   reply, or the 8 base colours), and `DitherBitmaps()` turns an image
   into a bitmap per colour of it, error-diffused, for `MixOverpaint`.

The x3270 family supports none of these; IBM Personal Communications,
HostExplorer and Quick3270 do. The formats are from
[GA23-0059-07][ga23-07] (5.13, 5.38, 6.12, 6.51) and Greg Price's
[3270 Graphics][grfx].

`cmd/gfxtest` is a TN3270 server for trying the graphics on an emulator:

    go run ./cmd/gfxtest -addr :3270 -image photo.png -log gfxtest.log

It shows what the terminal reported (its size, cell size, GE, the Segment
reply, programmed symbols, every query reply in hex). It draws test
pictures as Object Pictures:

 * a one-plane test pattern;
 * colour bars as three XOR planes, or as seven bitmaps;
 * corner markers, to check the origin;
 * the `-image` file, dithered to 8 colours.

The origin, position, size, mix, Erase/Reset, and whether the graphics come
before or after the screen are all settings on the menu. It also sends
structured fields typed in hex.

With `-preview DIR`, each picture is also written to DIR as a PNG of what
the terminal should show, for comparison with what it does show. With mix
O the image goes as a bitmap per colour, overpainted, with no exclusive or,
diffused (Floyd-Steinberg) to the terminal's own palette from its Graphic
Color query reply: PCOMM's has 15 colours besides black (orange, purple,
the dark shades, grey). A terminal without the reply gets the 8 base
colours.
Test 9 sends a Query List for any replies the Summary lists but the
terminal didn't send (PCOMM lists its graphics replies but doesn't send
them for a plain Query). The
image test reports the file's size and format as decoded.

Every record each way goes to the log, with the graphics orders and the
terminal's inbound structured fields decoded. A trace from an emulator
that draws nothing is the place to start.

 * IND$FILE file transfer, the host's side, in DFT mode: `ParseIndFile()`
   reads the `IND$FILE GET|PUT name options` command an emulator types,
   `SendFileDFT()` and `ReceiveFileDFT()` move the file, `DFTMessage()`
   ends with IND$FILE's message (`TRANS03 File transfer complete`), and
   `DFTBufferSize()` tells whether the terminal does DFT at all.
   `ReadRecord()` reads one inbound record, structured fields and all.
   The layouts are a real TSO IND$FILE's, from x3270's test traces.

Fixes: a query reply too short to be one (length 0 to 3) no longer panics
`NegotiateTelnet()`, the query replies a client may send are limited to
16 KB, and encoded field text is telnet-escaped.

[ga23-07]: https://bitsavers.org/pdf/ibm/3174/GA23-0059-07_3270_Data_Stream_Programmers_Reference_199206.pdf
[grfx]: https://www.prycroft6.com.au/misc/3270grfx.html
