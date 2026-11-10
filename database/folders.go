// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

type Folders interface {
	// metrics
	Count(scope.C) (int, error)

	// folder operations
	Create(scope.C, *sitecontent.CreateFolderRequest) error
	List(scope.C, users.Identity) ([]*sitecontent.Folder, error)
	Delete(scope.C, *sitecontent.DeleteFolderRequest) error
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
	); ferr != nil {
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

func (s *foldersql) Delete(ctx scope.C, request *sitecontent.DeleteFolderRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to delete folder: %w", xerr)
	}
	defer done()

	// refuse to delete system-protected rows (e.g. the default "miscellaneous"
	// folder with protected=1); the where clause ensures we never touch them
	// even if the caller tries
	const statement = `
	delete from folders
	where
		users_id = ? and
		label = ? and
		protected = 0`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.User,
		request.Label,
	); rerr != nil {
		s.log.E.Fmt("unable to delete folder: %v", rerr)
		return fmt.Errorf("unable to delete folder: %w", rerr)
	}

	return tx.Commit()
}
