// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package commands

import (
	"fmt"

	"cattlecloud.net/go/babycli"
	"cattlecloud.net/onlylogins/pkg/tpm"
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

// describe formats a single device as a human readable line.
func describe(device tpm.Device) string {
	if device.Err != nil {
		return fmt.Sprintf("%s (%s): unavailable (%v)", device.Path, device.Kind, device.Err)
	}

	return fmt.Sprintf("%s (%s): %s, firmware %s, spec %s",
		device.Path, device.Kind, device.Manufacturer, device.Firmware, device.SpecRevision,
	)
}
