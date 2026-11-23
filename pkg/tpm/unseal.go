// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package tpm

import (
	"fmt"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
)

// Unseal loads the object held in sealed and releases its bound data, provided
// the PCRs recorded in sealed still hold the values they had when the object
// was created. The returned bytes are the original plaintext.
func Unseal(conn transport.TPM, sealed *Sealed) ([]byte, error) {
	srk, err := createSRK(conn)
	if err != nil {
		return nil, err
	}
	defer flushContext(conn, srk.ObjectHandle)

	private, err := tpm2.Unmarshal[tpm2.TPM2BPrivate](sealed.Private)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal sealed private area: %w", err)
	}

	public, err := tpm2.Unmarshal[tpm2.TPM2BPublic](sealed.Public)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal sealed public area: %w", err)
	}

	loaded, err := (tpm2.Load{
		ParentHandle: tpm2.NamedHandle{
			Handle: srk.ObjectHandle,
			Name:   srk.Name,
		},
		InPrivate: *private,
		InPublic:  *public,
	}).Execute(conn)
	if err != nil {
		return nil, fmt.Errorf("unable to load sealed object: %w", err)
	}
	defer flushContext(conn, loaded.ObjectHandle)

	selection, err := pcrSelection(sealed.PCRs)
	if err != nil {
		return nil, err
	}

	values, err := readPCRValues(conn, selection)
	if err != nil {
		return nil, err
	}
	composite := compositeDigest(values)

	session, cleanup, err := tpm2.PolicySession(conn, tpm2.TPMAlgSHA256, 16)
	if err != nil {
		return nil, fmt.Errorf("unable to start policy session: %w", err)
	}
	defer func() { _ = cleanup() }()

	if _, err := (tpm2.PolicyPCR{
		PolicySession: session.Handle(),
		PcrDigest:     tpm2.TPM2BDigest{Buffer: composite},
		Pcrs:          selection,
	}).Execute(conn); err != nil {
		return nil, fmt.Errorf("unable to satisfy PCR policy: %w", err)
	}

	unsealed, err := (tpm2.Unseal{
		ItemHandle: tpm2.AuthHandle{
			Handle: loaded.ObjectHandle,
			Name:   loaded.Name,
			Auth:   session,
		},
	}).Execute(conn)
	if err != nil {
		return nil, fmt.Errorf("unable to unseal data: %w", err)
	}

	return unsealed.OutData.Buffer, nil
}
