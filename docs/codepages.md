# Code page support

When a client connects, the `NegotiateTelnet()` function returns an implementation of the `DevInfo` interface. That interface has a `DevInfo.Codepage()` method. This method returns an implementation of the `Codepage` interface for the detected code page of the client, if the code page is known and supported (`nil` if not). The `ScreenOpts` structure that you give to the `ShowScreenOpts()` function has a `Codepage` field. Put `DevInfo.Codepage()` in this field, to make sure that the UTF-8–EBCDIC translation is correct for that client.

If `ScreenOpts.Codepage` is `nil` (because you do not set it, or because `DevInfo.Codepage()` returns `nil`), go3270 uses its global default code page for that interaction with the client. The global default is the CP1047 code page.

All the examples in this repository show the correct handling of code pages. They keep the `DevInfo` from the telnet negotiation, and give `DevInfo.Codepage()` to all screen send and receive calls.

To change the global default code page, call the SetCodepage() function when your application initializes. Call it before you use the library for client connections. This is a global setting, not a setting for each connection. SetCodepage() accepts a Codepage interface. This interface has methods that encode Go UTF-8 strings to EBCDIC, and decode EBCDIC byte slices to Go UTF-8 strings.

go3270 now has these codepage functions. They cover all the single-byte code pages that x3270 v4.3ga10 supports:

 * `Codepage037()`
 * `Codepage273()`
 * `Codepage275()`
 * `Codepage277()`
 * `Codepage278()`
 * `Codepage280()`
 * `Codepage284()`
 * `Codepage285()`
 * `Codepage297()`
 * `Codepage424()`
 * `Codepage500()`
 * `Codepage803()`
 * `Codepage870()`
 * `Codepage871()`
 * `Codepage875()`
 * `Codepage880()`
 * `Codepage924()`
 * `Codepage1026()`
 * `Codepage1047()`
 * `Codepage1140()`
 * `Codepage1141()`
 * `Codepage1142()`
 * `Codepage1143()`
 * `Codepage1144()`
 * `Codepage1145()`
 * `Codepage1146()`
 * `Codepage1147()`
 * `Codepage1148()`
 * `Codepage1149()`
 * `Codepage1160()`

There is also `CodepageBracket()`, the default code page of the x3270 family. It is almost the same as CP 1047, but `^` and `¬` are at their positions in CP 37. (x3270 reports the same code page ID for CP37 and "bracket". "bracket" is the default code page for x3270. Thus, if the client is in the x3270 family and the codepage is 37, go3270 uses "bracket".)

If you need support for other standard EBCDIC code pages, tell the maintainers.

To configure go3270 to use one of the codepages by default, use code similar to this example:

```
import (
    "github.com/D-Modem/go3270"
)

func init() {
    go3270.SetCodepage(go3270.Codepage1047())
}
```

The library detects the codepage of the client. Thus, do not use the default codepage as your usual method. Always give the `DevInfo.Codepage()` value to the `ShowScreenOpts()` function, or as the last optional argument to the `HandleScreen()` or `HandleScreenAlt()` functions. Use the global default only as a fallback, if the library does not detect the code page of the client correctly.

All the codepage implementations of go3270 also support most characters from the "graphic escape" code page 310. For a correct display, the client must support graphic escape, and its font must have the correct characters. The `DevInfo.SupportsGE()` function returns a boolean that tells if the client supports the graphic escape alternate code page. Use the related Unicode characters in your Go UTF-8 strings. The library sends them as the EBCDIC two-byte sequence of 0x08, followed by the position in code page 310. The library also processes GE sequences in incoming field values.
