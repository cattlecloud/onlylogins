// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
)

type Snaps interface {
	Create(scope.C, *litesql.SnapshotOptions) error
}

type snapsql struct {
	log *ulog.Log
	ldb *litesql.LiteDB
}

func (s *store) Snaps() Snaps {
	return &snapsql{
		log: ulog.New("database/snaps"),
		ldb: s.ldb,
	}
}

func (s *snapsql) Create(ctx scope.C, opts *litesql.SnapshotOptions) error {
	return s.ldb.Snapshot(ctx, opts)
}
