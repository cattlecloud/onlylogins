// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package format

import "strings"

// maskSize bounds the number of bullets rendered for a masked password, so
// the display never reveals more than a hint of the true length.
const maskSize = 16

// Mask renders a password as a string of bullet characters, one per code
// point in the plaintext, capped at maskMax. This lets the masked display
// hint at the password length without revealing the value itself.
func Mask(plaintext string) string {
	n := min(maskSize, len([]rune(plaintext)))
	return strings.Repeat("\u2022", n)
}
