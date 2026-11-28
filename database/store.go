// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"errors"
	"fmt"
	"os"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/pkg/encrypt"
)

type Store interface {
	Init(scope.C) error
	Close() error
	Users() Users
	Admin() Admin
	Audit() Audits
	Crons() Crons
	Maint() Maint
	Snaps() Snaps
	Folders() Folders
	Documents() Documents
	Plans() Plans
	Shares(encrypt.Cypher) Shares
	Stripe() Stripe
}

// Open a database using the given configuration.
func Open(config *Config) (Store, error) {
	// touch the database file so that we ensure it exists; and if it doesn't,
	// the file exists with 0600 permissions which sqlite will then inherit
	// when it actually creates the database.
	if err := touch(config.Path); err != nil {
		return nil, fmt.Errorf("unable to touch database file: %w", err)
	}

	ldb, lerr := litesql.Open(config.Path, litesql.TypicalConfiguration)
	if lerr != nil {
		return nil, fmt.Errorf("unable to open sqlite3 database: %w", lerr)
	}
	return &store{ldb: ldb}, nil
}

// touch ensures the given file exists, creating it with 0600 permissions if it
// does not. Existing files are left untouched; their permissions are preserved.
func touch(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	return file.Close()
}

// New returns a Store backed by the given database.
func New(ldb *litesql.LiteDB) Store {
	return &store{ldb: ldb}
}

type store struct {
	ldb *litesql.LiteDB
}

func (s *store) Close() error {
	return s.ldb.Close()
}

type Config struct {
	Path      string `toml:"path"`
	Snapshots string `toml:"snapshots"`
}

func (c *Config) Validate() error {
	switch {
	case c == nil:
		return errors.New("database config not set")
	case c.Path == "":
		return errors.New("database path config not set")
	case c.Snapshots == "":
		return errors.New("database snapshots config not set")
	default:
		return nil
	}
}
