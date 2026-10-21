// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package format

import "strings"

// Base32 returns the base-32 representation of num.
//
// Not the same thing as base32-encoding.
func Base32(num int64) string {
	const chars = "1234567890wbcdzfghxjklmnopqrstyv"
	name := new(strings.Builder)
	for num > 0 {
		remainder := num % 32
		name.Write([]byte{chars[remainder]})
		num /= 32
	}
	return name.String()
}
