// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"github.com/shoenig/go-conceal"
)

var (
	ErrDocumentLabelTaken = errors.New("document name already in use")
	ErrDocumentNotFound   = errors.New("document not found")
	ErrFolderNotFound     = errors.New("folder not found")
)

type Documents interface {
	// metrics
	Count(scope.C) (int, error)

	// document operations
	Create(scope.C, *sitecontent.CreateDocumentRequest) (records.DocumentsAuto, error)
	List(scope.C, *sitecontent.ListDocumentsRequest) ([]*sitecontent.DocumentListing, error)
	Get(scope.C, *sitecontent.GetDocumentRequest) (*sitecontent.Document, error)

	// document field operations
	CreateWeblink(scope.C, *sitecontent.CreateWeblinkRequest) (records.WeblinksAuto, error)
	UpdateWeblink(scope.C, *sitecontent.UpdateWeblinkRequest) error
	DeleteWeblink(scope.C, *sitecontent.DeleteWeblinkRequest) error

	CreateUsername(scope.C, *sitecontent.CreateUsernameRequest) (records.UsernamesAuto, error)
	UpdateUsername(scope.C, *sitecontent.UpdateUsernameRequest) error
	DeleteUsername(scope.C, *sitecontent.DeleteUsernameRequest) error
}

type documentsql struct {
	log *ulog.Log
	ldb *litesql.LiteDB
}

func (s *store) Documents() Documents {
	return &documentsql{
		log: ulog.New("database/documents"),
		ldb: s.ldb,
	}
}

func (s *documentsql) Count(ctx scope.C) (int, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return 0, fmt.Errorf("unable to count documents: %w", xerr)
	}
	defer done()

	const statement = `select count(id) from documents`

	return litesql.QueryRow(ctx, tx, scanCount, statement)
}

func (s *documentsql) Create(ctx scope.C, request *sitecontent.CreateDocumentRequest) (records.DocumentsAuto, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return litesql.TxFailure, xerr
	}
	defer done()

	const statement = `
	insert into documents (
		folders_id,
		label
	) select id, ?
	from folders
	where
		users_id = ? and
		label = ?`

	id, rerr := s.ldb.ExecID(ctx, tx, statement,
		request.Label,
		request.User,
		request.Folder,
	)
	switch {
	case isDocumentLabelTaken(rerr):
		return litesql.ExecFailure, fmt.Errorf("unable to create document: %w", ErrDocumentLabelTaken)
	case isFolderMissing(rerr):
		return litesql.ExecFailure, fmt.Errorf("unable to create document: %w", ErrFolderNotFound)
	case rerr != nil:
		s.log.E.Fmt("unable to insert document: %v", rerr)
		return litesql.ExecFailure, fmt.Errorf("unable to create document: %w", rerr)
	}

	return records.DocumentsAuto(id), tx.Commit()
}

func isDocumentLabelTaken(err error) bool {
	return err != nil && strings.Contains(
		err.Error(),
		"UNIQUE constraint failed: documents.folders_id, documents.label",
	)
}

func isFolderMissing(err error) bool {
	return err != nil && strings.Contains(err.Error(), "expected to affect 1 row")
}

func (s *documentsql) List(ctx scope.C, request *sitecontent.ListDocumentsRequest) ([]*sitecontent.DocumentListing, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to list documents: %w", xerr)
	}
	defer done()

	const statement = `
	select
		d.id,
		d.label
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.label = ?
	order by d.modification desc`

	f := func(sf litesql.ScanFunc) (*sitecontent.DocumentListing, error) {
		listing := new(sitecontent.DocumentListing)
		err := sf(&listing.ID, &listing.Label)
		return listing, err
	}

	return litesql.QueryRows(ctx, tx, f, statement, request.User, request.Folder)
}

