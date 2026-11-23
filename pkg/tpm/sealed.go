// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package tpm

import (
	"encoding/json"
	"fmt"
)

// SealedVersion is the current version of the sealed blob format.
const SealedVersion uint16 = 1

// Sealed is the on-disk representation of a passphrase sealed to a TPM.
//
// Private and Public hold the marshalled TPM2BPrivate and TPM2BPublic values
// returned by TPM2_Create; together with the storage root key they allow the
// object to be loaded and unsealed again. PCRs records which PCRs the policy
// is bound to, because the public area stores only the resulting digest.
type Sealed struct {
	Version uint16 `json:"version"`
	Hash    string `json:"hash"`
	PCRs    []int  `json:"pcrs"`
	Private []byte `json:"private"`
	Public  []byte `json:"public"`
}

// Encode serializes the sealed blob as JSON. The binary fields are encoded
// using base64.
func (s *Sealed) Encode() ([]byte, error) {
	return json.Marshal(s)
}

// DecodeSealed deserializes a sealed blob produced by Encode.
func DecodeSealed(data []byte) (*Sealed, error) {
	var s Sealed
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unable to decode sealed blob: %w", err)
	}
	if s.Version != SealedVersion {
		return nil, fmt.Errorf("unsupported sealed blob version %d", s.Version)
	}
	return &s, nil
}
