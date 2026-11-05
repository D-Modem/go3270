# go3270

A Go library for writing TN3270 servers: build 3270 data streams from
fields and read back the attention key and the values the user typed.

This is the D-Modem suite's fork of [racingmars/go3270](https://github.com/racingmars/go3270)
by Matthew R. Wilson, kept API-compatible with it. [bbs](https://github.com/D-Modem/bbs)
uses it for its TN3270 callers.

- Screens from fields, with `ShowScreenOpts()`, `HandleScreen()` and `RunTransactions()`
- Telnet negotiation, terminal size detection and larger-than-24x80 screens
- 31 EBCDIC code pages, detected from the client, plus graphic escape (code page 310)
- In this fork: every query reply kept, programmed symbols, native 3270 vector graphics and images
- In this fork: IND$FILE file transfer (DFT), the host's side
- `cmd/gfxtest`, a TN3270 server for trying graphics on an emulator

## Quick start

    go get github.com/D-Modem/go3270

```go
import "github.com/D-Modem/go3270"
```

then see `example1` to `example5`; `example4` shows the recommended
`RunTransactions()` structure. Code that imported
`github.com/racingmars/go3270` changes only its import path.

## Documentation

- [docs/](docs/README.md): usage, code pages, graphics and IND$FILE, references
- [API reference](https://pkg.go.dev/github.com/D-Modem/go3270)

## The D-Modem suite

| Repository | What it is |
| --- | --- |
| [D-Modem](https://github.com/D-Modem/D-Modem) | The open V.92 softmodem: dmodemd, libdmodem and the Asterisk, FreeSWITCH and ARI modules |
| [app](https://github.com/D-Modem/app) | The D-Modem app for Android and iOS: a dial-up modem on a SIP call |
| [hardware](https://github.com/D-Modem/hardware) | pico-serial: RP2040 serial adapters and phone-line USB sticks |
| [menu](https://github.com/D-Modem/menu) | modemmenu: the login and menu dial-up callers reach |
| [bbs](https://github.com/D-Modem/bbs) | A bulletin board for telnet, SSH, TN3270 and modem callers |
| [go3270](https://github.com/D-Modem/go3270) | A Go library for 3270 terminal servers, used by bbs |
| [brand](https://github.com/D-Modem/brand) | Logos, colours, type and the naming guide |

## Licence

MIT, as upstream; see [LICENSE](LICENSE).
