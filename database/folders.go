// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"errors"
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
)

// ErrFolderExists indicates a folder with the given label/slug already exists
// for the user, so the rename cannot be applied.
var ErrFolderExists = errors.New("a folder with that name already exists")

type Folders interface {
	// metrics
	Count(scope.C) (int, error)

	// folder operations
	Create(scope.C, *sitecontent.CreateFolderRequest) error
	List(scope.C, users.Identity) ([]*sitecontent.Folder, error)
	Get(scope.C, users.Identity, string) (*sitecontent.Folder, error)
	Rename(scope.C, *sitecontent.RenameFolderRequest) error
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
		users_id, label, slug, icon
	) values ( ?, ?, ?, ? )`

	if _, ferr := s.ldb.ExecID(ctx, tx, statement,
		request.User,
		request.Label,
		request.Slug,
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
		slug,
		icon,
		protected
	from folders
	where
		users_id = ?
	order by slug asc`

	f := func(scan litesql.ScanFunc) (*sitecontent.Folder, error) {
		folder := new(sitecontent.Folder)
		err := scan(&folder.ID, &folder.Label, &folder.Slug, &folder.Icon, &folder.Protected)
		return folder, err
	}

	return litesql.QueryRows(ctx, tx, f, statement, user)
}

func (s *foldersql) Get(ctx scope.C, user users.Identity, slug string) (*sitecontent.Folder, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to get folder: %w", xerr)
	}
	defer done()

	const statement = `
	select
		id,
		label,
		slug,
		icon,
		protected
	from folders
	where
		users_id = ? and
		slug = ?
	limit 1`

	f := func(scan litesql.ScanFunc) (*sitecontent.Folder, error) {
		folder := new(sitecontent.Folder)
		err := scan(&folder.ID, &folder.Label, &folder.Slug, &folder.Icon, &folder.Protected)
		return folder, err
	}

	return litesql.QueryRow(ctx, tx, f, statement, user, slug)
}

func (s *foldersql) Rename(ctx scope.C, request *sitecontent.RenameFolderRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to rename folder: %w", xerr)
	}
	defer done()

	// locate the folder to rename, refusing system-protected rows (i.e. the
	// default "miscellaneous" folder with protected=1)
	const findSQL = `
	select id
	from folders
	where
		users_id = ? and
		slug = ? and
		protected = 0
	limit 1`

	findID := func(scan litesql.ScanFunc) (records.FoldersAuto, error) {
		var id records.FoldersAuto
		err := scan(&id)
		return id, err
	}

	id, ferr := litesql.QueryRow(ctx, tx, findID, findSQL, request.User, request.Slug)
	if ferr != nil {
		s.log.E.Fmt("unable to rename folder: %v", ferr)
		return fmt.Errorf("unable to rename folder: %w", ferr)
	}

	// refuse to rename onto an existing folder; a folder with the same slug
	// (excluding this folder) means the name is already taken
	const conflictSQL = `
	select count(id)
	from folders
	where
		users_id = ? and
		(label = ? or slug = ?) and
		id != ?`

	count, cerr := litesql.QueryRow(ctx, tx, scanCount, conflictSQL,
		request.User, request.Label, request.NewSlug, id,
	)
	if cerr != nil {
		s.log.E.Fmt("unable to rename folder: %v", cerr)
		return fmt.Errorf("unable to rename folder: %w", cerr)
	}

	if count > 0 {
		return fmt.Errorf("unable to rename folder: %w", ErrFolderExists)
	}

	const updateSQL = `
	update folders
	set
		label = ?,
		slug = ?
	where id = ?`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, updateSQL,
		request.Label,
		request.NewSlug,
		id,
	); rerr != nil {
		s.log.E.Fmt("unable to rename folder: %v", rerr)
		return fmt.Errorf("unable to rename folder: %w", rerr)
	}

	return tx.Commit()
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
		slug = ? and
		protected = 0`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.User,
		request.Slug,
	); rerr != nil {
		s.log.E.Fmt("unable to delete folder: %v", rerr)
		return fmt.Errorf("unable to delete folder: %w", rerr)
	}

	return tx.Commit()
}
