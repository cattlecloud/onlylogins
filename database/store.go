// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package database

import (
	"errors"
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
)

type Store interface {
	Init(scope.C) error
	Users() Users
	Audit() Audits
	Close() error
	Folders() Folders
	Documents() Documents
}

// Open a database using the given configuration.
func Open(config *Config) (Store, error) {
	ldb, lerr := litesql.Open(config.Path, litesql.TypicalConfiguration)
	if lerr != nil {
		return nil, fmt.Errorf("unable to open sqlite3 database: %w", lerr)
	}
	return &store{ldb: ldb}, nil
}

// Open an in-memory database for testing using the given litesql.Configuration.
func Testing(config *litesql.Configuration) (Store, error) {
	ldb, lerr := litesql.Open(":memory:", config)
	if lerr != nil {
		return nil, fmt.Errorf("unable to open testing sqlite3 database: %w", lerr)
	}
	return &store{ldb: ldb}, nil
}

type store struct {
	ldb *litesql.LiteDB
}

func (s *store) Close() error {
	return s.ldb.Close()
}

type Config struct {
	Path string `toml:"path"`
	// TODO: snapshots
}

func (c *Config) Validate() error {
	switch {
	case c == nil:
		return errors.New("database config not set")
	case c.Path == "":
		return errors.New("database path config not set")
	default:
		return nil
	}
}