func (s *documentsql) Get(ctx scope.C, request *sitecontent.GetDocumentRequest) (*sitecontent.Document, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to get document: %w", xerr)
	}
	defer done()

	const docStatement = `
	select
		d.id,
		d.label
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.label = ? and
		d.label = ?`

	type docRow struct {
		ID    records.DocumentsAuto
		Label string
	}
	scanDoc := func(sf litesql.ScanFunc) (docRow, error) {
		var row docRow
		err := sf(&row.ID, &row.Label)
		return row, err
	}

	doc, derr := litesql.QueryRow(ctx, tx, scanDoc, docStatement,
		request.User, request.Folder, request.Document,
	)
	switch {
	case errors.Is(derr, sql.ErrNoRows):
		return nil, fmt.Errorf("unable to get document: %w", ErrDocumentNotFound)
	case derr != nil:
		return nil, fmt.Errorf("unable to get document: %w", derr)
	}

	const weblinksStatement = `
	select id, v
	from weblinks
	where did = ?
	order by id asc`
	scanWeblink := func(sf litesql.ScanFunc) (*sitecontent.Weblink, error) {
		w := new(sitecontent.Weblink)
		var v []byte
		err := sf(&w.ID, &v)
		w.V = conceal.NewBytes(v)
		return w, err
	}
	weblinks, werr := litesql.QueryRows(ctx, tx, scanWeblink, weblinksStatement, doc.ID)
	if werr != nil {
		return nil, fmt.Errorf("unable to get weblinks: %w", werr)
	}

	const usernamesStatement = `
	select id, v
	from usernames
	where did = ?
	order by id asc`
	scanUsername := func(sf litesql.ScanFunc) (*sitecontent.Username, error) {
		u := new(sitecontent.Username)
		var v []byte
		err := sf(&u.ID, &v)
		u.V = conceal.NewBytes(v)
		return u, err
	}
	usernames, uerr := litesql.QueryRows(ctx, tx, scanUsername, usernamesStatement, doc.ID)
	if uerr != nil {
		return nil, fmt.Errorf("unable to get usernames: %w", uerr)
	}

	const passwordsStatement = `
	select id, v
	from passwords
	where did = ?
	order by id asc`
	scanPassword := func(sf litesql.ScanFunc) (*sitecontent.Password, error) {
		p := new(sitecontent.Password)
		var v []byte
		err := sf(&p.ID, &v)
		p.V = conceal.NewBytes(v)
		return p, err
	}
	passwords, perr := litesql.QueryRows(ctx, tx, scanPassword, passwordsStatement, doc.ID)
	if perr != nil {
		return nil, fmt.Errorf("unable to get passwords: %w", perr)
	}

	const keyvalsStatement = `
	select id, k, v
	from keyvals
	where did = ?
	order by id asc`
	scanKeyval := func(sf litesql.ScanFunc) (*sitecontent.Keyval, error) {
		kv := new(sitecontent.Keyval)
		var k string
		var v []byte
		err := sf(&kv.ID, &k, &v)
		kv.K = conceal.New(k)
		kv.V = conceal.NewBytes(v)
		return kv, err
	}
	keyvals, kerr := litesql.QueryRows(ctx, tx, scanKeyval, keyvalsStatement, doc.ID)
	if kerr != nil {
		return nil, fmt.Errorf("unable to get keyvals: %w", kerr)
	}

	const notesStatement = `
	select id, v
	from notes
	where did = ?
	order by id asc`
	scanNote := func(sf litesql.ScanFunc) (*sitecontent.Note, error) {
		n := new(sitecontent.Note)
		var v []byte
		err := sf(&n.ID, &v)
		n.V = conceal.NewBytes(v)
		return n, err
	}
	notes, nerr := litesql.QueryRows(ctx, tx, scanNote, notesStatement, doc.ID)
	if nerr != nil {
		return nil, fmt.Errorf("unable to get notes: %w", nerr)
	}

	return &sitecontent.Document{
		ID:        doc.ID,
		Label:     doc.Label,
		Folder:    request.Folder,
		Weblinks:  weblinks,
		Usernames: usernames,
		Passwords: passwords,
		Keyvals:   keyvals,
		Notes:     notes,
	}, nil
}

func (s *documentsql) CreateWeblink(ctx scope.C, request *sitecontent.CreateWeblinkRequest) (records.WeblinksAuto, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return litesql.TxFailure, xerr
	}
	defer done()

	const statement = `
	insert into weblinks (did, v)
	select d.id, ?
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.label = ? and
		d.label = ?`

	id, rerr := s.ldb.ExecID(ctx, tx, statement,
		request.Encrypted.Unveil(),
		request.User,
		request.Folder,
		request.Document,
	)
	switch {
	case isFolderMissing(rerr):
		return litesql.ExecFailure, fmt.Errorf("unable to create weblink: %w", ErrDocumentNotFound)
	case rerr != nil:
		s.log.E.Fmt("unable to insert weblink: %v", rerr)
		return litesql.ExecFailure, fmt.Errorf("unable to create weblink: %w", rerr)
	}

	return records.WeblinksAuto(id), tx.Commit()
}

