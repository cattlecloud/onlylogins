// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

// Package tpm discovers and describes the TPM 2.0 devices available on the
// system. It is the foundation for using a TPM to protect application secrets.
package tpm

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
	"github.com/google/go-tpm/tpm2/transport/linuxtpm"
	"github.com/google/go-tpm/tpm2/transport/linuxudstpm"
	"github.com/hashicorp/go-set/v3"
	"github.com/shoenig/ignore"
	"github.com/shoenig/lang"
)

// Kinds of TPM device attachments.
const (
	kindRaw             = "raw"
	kindResourceManager = "resource manager"
	kindSocket          = "socket"
	kindUnknown         = "unknown"
)

// socketPaths are well known locations of software TPM emulators, which are
// common in virtual machines and containers where no kernel TPM device exists.
var socketPaths = []string{
	"/tmp/tpm_server.socket",            // IBM TPM simulator
	"/tmp/swtpm.sock",                   // swtpm default
	"/var/lib/swtpm/sockets/swtpm.sock", // swtpm system service
	"/run/swtpm/swtpm.sock",             // swtpm runtime directory
}

// manufacturerNames maps the 4 character TPM_PT_MANUFACTURER identifier to a
// human readable vendor name, as assigned by the Trusted Computing Group.
var manufacturerNames = map[uint32]string{
	0x414D4400: "AMD",
	0x41544D4C: "Atmel",
	0x4252434D: "Broadcom",
	0x43495343: "Cisco",
	0x48504500: "HPE",
	0x49424D00: "IBM",
	0x49465800: "Infineon",
	0x494E5443: "Intel",
	0x4C454E00: "Lenovo",
	0x4D534654: "Microsoft",
	0x4E534D20: "National Semiconductor",
	0x4E544300: "Nuvoton",
	0x4E545A00: "Nationz",
	0x51554D00: "Qualcomm",
	0x534D5343: "SMSC",
	0x534D534E: "Samsung",
	0x53544D20: "STMicroelectronics",
	0x57454300: "Winbond",
	0xFFFFF1D0: "FIDO Alliance",
}

// Device describes a TPM 2.0 device found on the system.
type Device struct {
	// Path is the filesystem path used to communicate with the device.
	Path string

	// Kind describes how the device is attached to the system.
	Kind string

	// Manufacturer is the human readable vendor name.
	Manufacturer string

	// ManufacturerID is the raw TPM_PT_MANUFACTURER identifier.
	ManufacturerID uint32

	// Firmware is the firmware version, formatted as major.minor.revision.build.
	Firmware string

	// SpecRevision is the TPM specification revision, formatted as major.minor.
	SpecRevision string

	// Err is set when the device exists but could not be opened or queried.
	Err error
}

// Find discovers the TPM 2.0 devices on the system and probes each one for
// identifying information. In addition to kernel devices and well known
// software TPM sockets, any paths in extra are probed. A device which exists
// but cannot be opened is included with its Err field set.
func Find(extra []string) []Device {
	return lang.Map(candidates(extra), probe)
}

// candidates returns the deduplicated, sorted set of paths to probe.
func candidates(extra []string) []string {
	paths := set.NewTreeSet(cmp.Compare[string])

	// kernel devices, e.g. /dev/tpm0 and /dev/tpmrm0
	if entries, err := os.ReadDir("/dev"); err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "tpm") {
				paths.Insert(filepath.Join("/dev", entry.Name()))
			}
		}
	}

	// software TPM emulators listening on a unix socket
	for _, path := range socketPaths {
		if _, err := os.Stat(path); err == nil {
			paths.Insert(path)
		}
	}

	// devices explicitly requested by the caller
	paths.InsertSlice(extra)

	return paths.Slice()
}

// probe opens a single device and queries its identifying properties.
func probe(path string) Device {
	conn, kind, err := openTPM(path)
	device := Device{
		Path: path,
		Kind: kind,
	}
	if err != nil {
		device.Err = err
		return device
	}
	defer ignore.Close(conn)

	// the manufacturer property is mandatory for TPM 2.0, so a failure here
	// means the device is not a TPM 2.0 device (or is otherwise unusable)
	manufacturer, err := property(conn, tpm2.TPMPTManufacturer)
	if err != nil {
		device.Err = fmt.Errorf("not a TPM 2.0 device: %w", err)
		return device
	}
	device.ManufacturerID = manufacturer
	device.Manufacturer = manufacturerName(manufacturer)

	revision, err := property(conn, tpm2.TPMPTRevision)
	if err != nil {
		device.Err = err
		return device
	}
	device.SpecRevision = formatRevision(revision)

	version1, err := property(conn, tpm2.TPMPTFirmwareVersion1)
	if err != nil {
		device.Err = err
		return device
	}

	version2, err := property(conn, tpm2.TPMPTFirmwareVersion2)
	if err != nil {
		device.Err = err
		return device
	}

	device.Firmware = formatFirmware(version1, version2)

	return device
}

// openTPM connects to the TPM at path and reports how it is attached.
func openTPM(path string) (transport.TPMCloser, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, kindUnknown, err
	}

	switch {
	case info.Mode()&os.ModeDevice != 0:
		conn, err := linuxtpm.Open(path)
		return conn, kindOfDevice(path), err
	case info.Mode()&os.ModeSocket != 0:
		conn, err := linuxudstpm.Open(path)
		return conn, kindSocket, err
	default:
		return nil, kindUnknown, fmt.Errorf("unsupported TPM file mode %s", info.Mode())
	}
}

// property queries a single fixed TPM property.
func property(t transport.TPM, prop tpm2.TPMPT) (uint32, error) {
	rsp, err := tpm2.GetCapability{
		Capability:    tpm2.TPMCapTPMProperties,
		Property:      uint32(prop),
		PropertyCount: 1,
	}.Execute(t)
	if err != nil {
		return 0, err
	}

	props, err := rsp.CapabilityData.Data.TPMProperties()
	if err != nil {
		return 0, err
	}

	// the TPM returns the next available property, which may not be the one
	// that was requested
	for _, p := range props.TPMProperty {
		if p.Property == prop {
			return p.Value, nil
		}
	}

	return 0, fmt.Errorf("tpm property 0x%X not available", uint32(prop))
}

// kindOfDevice classifies a device file as a raw TPM or resource manager.
func kindOfDevice(path string) string {
	if strings.HasPrefix(path, "/dev/tpmrm") {
		return kindResourceManager
	}
	return kindRaw
}

// formatRevision formats a TPM_PT_REVISION value, which is the specification
// revision in hundredths.
func formatRevision(revision uint32) string {
	return fmt.Sprintf("%d.%02d", revision/100, revision%100)
}

// formatFirmware formats the two TPM_PT_FIRMWARE_VERSION values as
// major.minor.revision.build.
func formatFirmware(version1, version2 uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		version1>>16, version1&0xFFFF,
		version2>>16, version2&0xFFFF,
	)
}

// manufacturerName returns the vendor name for a TPM manufacturer identifier.
func manufacturerName(id uint32) string {
	if name, ok := manufacturerNames[id]; ok {
		return name
	}
	return fmt.Sprintf("unknown (0x%08X)", id)
}
