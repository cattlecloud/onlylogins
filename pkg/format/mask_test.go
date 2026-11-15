// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package format

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestMask(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		plaintext string
		exp       string
	}{
		{
			name:      "empty",
			plaintext: "",
			exp:       "",
		},
		{
			name:      "short",
			plaintext: "abc",
			exp:       "\u2022\u2022\u2022",
		},
		{
			name:      "exactly cap",
			plaintext: "1234567890abcdef",
			exp:       "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022",
		},
		{
			name:      "longer than cap",
			plaintext: "1234567890abcdefghijklmnopqrstuvwxyz",
			exp:       "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022",
		},
		{
			name:      "multi-byte code points",
			plaintext: "パスワード",
			exp:       "\u2022\u2022\u2022\u2022\u2022",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := Mask(tc.plaintext)
			must.Eq(t, tc.exp, result)
		})
	}
}
