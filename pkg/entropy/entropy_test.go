// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package entropy

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestBits(t *testing.T) {
	t.Parallel()

	t.Run("Empty", func(t *testing.T) {
		must.Eq(t, 0.0, Bits(""))
	})

	t.Run("SingleRune", func(t *testing.T) {
		must.Eq(t, 0.0, Bits("a"))
	})

	t.Run("AllSameRune", func(t *testing.T) {
		must.Eq(t, 0.0, Bits("aaaa"))
	})

	t.Run("TwoEven", func(t *testing.T) {
		// H = 1 bit, N = 2 runes => 2.0
		must.Eq(t, 2.0, Bits("ab"))
	})

	t.Run("TwoEvenLong", func(t *testing.T) {
		// H = 1 bit, N = 4 runes => 4.0
		must.Eq(t, 4.0, Bits("aabb"))
	})

	t.Run("FourEven", func(t *testing.T) {
		// H = 2 bits, N = 4 runes => 8.0
		must.Eq(t, 8.0, Bits("abcd"))
	})

	t.Run("EightEven", func(t *testing.T) {
		// H = 3 bits, N = 8 runes => 24.0
		must.Eq(t, 24.0, Bits("abcdefgh"))
	})

	t.Run("UnicodeRuneLen", func(t *testing.T) {
		// "éllo" is 4 runes (é is a single code point) but 5 bytes.
		// é:1, l:2, o:1 => H = 1.5, total = 6.0. Verifies rune counting.
		must.Eq(t, 6.0, Bits("éllo"))
	})

	t.Run("RepeatHeavy", func(t *testing.T) {
		// a:3, b:3 => H = 1.0, N = 6 runes => 6.0
		must.Eq(t, 6.0, Bits("aaabbb"))
	})

	t.Run("RealisticPasswordPositive", func(t *testing.T) {
		// sanity: a mixed password should have non-zero bits
		must.Greater(t, 0.0, Bits("P@ssw0rd!"))
	})
}

func TestSequence(t *testing.T) {
	t.Parallel()

	t.Run("Empty", func(t *testing.T) {
		must.False(t, Sequence(""))
	})

	t.Run("Single", func(t *testing.T) {
		must.False(t, Sequence("a"))
	})

	t.Run("TwoIncrease", func(t *testing.T) {
		must.False(t, Sequence("ab"))
	})

	t.Run("FourIncrease_Boundary", func(t *testing.T) {
		// "abcd" -> 4 runes, 3 deltas, run reaches 3 -> not flagged
		must.False(t, Sequence("abcd"))
	})

	t.Run("FourRepeat_Boundary", func(t *testing.T) {
		// "aaaa" -> 4 runes, 3 deltas of 0 -> not flagged
		must.False(t, Sequence("aaaa"))
	})

	t.Run("FiveIncrease_Reject", func(t *testing.T) {
		// "abcde" -> 4 deltas of +1, run reaches 4 -> flagged
		must.True(t, Sequence("abcde"))
	})

	t.Run("FiveRepeat_Reject", func(t *testing.T) {
		// "aaaaa" -> 4 deltas of 0, run reaches 4 -> flagged
		must.True(t, Sequence("aaaaa"))
	})

	t.Run("SixRepeat_DocExample", func(t *testing.T) {
		// "aaaaaa" is the doc's example of a string that must be rejected
		must.True(t, Sequence("aaaaaa"))
	})

	t.Run("SixDecrease", func(t *testing.T) {
		must.True(t, Sequence("zyxwvu"))
	})

	t.Run("SevenIncrease_Doc", func(t *testing.T) {
		// "abcdefg" is the doc's example of a string that must be rejected
		must.True(t, Sequence("abcdefg"))
	})

	t.Run("NumericSequence", func(t *testing.T) {
		must.True(t, Sequence("123456"))
	})

	t.Run("MixedNoRun", func(t *testing.T) {
		// deltas are all outside {-1, 0, 1}, so run never grows
		must.False(t, Sequence("c4t!?"))
	})

	t.Run("InterleavedCompatible", func(t *testing.T) {
		// "aabbcc": every adjacent delta is 0 or +1, so run never resets
		// -> flagged (intentional per spec)
		must.True(t, Sequence("aabbcc"))
	})
}
