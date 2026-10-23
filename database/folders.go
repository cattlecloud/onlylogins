// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package database

import (
	"errors"
	"fmt"
	"strings"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

var ErrFolderLabelTaken = errors.New("folder name is already in use")

type Folders interface {
	// metrics
	Count(scope.C) (int, error)

	// folder operations
	Create(scope.C, *sitecontent.CreateFolderRequest) error
	List(scope.C, users.Identity) ([]*sitecontent.Folder, error)
}

type foldersql struct {
	log *ulog.Log
	ldb *litesql.LiteDB
}

func (s *store) Folders() Folders {
	return &foldersql{
		log: ulog.New("database/folders"),
		ldb: s.ldb,
	}
}

func (s *foldersql) Count(ctx scope.C) (int, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return 0, fmt.Errorf("unable to count folders: %w", xerr)
	}
	defer done()

	const statement = `select count(id) from folders`

	return litesql.QueryRow(ctx, tx, scanCount, statement)
}

func (s *foldersql) Create(ctx scope.C, request *sitecontent.CreateFolderRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const statement = `
	insert into folders (
		users_id, label, icon
	) values ( ?, ?, ? )`

	if _, ferr := s.ldb.ExecID(ctx, tx, statement,
		request.User,
		request.Label,
		int(request.Icon),
	); isFolderLabelTaken(ferr) {
		return fmt.Errorf("unable to create folder: %w", ErrFolderLabelTaken)
	} else if ferr != nil {
		return fmt.Errorf("unable to create folder: %w", ferr)
	}

	return tx.Commit()
}

func (s *foldersql) List(ctx scope.C, user users.Identity) ([]*sitecontent.Folder, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to list folders: %w", xerr)
	}
	defer done()

	const statement = `
	select
		id,
		label,
		icon
	from folders
	where
		users_id = ?
	order by label asc`

	f := func(sf litesql.ScanFunc) (*sitecontent.Folder, error) {
		folder := new(sitecontent.Folder)
		err := sf(&folder.ID, &folder.Label, &folder.Icon)
		return folder, err
	}

	return litesql.QueryRows(ctx, tx, f, statement, user)
}

// isFolderLabelTaken reports whether err indicates that the folders
// (users_id, label) UNIQUE constraint failed during an insert.
func isFolderLabelTaken(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: folders.users_id, folders.label")
}
