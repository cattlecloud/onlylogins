// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package tpm

import (
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/shoenig/test/must"
)

func TestManufacturerName(t *testing.T) {
	t.Parallel()

	t.Run("known", func(t *testing.T) {
		must.Eq(t, "STMicroelectronics", manufacturerName(0x53544D20))
		must.Eq(t, "Infineon", manufacturerName(0x49465800))
		must.Eq(t, "FIDO Alliance", manufacturerName(0xFFFFF1D0))
	})

	t.Run("unknown", func(t *testing.T) {
		must.Eq(t, "unknown (0xDEADBEEF)", manufacturerName(0xDEADBEEF))
	})
}

func TestKindOfDevice(t *testing.T) {
	t.Parallel()

	must.Eq(t, kindResourceManager, kindOfDevice("/dev/tpmrm0"))
	must.Eq(t, kindResourceManager, kindOfDevice("/dev/tpmrm1"))
	must.Eq(t, kindRaw, kindOfDevice("/dev/tpm0"))
	must.Eq(t, kindRaw, kindOfDevice("/dev/tpm1"))
}

func TestFormatRevision(t *testing.T) {
	t.Parallel()

	must.Eq(t, "1.38", formatRevision(138))
	must.Eq(t, "1.59", formatRevision(159))
	must.Eq(t, "1.05", formatRevision(105))
	must.Eq(t, "2.00", formatRevision(200))
}

func TestFormatFirmware(t *testing.T) {
	t.Parallel()

	must.Eq(t, "1.42.2.0", formatFirmware(0x0001002A, 0x00020000))
	must.Eq(t, "7.85.0.0", formatFirmware(0x00070055, 0x00000000))
	must.Eq(t, "1.0.0.1", formatFirmware(0x00010000, 0x00000001))
}

func TestOpenDeviceMissing(t *testing.T) {
	t.Parallel()

	conn, kind, err := openTPM(filepath.Join(t.TempDir(), "missing"))
	must.Nil(t, conn)
	must.Eq(t, kindUnknown, kind)
	must.Error(t, err)
}

func TestOpenDeviceRegularFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "not-a-tpm")
	must.NoError(t, os.WriteFile(path, nil, 0600))

	conn, kind, err := openTPM(path)
	must.Nil(t, conn)
	must.Eq(t, kindUnknown, kind)
	must.ErrorContains(t, err, "unsupported TPM file mode")
}

func TestOpenDeviceSocket(t *testing.T) {
	t.Parallel()

	// unix socket paths are limited to 104 bytes on macOS
	path := filepath.Join(t.TempDir(), "s")
	if len(path) > 100 {
		t.Skip("temporary directory path is too long for a unix socket")
	}

	var lc net.ListenConfig
	listener, err := lc.Listen(t.Context(), "unix", path)
	must.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	conn, kind, err := openTPM(path)
	must.NoError(t, err)
	must.NotNil(t, conn)
	must.Eq(t, kindSocket, kind)
	must.NoError(t, conn.Close())
}

func TestCandidatesExtra(t *testing.T) {
	t.Parallel()

	paths := candidates([]string{"b", "a", "a"})

	// only assert on the extras; discovered devices vary by host
	must.SliceContains(t, paths, "a")
	must.SliceContains(t, paths, "b")
	must.True(t, slices.IsSorted(paths))

	count := 0
	for _, path := range paths {
		if path == "a" {
			count++
		}
	}
	must.Eq(t, 1, count)
}

func TestFindMissing(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing")
	devices := Find([]string{path})

	must.SliceContainsFunc(t, devices, path, func(device Device, path string) bool {
		return device.Path == path && device.Kind == kindUnknown && device.Err != nil
	})
}
