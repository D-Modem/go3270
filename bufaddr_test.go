// SPDX-License-Identifier: AGPL-3.0-only AND MIT
// From https://github.com/racingmars/go3270/: Copyright 2020 by
// Matthew R. Wilson, under the MIT license (LICENSE-MIT in the project
// root).  D-Modem's changes: Copyright (C) 2026 D-Modem Authors, under
// the AGPL-3.0-only (LICENSE).

package go3270

import (
	"testing"
)

// TestEncode: getpos encodes 12-bit buffer addresses, and 14-bit ones on a
// large screen.
func TestEncode(t *testing.T) {
	encoded := getpos(0, 0, 80)
	if encoded[0] != 0x40 || encoded[1] != 0x40 {
		t.Error("Position (0, 0) not correctly encoded")
	}

	encoded = getpos(11, 39, 80)
	if encoded[0] != 0x4e || encoded[1] != 0xd7 {
		t.Error("Position (11, 39) not correctly encoded")
	}

	// Large screen, 14-bit addressing
	encoded = getpos(100, 120, 130)
	if encoded[0] != 0x33 || encoded[1] != 0x40 {
		t.Errorf("Position (100, 120) on 130-col screen not correctly encoded")
	}
}

// TestDecode: decodeBufAddr decodes 12-bit and 14-bit buffer addresses.
func TestDecode(t *testing.T) {
	decoded := decodeBufAddr([2]byte{0x40, 0x40})
	if decoded != 0 {
		t.Error("Buffer address incorrectly decoded")
	}

	decoded = decodeBufAddr([2]byte{0x4e, 0xd7})
	if decoded != 919 {
		t.Error("Buffer address incorrectly decoded")
	}

	// Large screen, 14-bit addressing
	decoded = decodeBufAddr([2]byte{0x33, 0x40})
	if decoded != 13120 {
		t.Error("14-bit buffer address incorrectly decoded")
	}
}
