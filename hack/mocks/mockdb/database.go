// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package mockdb

import (
	"testing"
	"time"

	"cattlecloud.net/go/litesql/litesqltest"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/database"
	"github.com/shoenig/test/must"
)

const (
	// timeout is long enough for any reasonable testcase to complete; if you
	// exceed this threshold the test needs to be refactored
	timeout = 10 * time.Second
)

// Open a temporary implementation of database.Store.
func Open(t *testing.T) database.Store {
	store := database.New(litesqltest.Open(t))

	ctx, cancel := scope.TTL(timeout)
	t.Cleanup(cancel)

	ierr := store.Init(ctx)
	must.NoError(t, ierr)

	return store
}
