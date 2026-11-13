// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"github.com/shoenig/test/must"
)

const (
	// timeout is long enough for any reasonable testcase to complete; if you
	// exceed this threshold the test needs to be refactored
	timeout = 1 * time.Minute
)

func openTestDB(t *testing.T) (Store, scope.C) {
	ctx, cancel := scope.TTL(timeout)
	t.Cleanup(cancel)

	dir := t.TempDir()
	filename := filepath.Join(dir, "test.db")
	t.Log("testdb", filename)

	db, err := Open(&Config{Path: filename})
	must.NoError(t, err)

	ierr := db.Init(ctx)
	must.NoError(t, ierr)

	return db, ctx
}

func TestOpenDB(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	must.NotNil(t, ctx)
	must.NotNil(t, db)
}

func TestOpen_CreatesFile0600(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), "new.db")

	db, err := Open(&Config{Path: filename})
	must.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	info, serr := os.Stat(filename)
	must.NoError(t, serr)
	must.Eq(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestOpen_PreservesExistingPermissions(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), "existing.db")
	file, cerr := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0o644)
	must.NoError(t, cerr)
	must.NoError(t, file.Close())

	db, err := Open(&Config{Path: filename})
	must.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	info, serr := os.Stat(filename)
	must.NoError(t, serr)
	must.Eq(t, os.FileMode(0o644), info.Mode().Perm())
}
