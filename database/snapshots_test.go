// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"os"
	"path/filepath"
	"testing"

	"cattlecloud.net/go/litesql"
	"github.com/shoenig/test/must"
)

func TestSnaps_Create(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)

	dir := t.TempDir()
	snaps := db.Snaps()

	err := snaps.Create(ctx, &litesql.SnapshotOptions{
		Directory: dir,
		Step:      -1,
	})
	must.NoError(t, err)

	entries, lerr := os.ReadDir(dir)
	must.NoError(t, lerr)
	must.SliceLen(t, 1, entries)

	snapshotPath := filepath.Join(dir, entries[0].Name())
	info, serr := os.Stat(snapshotPath)
	must.NoError(t, serr)
	must.Positive(t, info.Size())
}
