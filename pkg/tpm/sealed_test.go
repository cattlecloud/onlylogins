// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package tpm

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestSealedRoundTrip(t *testing.T) {
	t.Parallel()

	in := &Sealed{
		Version: SealedVersion,
		Hash:    "sha256",
		PCRs:    []int{0, 1, 2},
		Private: []byte{1, 2, 3, 4},
		Public:  []byte{5, 6, 7, 8},
	}

	data, err := in.Encode()
	must.NoError(t, err)

	out, err := DecodeSealed(data)
	must.NoError(t, err)
	must.Eq(t, in, out)
}

func TestDecodeSealedUnknownVersion(t *testing.T) {
	t.Parallel()

	data := []byte(`{"version":99,"hash":"sha256","pcrs":[0],"private":"AA==","public":"AA=="}`)
	_, err := DecodeSealed(data)
	must.Error(t, err)
}

func TestDecodeSealedInvalid(t *testing.T) {
	t.Parallel()

	_, err := DecodeSealed([]byte("not json"))
	must.Error(t, err)
}
