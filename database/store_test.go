// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package database

import (
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
