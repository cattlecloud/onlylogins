// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cattlecloud.net/go/atomicfs/v2"
	"cattlecloud.net/go/babycli"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/tpm"
	"github.com/shoenig/ignore"
)

func tpmCmd() *babycli.Component {
	return &babycli.Component{
		Name: "tpm",
		Help: "operate on tpm devices",
		Description: `
Inspect and operate the Trusted Platform Module (TPM)
devices attached to this system. A TPM is a dedicated
security device that generates and stores cryptographic
keys, allowing secrets to be bound to the hardware of
a specific machine.`,
		Components: babycli.Components{
			tpmDevicesCmd(),
			tpmSealCmd(),
			tpmVerifyCmd(),
		},
	}
}

func tpmDevicesCmd() *babycli.Component {
	return &babycli.Component{
		Name: "devices",
		Help: "list available tpm devices",
		Description: `
List the TPM 2.0 devices available on this system.
Kernel devices (/dev/tpm*), well known software TPM
sockets, and any extra devices given with --device are
opened and queried for their manufacturer, firmware
version, and specification revision. A device that
exists but cannot be opened, for example because of
missing permissions, is listed along with the reason.`,
		Flags: babycli.Flags{
			{
				Type:    babycli.StringFlag,
				Long:    "device",
				Short:   "d",
				Help:    "additional device path to probe",
				Repeats: true,
			},
		},
		Function: func(c *babycli.Component) babycli.Code {
			devices := tpm.Find(c.GetStrings("device"))
			switch len(devices) {
			case 0:
				fmt.Println("no TPM devices found")
			default:
				for _, device := range devices {
					fmt.Println(describe(device))
				}
			}
			return babycli.Success
		},
	}
}

func tpmSealCmd() *babycli.Component {
	return &babycli.Component{
		Name: "seal",
		Help: "seal a passphrase to the tpm",
		Description: `
Read a passphrase from standard input and seal it to the
TPM, bound to the given set of PCRs. The resulting blob
is written to the file given with --output. Keep a paper
copy of the plaintext passphrase in a secure location so
it can be restored in an emergency, for example total
loss of the server, or if PCR values change.`,
		Flags: babycli.Flags{
			{
				Type:    babycli.StringFlag,
				Long:    "output",
				Short:   "o",
				Help:    "path to write the sealed blob",
				Require: true,
			},
			{
				Type:    babycli.StringFlag,
				Long:    "pcrs",
				Short:   "p",
				Help:    "comma-separated PCR indices to seal to",
				Default: &babycli.Default{Value: tpm.DefaultPCRs, Show: true},
			},
			{
				Type:    babycli.StringFlag,
				Long:    "device",
				Short:   "d",
				Help:    "TPM device path to seal against",
				Require: true,
			},
		},
		Function: func(c *babycli.Component) babycli.Code {
			output := c.GetString("output")
			device := c.GetString("device")

			pcrs, err := tpm.ParsePCRs(c.GetString("pcrs"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "tpm seal: %v\n", err)
				return babycli.Failure
			}

			raw, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "tpm seal: unable to read passphrase: %v\n", err)
				return babycli.Failure
			}

			passphrase := bytes.TrimRight(raw, "\r\n")
			if len(passphrase) == 0 {
				fmt.Fprintln(os.Stderr, "tpm seal: no passphrase read from standard input")
				return babycli.Failure
			}
			if err := encrypt.ValidatePassphrase(string(passphrase)); err != nil {
				fmt.Fprintf(os.Stderr, "tpm seal: %v\n", err)
				return babycli.Failure
			}

			conn, err := tpm.Open(device)
			if err != nil {
				fmt.Fprintf(os.Stderr, "tpm seal: %v\n", err)
				return babycli.Failure
			}
			defer ignore.Close(conn)

			sealed, err := tpm.Seal(conn, pcrs, passphrase)
			if err != nil {
				fmt.Fprintf(os.Stderr, "tpm seal: %v\n", err)
				return babycli.Failure
			}

			js, err := sealed.Encode()
			if err != nil {
				fmt.Fprintf(os.Stderr, "tpm seal: unable to json encode sealed data: %v\n", err)
				return babycli.Failure
			}

			// safely write blob to disk (sync + rename)
			if err := atomicfs.New(
				atomicfs.WithTempDir(filepath.Dir(output)),
				atomicfs.WithMode(0600),
			).WriteFile(output, bytes.NewReader(js)); err != nil {
				fmt.Fprintf(os.Stderr, "tpm seal: unable to write sealed json: %v\n", err)
				return babycli.Failure
			}

			fmt.Printf("complete: sealed passphrase to %s (pcrs %s)\n", output, tpm.FormatPCRs(pcrs))
			fmt.Println("reminder: store the plaintext passphrase on paper in a secure location")
			fmt.Println("done.")
			return babycli.Success
		},
	}
}

func tpmVerifyCmd() *babycli.Component {
	return &babycli.Component{
		Name: "verify",
		Help: "verify a sealed passphrase",
		Description: `
Unseal the passphrase held in the blob given with --input
and confirm it matches the expected passphrase format.`,
		Flags: babycli.Flags{
			{
				Type:    babycli.StringFlag,
				Long:    "input",
				Short:   "i",
				Help:    "path to the sealed blob",
				Require: true,
			},
			{
				Type:    babycli.StringFlag,
				Long:    "device",
				Short:   "d",
				Help:    "TPM device path to unseal against",
				Require: true,
			},
		},
		Function: func(c *babycli.Component) babycli.Code {
			input := c.GetString("input")
			device := c.GetString("device")

			data, err := os.ReadFile(input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "tpm verify: unable to read sealed blob: %v\n", err)
				return babycli.Failure
			}

			sealed, err := tpm.DecodeSealed(data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "tpm verify: %v\n", err)
				return babycli.Failure
			}

			conn, err := tpm.Open(device)
			if err != nil {
				fmt.Fprintf(os.Stderr, "tpm verify: %v\n", err)
				return babycli.Failure
			}
			defer ignore.Close(conn)

			passphrase, err := tpm.Unseal(conn, sealed)
			if err != nil {
				fmt.Fprintf(os.Stderr, "tpm verify: %v\n", err)
				return babycli.Failure
			}

			passphrase = bytes.TrimRight(passphrase, "\r\n")
			if err := encrypt.ValidatePassphrase(string(passphrase)); err != nil {
				fmt.Fprintf(os.Stderr, "tpm verify: %v\n", err)
				return babycli.Failure
			}

			fmt.Printf("verified sealed passphrase (pcrs %s)\n", tpm.FormatPCRs(sealed.PCRs))
			return babycli.Success
		},
	}
}

// describe formats a single device as a human readable line.
func describe(device tpm.Device) string {
	if device.Err != nil {
		return fmt.Sprintf("%s (%s): unavailable (%v)", device.Path, device.Kind, device.Err)
	}

	return fmt.Sprintf("%s (%s): %s, firmware %s, spec %s",
		device.Path, device.Kind, device.Manufacturer, device.Firmware, device.SpecRevision,
	)
}
