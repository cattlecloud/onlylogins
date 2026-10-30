// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package entropy

import "math"

// Bits returns the total Shannon information content (in bits) of s, equal to
// the per-rune Shannon entropy H multiplied by the number of runes N.
//
// H = -sum(p(x) * log2(p(x)))        // shannon entropy bits per rune
// Bits = H * N                       // total bits over the string
//
// For an N-rune string uniformly drawn from a set of S distinct runes this
// simplifies to log2(S) * N.
func Bits(s string) float64 {
	runes := []rune(s)
	size := float64(len(runes))

	// track count of each character in s
	counts := make(map[rune]int)
	for _, r := range runes {
		counts[r]++
	}

	// accumulate H = -sum(p * log2(p)) over distinct runes
	entropy := 0.0
	for _, c := range counts {
		p := float64(c) / size
		entropy -= p * math.Log2(p)
	}

	// total bits = per-rune Shannon entropy * rune count
	return entropy * size
}

// Sequence return true if s contains a sequence of repeating, incrementing, or decrementing
// characters. This prevents passwords like "abcd...", "zyxwu...", etc.
func Sequence(s string) bool {
	// tolerance is the largest number of characters allowed in a sequence run.
	const tolerance = 4

	runes := []rune(s)

	run := 0
	for i := 0; i < len(runes)-1; i++ {
		delta := int(runes[i+1]) - int(runes[i])
		if delta == -1 || delta == 0 || delta == 1 {
			run++
		} else {
			run = 0
		}
		if run >= tolerance {
			// true indicates there is a sequence detected
			return true
		}
	}

	// no violating sequence detected
	return false
}
