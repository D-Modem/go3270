// SPDX-License-Identifier: AGPL-3.0-only AND MIT
// From https://github.com/racingmars/go3270/: Copyright 2025 by
// Matthew R. Wilson, under the MIT license (LICENSE-MIT in the project
// root).  D-Modem's changes: Copyright (C) 2026 D-Modem Authors, under
// the AGPL-3.0-only (LICENSE).

// Example 5 demonstrates support for larger-than-default alternate screen
// sizes in terminals that are larger than 24x80.

package main

import (
	"fmt"
	"net"

	"github.com/D-Modem/go3270"
)

// main listens on port 3270 and handles each connection in its own goroutine.
func main() {
	ln, err := net.Listen("tcp", ":3270")
	if err != nil {
		panic(err)
	}
	fmt.Println("LISTENING ON PORT 3270 FOR CONNECTIONS")
	fmt.Println("Press Ctrl-C to end server.")
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go handle(conn)
	}
}

// handle is the handler for individual user connections.
func handle(conn net.Conn) {
	defer conn.Close()

	// Always begin new connection by negotiating the telnet options
	devinfo, err := go3270.NegotiateTelnet(conn)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = go3270.RunTransactions(conn, devinfo, bigscreen, nil)
	if err != nil {
		fmt.Println(err)
	}
}
