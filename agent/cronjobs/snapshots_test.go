// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package cronjobs

import (
	"os"
	"path/filepath"
	"testing"

	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/crons"
	"github.com/shoenig/test/must"
)

func TestSnapshots_Start(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	dir := t.TempDir()
	snaps := NewSnapshots(
		&crons.Config{Expression: "* * * * *"},
		&database.Config{Snapshots: dir},
		store,
	)

	result := snaps.Start()
	must.NotNil(t, result)
	must.Eq(t, crons.ExitSuccess, result.Code)

	entries, lerr := os.ReadDir(dir)
	must.NoError(t, lerr)
	must.SliceLen(t, 1, entries)

	snapshotPath := filepath.Join(dir, entries[0].Name())
	info, serr := os.Stat(snapshotPath)
	must.NoError(t, serr)
	must.Positive(t, info.Size())
}