func (s *documentsql) UpdateWeblink(ctx scope.C, request *sitecontent.UpdateWeblinkRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to update weblink: %w", xerr)
	}
	defer done()

	const statement = `
	update weblinks
	set v = ?
	where id = ? and did in (
		select d.id
		from documents d
		join folders f on f.id = d.folders_id
		where
			f.users_id = ? and
			f.label = ? and
			d.label = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.Encrypted.Unveil(),
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
		if isFolderMissing(rerr) {
			return fmt.Errorf("unable to update weblink: %w", ErrDocumentNotFound)
		}
		s.log.E.Fmt("unable to update weblink: %v", rerr)
		return fmt.Errorf("unable to update weblink: %w", rerr)
	}

	return tx.Commit()
}

func (s *documentsql) DeleteWeblink(ctx scope.C, request *sitecontent.DeleteWeblinkRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to delete weblink: %w", xerr)
	}
	defer done()

	const statement = `
	delete from weblinks
	where id = ? and did in (
		select d.id
		from documents d
		join folders f on f.id = d.folders_id
		where
			f.users_id = ? and
			f.label = ? and
			d.label = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
		if isFolderMissing(rerr) {
			return fmt.Errorf("unable to delete weblink: %w", ErrDocumentNotFound)
		}
		s.log.E.Fmt("unable to delete weblink: %v", rerr)
		return fmt.Errorf("unable to delete weblink: %w", rerr)
	}

	return tx.Commit()
}

func (s *documentsql) CreateUsername(ctx scope.C, request *sitecontent.CreateUsernameRequest) (records.UsernamesAuto, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return litesql.TxFailure, xerr
	}
	defer done()

	const statement = `
	insert into usernames (did, v)
	select d.id, ?
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.label = ? and
		d.label = ?`

	id, rerr := s.ldb.ExecID(ctx, tx, statement,
		request.Encrypted.Unveil(),
		request.User,
		request.Folder,
		request.Document,
	)
	switch {
	case isFolderMissing(rerr):
		return litesql.ExecFailure, fmt.Errorf("unable to create username: %w", ErrDocumentNotFound)
	case rerr != nil:
		s.log.E.Fmt("unable to insert username: %v", rerr)
		return litesql.ExecFailure, fmt.Errorf("unable to create username: %w", rerr)
	}

	return records.UsernamesAuto(id), tx.Commit()
}

func (s *documentsql) UpdateUsername(ctx scope.C, request *sitecontent.UpdateUsernameRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to update username: %w", xerr)
	}
	defer done()

	const statement = `
	update usernames
	set v = ?
	where id = ? and did in (
		select d.id
		from documents d
		join folders f on f.id = d.folders_id
		where
			f.users_id = ? and
			f.label = ? and
			d.label = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.Encrypted.Unveil(),
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
		if isFolderMissing(rerr) {
			return fmt.Errorf("unable to update username: %w", ErrDocumentNotFound)
		}
		s.log.E.Fmt("unable to update username: %v", rerr)
		return fmt.Errorf("unable to update username: %w", rerr)
	}

	return tx.Commit()
}

func (s *documentsql) DeleteUsername(ctx scope.C, request *sitecontent.DeleteUsernameRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to delete username: %w", xerr)
	}
	defer done()

	const statement = `
	delete from usernames
	where id = ? and did in (
		select d.id
		from documents d
		join folders f on f.id = d.folders_id
		where
			f.users_id = ? and
			f.label = ? and
			d.label = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
		if isFolderMissing(rerr) {
			return fmt.Errorf("unable to delete username: %w", ErrDocumentNotFound)
		}
		s.log.E.Fmt("unable to delete username: %v", rerr)
		return fmt.Errorf("unable to delete username: %w", rerr)
	}

	return tx.Commit()
}
