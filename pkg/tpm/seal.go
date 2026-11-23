// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package tpm

import (
	"cmp"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
	"github.com/hashicorp/go-set/v3"
	"github.com/shoenig/lang"
)

// DefaultPCRs is the comma-separated list of PCR indices used when sealing
// unless overridden. PCRs 0, 1, and 2 cover the platform firmware, firmware
// configuration, and firmware code.
const DefaultPCRs = "0,1,2"

// maxPCRIndex is the largest PCR index accepted by ParsePCRs. PC Client
// platforms are required to implement at least PCRs 0 through 23.
const maxPCRIndex = 23

// ParsePCRs parses a comma-separated list of PCR indices, returning them in
// ascending order.
func ParsePCRs(s string) ([]int, error) {
	parts := strings.Split(s, ",")

	pcrs := set.NewTreeSet(cmp.Compare[int])
	for _, part := range parts {
		part = strings.TrimSpace(part)
		index, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid PCR index %q", part)
		}
		if index < 0 || index > maxPCRIndex {
			return nil, fmt.Errorf("PCR index %d out of range 0-%d", index, maxPCRIndex)
		}
		if !pcrs.Insert(index) {
			return nil, fmt.Errorf("duplicate PCR index %d", index)
		}
	}

	if pcrs.Empty() {
		return nil, errors.New("no PCRs selected")
	}

	return pcrs.Slice(), nil
}

// FormatPCRs renders a list of PCR indices as a comma-separated string.
func FormatPCRs(pcrs []int) string {
	return strings.Join(lang.Map(pcrs, strconv.Itoa), ",")
}

// Seal binds data to the TPM and to the current values of pcrs. The returned
// blob can later be loaded and unsealed on the same TPM, provided the selected
// PCRs still hold the same values.
func Seal(conn transport.TPM, pcrs []int, data []byte) (*Sealed, error) {
	srk, err := createSRK(conn)
	if err != nil {
		return nil, err
	}
	defer flushContext(conn, srk.ObjectHandle)

	digest, err := policyDigest(conn, pcrs)
	if err != nil {
		return nil, err
	}

	created, err := (tpm2.Create{
		ParentHandle: tpm2.NamedHandle{
			Handle: srk.ObjectHandle,
			Name:   srk.Name,
		},
		InSensitive: tpm2.TPM2BSensitiveCreate{
			Sensitive: &tpm2.TPMSSensitiveCreate{
				Data: tpm2.NewTPMUSensitiveCreate(&tpm2.TPM2BSensitiveData{
					Buffer: data,
				}),
			},
		},
		InPublic: tpm2.New2B(tpm2.TPMTPublic{
			Type:    tpm2.TPMAlgKeyedHash,
			NameAlg: tpm2.TPMAlgSHA256,
			ObjectAttributes: tpm2.TPMAObject{
				FixedTPM:        true,
				FixedParent:     true,
				AdminWithPolicy: true,
			},
			AuthPolicy: tpm2.TPM2BDigest{
				Buffer: digest,
			},
		}),
	}).Execute(conn)
	if err != nil {
		return nil, fmt.Errorf("unable to seal data: %w", err)
	}

	return &Sealed{
		Version: SealedVersion,
		Hash:    "sha256",
		PCRs:    append([]int(nil), pcrs...),
		Private: tpm2.Marshal(created.OutPrivate),
		Public:  tpm2.Marshal(created.OutPublic),
	}, nil
}

// createSRK creates a storage root key in the owner hierarchy using the well
// known TCG RSA template. Because the hierarchy seed and template are fixed,
// the same key is produced every time, so it does not need to be persisted.
func createSRK(conn transport.TPM) (*tpm2.CreatePrimaryResponse, error) {
	rsp, err := (tpm2.CreatePrimary{
		PrimaryHandle: tpm2.TPMRHOwner,
		InPublic:      tpm2.New2B(tpm2.RSASRKTemplate),
	}).Execute(conn)
	if err != nil {
		return nil, fmt.Errorf("unable to create storage root key: %w", err)
	}
	return rsp, nil
}

// policyDigest computes the policy digest that requires pcrs to hold their
// current values. It reads the PCRs, hashes their concatenation, and plays
// TPM2_PolicyPCR back in a trial session to obtain the final digest.
func policyDigest(conn transport.TPM, pcrs []int) ([]byte, error) {
	selection, err := pcrSelection(pcrs)
	if err != nil {
		return nil, err
	}

	values, err := readPCRValues(conn, selection)
	if err != nil {
		return nil, err
	}
	composite := compositeDigest(values)

	trial, err := (tpm2.StartAuthSession{
		SessionType: tpm2.TPMSETrial,
		AuthHash:    tpm2.TPMAlgSHA256,
		NonceCaller: tpm2.TPM2BNonce{Buffer: make([]byte, 16)},
	}).Execute(conn)
	if err != nil {
		return nil, fmt.Errorf("unable to start trial policy session: %w", err)
	}
	defer flushContext(conn, trial.SessionHandle)

	if _, err := (tpm2.PolicyPCR{
		PolicySession: trial.SessionHandle,
		PcrDigest:     tpm2.TPM2BDigest{Buffer: composite},
		Pcrs:          selection,
	}).Execute(conn); err != nil {
		return nil, fmt.Errorf("unable to apply PCR policy: %w", err)
	}

	digest, err := (tpm2.PolicyGetDigest{
		PolicySession: trial.SessionHandle,
	}).Execute(conn)
	if err != nil {
		return nil, fmt.Errorf("unable to read policy digest: %w", err)
	}

	return digest.PolicyDigest.Buffer, nil
}

// pcrSelection builds the PCR selection for the SHA-256 bank.
func pcrSelection(pcrs []int) (tpm2.TPMLPCRSelection, error) {
	indices := make([]uint, len(pcrs))
	for i, pcr := range pcrs {
		if pcr < 0 || pcr > maxPCRIndex {
			return tpm2.TPMLPCRSelection{}, fmt.Errorf("PCR index %d out of range 0-%d", pcr, maxPCRIndex)
		}
		indices[i] = uint(pcr)
	}
	return tpm2.TPMLPCRSelection{
		PCRSelections: []tpm2.TPMSPCRSelection{
			{
				Hash:      tpm2.TPMAlgSHA256,
				PCRSelect: tpm2.PCClientCompatible.PCRs(indices...),
			},
		},
	}, nil
}

// readPCRValues returns the digest of each selected PCR, in ascending order.
func readPCRValues(conn transport.TPM, selection tpm2.TPMLPCRSelection) ([][]byte, error) {
	rsp, err := (tpm2.PCRRead{PCRSelectionIn: selection}).Execute(conn)
	if err != nil {
		return nil, fmt.Errorf("unable to read PCRs: %w", err)
	}
	values := make([][]byte, 0, len(rsp.PCRValues.Digests))
	for _, digest := range rsp.PCRValues.Digests {
		values = append(values, digest.Buffer)
	}
	return values, nil
}

// compositeDigest returns the SHA-256 of the concatenated PCR values, which is
// the input expected by TPM2_PolicyPCR.
func compositeDigest(values [][]byte) []byte {
	h := sha256.New()
	for _, value := range values {
		_, _ = h.Write(value)
	}
	return h.Sum(nil)
}

// flushContext best-effort flushes a transient handle, ignoring errors.
func flushContext(conn transport.TPM, handle tpm2.TPMHandle) {
	_, _ = (tpm2.FlushContext{FlushHandle: handle}).Execute(conn)
}
