// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package tpm

import (
	"errors"
	"fmt"

	"github.com/google/go-tpm/tpm2/transport"
)

// Open connects to the TPM at the given device path.
func Open(path string) (transport.TPMCloser, error) {
	if path == "" {
		return nil, errors.New("no TPM device specified")
	}
	conn, _, err := openTPM(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open TPM at %s: %w", path, err)
	}
	return conn, nil
}
