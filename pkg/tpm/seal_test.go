// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package tpm

import (
	"crypto/sha256"
	"testing"

	"github.com/shoenig/test/must"
)

func TestParsePCRs(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		pcrs, err := ParsePCRs("0,1,2")
		must.NoError(t, err)
		must.Eq(t, []int{0, 1, 2}, pcrs)
	})

	t.Run("unsorted", func(t *testing.T) {
		pcrs, err := ParsePCRs("2,0,1")
		must.NoError(t, err)
		must.Eq(t, []int{0, 1, 2}, pcrs)
	})

	t.Run("whitespace", func(t *testing.T) {
		pcrs, err := ParsePCRs(" 0 , 1 ")
		must.NoError(t, err)
		must.Eq(t, []int{0, 1}, pcrs)
	})

	t.Run("duplicate", func(t *testing.T) {
		_, err := ParsePCRs("0,0")
		must.Error(t, err)
	})

	t.Run("empty", func(t *testing.T) {
		_, err := ParsePCRs("")
		must.Error(t, err)
	})

	t.Run("non-numeric", func(t *testing.T) {
		_, err := ParsePCRs("a")
		must.Error(t, err)
	})

	t.Run("out of range", func(t *testing.T) {
		_, err := ParsePCRs("24")
		must.Error(t, err)
	})
}

func TestFormatPCRs(t *testing.T) {
	t.Parallel()

	must.Eq(t, "0,1,2", FormatPCRs([]int{0, 1, 2}))
	must.Eq(t, "", FormatPCRs(nil))
}

func TestCompositeDigest(t *testing.T) {
	t.Parallel()

	values := [][]byte{{0x01, 0x02}, {0x03, 0x04}}
	want := sha256.Sum256([]byte{0x01, 0x02, 0x03, 0x04})

	must.Eq(t, want[:], compositeDigest(values))
}
