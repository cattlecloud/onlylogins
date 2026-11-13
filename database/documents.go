// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"fmt"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"github.com/shoenig/go-conceal"
)

type Documents interface {
	// metrics
	Count(scope.C) (int, error)

	// document operations
	Create(scope.C, *sitecontent.CreateDocumentRequest) (records.DocumentsAuto, error)
	List(scope.C, *sitecontent.ListDocumentsRequest) ([]*sitecontent.DocumentListing, error)
	Get(scope.C, *sitecontent.GetDocumentRequest) (*sitecontent.Document, error)
	Resolve(scope.C, *sitecontent.GetDocumentRequest) (records.DocumentsAuto, error)
	Delete(scope.C, *sitecontent.DeleteDocumentRequest) error

	// document > weblinks
	CreateWeblink(scope.C, *sitecontent.CreateWeblinkRequest) (records.WeblinksAuto, error)
	GetWeblink(scope.C, *sitecontent.GetWeblinkRequest) (*sitecontent.Weblink, error)
	UpdateWeblink(scope.C, *sitecontent.UpdateWeblinkRequest) error
	DeleteWeblink(scope.C, *sitecontent.DeleteWeblinkRequest) error

	// document > usernames
	CreateUsername(scope.C, *sitecontent.CreateUsernameRequest) (records.UsernamesAuto, error)
	GetUsername(scope.C, *sitecontent.GetUsernameRequest) (*sitecontent.Username, error)
	UpdateUsername(scope.C, *sitecontent.UpdateUsernameRequest) error
	DeleteUsername(scope.C, *sitecontent.DeleteUsernameRequest) error

	// document > passwords
	CreatePassword(scope.C, *sitecontent.CreatePasswordRequest) (records.PasswordsAuto, error)
	GetPassword(scope.C, *sitecontent.GetPasswordRequest) (*sitecontent.Password, error)
	UpdatePassword(scope.C, *sitecontent.UpdatePasswordRequest) error
	DeletePassword(scope.C, *sitecontent.DeletePasswordRequest) error

	// document > keyvals
	CreateKeyval(scope.C, *sitecontent.CreateKeyvalRequest) (records.KeyvalsAuto, error)
	GetKeyval(scope.C, *sitecontent.GetKeyvalRequest) (*sitecontent.Keyval, error)
	UpdateKeyval(scope.C, *sitecontent.UpdateKeyvalRequest) error
	DeleteKeyval(scope.C, *sitecontent.DeleteKeyvalRequest) error

	// document > note
	CreateNote(scope.C, *sitecontent.CreateNoteRequest) (records.NotesAuto, error)
	GetNote(scope.C, *sitecontent.GetNoteRequest) (*sitecontent.Note, error)
	UpdateNote(scope.C, *sitecontent.UpdateNoteRequest) error
	DeleteNote(scope.C, *sitecontent.DeleteNoteRequest) error
}

type documentsql struct {
	log *ulog.Log
	enf *enforcement
	ldb *litesql.LiteDB
}

func (s *store) Documents() Documents {
	return &documentsql{
		log: ulog.New("database/documents"),
		enf: enforcements(s.ldb),
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

	// enforce quota on number of documents
	if err := s.enf.quotas(ctx, tx, request.User); err != nil {
		return litesql.ExecFailure, err
	}

	const statement = `
	insert into documents (
		folders_id,
		label,
		slug
	) select id, ?, ?
	from folders
	where
		users_id = ? and
		slug = ?`

	id, rerr := s.ldb.ExecID(ctx, tx, statement,
		request.Label,
		request.Slug,
		request.User,
		request.Folder,
	)
	if rerr != nil {
		s.log.E.Fmt("unable to insert document: %v", rerr)
		return litesql.ExecFailure, fmt.Errorf("unable to create document: %w", rerr)
	}

	return records.DocumentsAuto(id), tx.Commit()
}

func (s *documentsql) Resolve(ctx scope.C, request *sitecontent.GetDocumentRequest) (records.DocumentsAuto, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return 0, fmt.Errorf("unable to resolve document: %w", xerr)
	}
	defer done()

	const statement = `
	select d.id
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?`

	f := func(scan litesql.ScanFunc) (records.DocumentsAuto, error) {
		var id records.DocumentsAuto
		err := scan(&id)
		return id, err
	}

	return litesql.QueryRow(ctx, tx, f, statement,
		request.User, request.Folder, request.Document,
	)
}

func (s *documentsql) Delete(ctx scope.C, request *sitecontent.DeleteDocumentRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return xerr
	}
	defer done()

	const statement = `
	delete from documents
	where id = ? and exists (
		select 1
		from folders f
		where
			f.id = documents.folders_id and
			f.users_id = ?
	)`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.ID,
		request.User,
	); err != nil {
		s.log.E.Fmt("unable to delete document: %v", err)
		return fmt.Errorf("unable to delete document: %w", err)
	}

	return tx.Commit()
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
		d.label,
		d.slug
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.slug = ?
	order by d.modification desc`

	f := func(scan litesql.ScanFunc) (*sitecontent.DocumentListing, error) {
		listing := new(sitecontent.DocumentListing)
		err := scan(&listing.ID, &listing.Label, &listing.Slug)
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
		d.label,
		d.slug,
		f.label,
		f.slug
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?`

	type docRow struct {
		ID         records.DocumentsAuto
		Label      string
		Slug       string
		Folder     string
		FolderSlug string
	}
	scanDoc := func(scan litesql.ScanFunc) (docRow, error) {
		var row docRow
		err := scan(&row.ID, &row.Label, &row.Slug, &row.Folder, &row.FolderSlug)
		return row, err
	}

	doc, derr := litesql.QueryRow(ctx, tx, scanDoc, docStatement,
		request.User, request.Folder, request.Document,
	)
	if derr != nil {
		return nil, fmt.Errorf("unable to get document: %w", derr)
	}

	const weblinksStatement = `
	select id, v
	from weblinks
	where did = ?
	order by id asc`
	scanWeblink := func(scan litesql.ScanFunc) (*sitecontent.Weblink, error) {
		w := new(sitecontent.Weblink)
		var v []byte
		err := scan(&w.ID, &v)
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
	scanUsername := func(scan litesql.ScanFunc) (*sitecontent.Username, error) {
		u := new(sitecontent.Username)
		var v []byte
		err := scan(&u.ID, &v)
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
	scanPassword := func(scan litesql.ScanFunc) (*sitecontent.Password, error) {
		p := new(sitecontent.Password)
		var v []byte
		err := scan(&p.ID, &v)
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
	scanKeyval := func(scan litesql.ScanFunc) (*sitecontent.Keyval, error) {
		kv := new(sitecontent.Keyval)
		var k string
		var v []byte
		err := scan(&kv.ID, &k, &v)
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
	order by id asc
	limit 1`
	scanNote := func(scan litesql.ScanFunc) (*sitecontent.Note, error) {
		n := new(sitecontent.Note)
		var v []byte
		err := scan(&n.ID, &v)
		n.V = conceal.NewBytes(v)
		return n, err
	}
	notes, nerr := litesql.QueryRows(ctx, tx, scanNote, notesStatement, doc.ID)
	if nerr != nil {
		return nil, fmt.Errorf("unable to get notes: %w", nerr)
	}

	var note *sitecontent.Note
	if len(notes) > 0 {
		note = notes[0]
	}

	return &sitecontent.Document{
		ID:         doc.ID,
		Label:      doc.Label,
		Slug:       doc.Slug,
		Folder:     doc.Folder,
		FolderSlug: doc.FolderSlug,
		Weblinks:   weblinks,
		Usernames:  usernames,
		Passwords:  passwords,
		Keyvals:    keyvals,
		Note:       note,
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
		f.slug = ? and
		d.slug = ?`

	id, rerr := s.ldb.ExecID(ctx, tx, statement,
		request.Encrypted.Unveil(),
		request.User,
		request.Folder,
		request.Document,
	)
	if rerr != nil {
		s.log.E.Fmt("unable to insert weblink: %v", rerr)
		return litesql.ExecFailure, fmt.Errorf("unable to create weblink: %w", rerr)
	}

	return records.WeblinksAuto(id), tx.Commit()
}

func (s *documentsql) GetWeblink(ctx scope.C, request *sitecontent.GetWeblinkRequest) (*sitecontent.Weblink, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to get weblink: %w", xerr)
	}
	defer done()

	const statement = `
	select w.id, w.v
	from weblinks w
	join documents d on d.id = w.did
	join folders f on f.id = d.folders_id
	where
		w.id = ? and
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?
	limit 1`

	scanWeblink := func(scan litesql.ScanFunc) (*sitecontent.Weblink, error) {
		w := new(sitecontent.Weblink)
		var v []byte
		err := scan(&w.ID, &v)
		w.V = conceal.NewBytes(v)
		return w, err
	}
	rows, werr := litesql.QueryRows(ctx, tx, scanWeblink, statement,
		request.ID, request.User, request.Folder, request.Document,
	)
	if werr != nil {
		return nil, fmt.Errorf("unable to get weblink: %w", werr)
	}
	if len(rows) == 0 {
		return nil, nil //nolint:nilnil
	}
	return rows[0], nil
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
			f.slug = ? and
			d.slug = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.Encrypted.Unveil(),
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
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
			f.slug = ? and
			d.slug = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
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
	select
		d.id, ?
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?`

	id, rerr := s.ldb.ExecID(ctx, tx, statement,
		request.Encrypted.Unveil(),
		request.User,
		request.Folder,
		request.Document,
	)
	if rerr != nil {
		s.log.E.Fmt("unable to insert username: %v", rerr)
		return litesql.ExecFailure, fmt.Errorf("unable to create username: %w", rerr)
	}

	return records.UsernamesAuto(id), tx.Commit()
}

func (s *documentsql) GetUsername(ctx scope.C, request *sitecontent.GetUsernameRequest) (*sitecontent.Username, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to get username: %w", xerr)
	}
	defer done()

	const statement = `
	select u.id, u.v
	from usernames u
	join documents d on d.id = u.did
	join folders f on f.id = d.folders_id
	where
		u.id = ? and
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?
	limit 1`

	scanUsername := func(scan litesql.ScanFunc) (*sitecontent.Username, error) {
		u := new(sitecontent.Username)
		var v []byte
		err := scan(&u.ID, &v)
		u.V = conceal.NewBytes(v)
		return u, err
	}
	rows, uerr := litesql.QueryRows(ctx, tx, scanUsername, statement,
		request.ID, request.User, request.Folder, request.Document,
	)
	if uerr != nil {
		return nil, fmt.Errorf("unable to get username: %w", uerr)
	}
	if len(rows) == 0 {
		return nil, nil //nolint:nilnil
	}
	return rows[0], nil
}

func (s *documentsql) UpdateUsername(ctx scope.C, request *sitecontent.UpdateUsernameRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to update username: %w", xerr)
	}
	defer done()

	const statement = `
	update usernames set
		v = ?
	where
		id = ?
	and did in (
	select d.id
	from documents d
	join folders f on f.id = d.folders_id
		where
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.Encrypted.Unveil(),
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
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
	where
		id = ? and did in (
	select d.id
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
		s.log.E.Fmt("unable to delete username: %v", rerr)
		return fmt.Errorf("unable to delete username: %w", rerr)
	}

	return tx.Commit()
}

func (s *documentsql) CreatePassword(ctx scope.C, request *sitecontent.CreatePasswordRequest) (records.PasswordsAuto, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return litesql.TxFailure, xerr
	}
	defer done()

	const statement = `
	insert into passwords (did, v)
	select d.id, ?
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?`

	id, rerr := s.ldb.ExecID(ctx, tx, statement,
		request.Encrypted.Unveil(),
		request.User,
		request.Folder,
		request.Document,
	)
	if rerr != nil {
		s.log.E.Fmt("unable to insert password: %v", rerr)
		return litesql.ExecFailure, fmt.Errorf("unable to create password: %w", rerr)
	}

	return records.PasswordsAuto(id), tx.Commit()
}

func (s *documentsql) GetPassword(ctx scope.C, request *sitecontent.GetPasswordRequest) (*sitecontent.Password, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to get password: %w", xerr)
	}
	defer done()

	const statement = `
	select p.id, p.v
	from passwords p
	join documents d on d.id = p.did
	join folders f on f.id = d.folders_id
	where
		p.id = ? and
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?
	limit 1`

	scanPassword := func(scan litesql.ScanFunc) (*sitecontent.Password, error) {
		p := new(sitecontent.Password)
		var v []byte
		err := scan(&p.ID, &v)
		p.V = conceal.NewBytes(v)
		return p, err
	}
	rows, perr := litesql.QueryRows(ctx, tx, scanPassword, statement,
		request.ID, request.User, request.Folder, request.Document,
	)
	if perr != nil {
		return nil, fmt.Errorf("unable to get password: %w", perr)
	}
	if len(rows) == 0 {
		return nil, nil //nolint:nilnil
	}
	return rows[0], nil
}

func (s *documentsql) UpdatePassword(ctx scope.C, request *sitecontent.UpdatePasswordRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to update password: %w", xerr)
	}
	defer done()

	const statement = `
	update passwords
	set v = ?
	where
		id = ? and did in (
	select d.id
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.Encrypted.Unveil(),
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
		s.log.E.Fmt("unable to update password: %v", rerr)
		return fmt.Errorf("unable to update password: %w", rerr)
	}

	return tx.Commit()
}

func (s *documentsql) DeletePassword(ctx scope.C, request *sitecontent.DeletePasswordRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to delete password: %w", xerr)
	}
	defer done()

	const statement = `
	delete from passwords
	where
		id = ? and did in (
	select d.id
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
		s.log.E.Fmt("unable to delete password: %v", rerr)
		return fmt.Errorf("unable to delete password: %w", rerr)
	}

	return tx.Commit()
}

func (s *documentsql) CreateNote(ctx scope.C, request *sitecontent.CreateNoteRequest) (records.NotesAuto, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return litesql.TxFailure, xerr
	}
	defer done()

	const statement = `
	insert into notes (did, v)
	select d.id, ?
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?`

	id, rerr := s.ldb.ExecID(ctx, tx, statement,
		request.Encrypted.Unveil(),
		request.User,
		request.Folder,
		request.Document,
	)
	if rerr != nil {
		s.log.E.Fmt("unable to insert note: %v", rerr)
		return litesql.ExecFailure, fmt.Errorf("unable to create note: %w", rerr)
	}

	return records.NotesAuto(id), tx.Commit()
}

func (s *documentsql) GetNote(ctx scope.C, request *sitecontent.GetNoteRequest) (*sitecontent.Note, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to get note: %w", xerr)
	}
	defer done()

	const statement = `
	select n.id, n.v
	from notes n
	join documents d on d.id = n.did
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?
	limit 1`

	scanNote := func(scan litesql.ScanFunc) (*sitecontent.Note, error) {
		n := new(sitecontent.Note)
		var v []byte
		err := scan(&n.ID, &v)
		n.V = conceal.NewBytes(v)
		return n, err
	}
	rows, nerr := litesql.QueryRows(ctx, tx, scanNote, statement,
		request.User, request.Folder, request.Document,
	)
	if nerr != nil {
		return nil, fmt.Errorf("unable to get note: %w", nerr)
	}
	if len(rows) == 0 {
		return nil, nil //nolint:nilnil // note is optional
	}
	return rows[0], nil
}

func (s *documentsql) UpdateNote(ctx scope.C, request *sitecontent.UpdateNoteRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to update note: %w", xerr)
	}
	defer done()

	const statement = `
	update notes
	set v = ?
	where id = ? and did in (
		select d.id
		from documents d
		join folders f on f.id = d.folders_id
		where
			f.users_id = ? and
			f.slug = ? and
			d.slug = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.Encrypted.Unveil(),
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
		s.log.E.Fmt("unable to update note: %v", rerr)
		return fmt.Errorf("unable to update note: %w", rerr)
	}

	return tx.Commit()
}

func (s *documentsql) DeleteNote(ctx scope.C, request *sitecontent.DeleteNoteRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to delete note: %w", xerr)
	}
	defer done()

	const statement = `
	delete from notes
	where id = ? and did in (
		select d.id
		from documents d
		join folders f on f.id = d.folders_id
		where
			f.users_id = ? and
			f.slug = ? and
			d.slug = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
		s.log.E.Fmt("unable to delete note: %v", rerr)
		return fmt.Errorf("unable to delete note: %w", rerr)
	}

	return tx.Commit()
}

func (s *documentsql) CreateKeyval(ctx scope.C, request *sitecontent.CreateKeyvalRequest) (records.KeyvalsAuto, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return litesql.TxFailure, xerr
	}
	defer done()

	const statement = `
	insert into keyvals (did, k, v)
	select d.id, ?, ?
	from documents d
	join folders f on f.id = d.folders_id
	where
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?`

	id, rerr := s.ldb.ExecID(ctx, tx, statement,
		request.Key,
		request.Encrypted.Unveil(),
		request.User,
		request.Folder,
		request.Document,
	)
	if rerr != nil {
		s.log.E.Fmt("unable to insert keyval: %v", rerr)
		return litesql.ExecFailure, fmt.Errorf("unable to create keyval: %w", rerr)
	}

	return records.KeyvalsAuto(id), tx.Commit()
}

func (s *documentsql) GetKeyval(ctx scope.C, request *sitecontent.GetKeyvalRequest) (*sitecontent.Keyval, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to get keyval: %w", xerr)
	}
	defer done()

	const statement = `
	select k.id, k.k, k.v
	from keyvals k
	join documents d on d.id = k.did
	join folders f on f.id = d.folders_id
	where
		k.id = ? and
		f.users_id = ? and
		f.slug = ? and
		d.slug = ?
	limit 1`

	scanKeyval := func(scan litesql.ScanFunc) (*sitecontent.Keyval, error) {
		kv := new(sitecontent.Keyval)
		var k string
		var v []byte
		err := scan(&kv.ID, &k, &v)
		kv.K = conceal.New(k)
		kv.V = conceal.NewBytes(v)
		return kv, err
	}
	rows, kerr := litesql.QueryRows(ctx, tx, scanKeyval, statement,
		request.ID, request.User, request.Folder, request.Document,
	)
	if kerr != nil {
		return nil, fmt.Errorf("unable to get keyval: %w", kerr)
	}
	if len(rows) == 0 {
		return nil, nil //nolint:nilnil
	}
	return rows[0], nil
}

func (s *documentsql) UpdateKeyval(ctx scope.C, request *sitecontent.UpdateKeyvalRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to update keyval: %w", xerr)
	}
	defer done()

	const statement = `
	update keyvals
	set k = ?, v = ?
	where id = ? and did in (
		select d.id
		from documents d
		join folders f on f.id = d.folders_id
		where
			f.users_id = ? and
			f.slug = ? and
			d.slug = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.Key,
		request.Encrypted.Unveil(),
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
		s.log.E.Fmt("unable to update keyval: %v", rerr)
		return fmt.Errorf("unable to update keyval: %w", rerr)
	}

	return tx.Commit()
}

func (s *documentsql) DeleteKeyval(ctx scope.C, request *sitecontent.DeleteKeyvalRequest) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to delete keyval: %w", xerr)
	}
	defer done()

	const statement = `
	delete from keyvals
	where id = ? and did in (
		select d.id
		from documents d
		join folders f on f.id = d.folders_id
		where
			f.users_id = ? and
			f.slug = ? and
			d.slug = ?
	)`

	if rerr := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		request.ID,
		request.User,
		request.Folder,
		request.Document,
	); rerr != nil {
		s.log.E.Fmt("unable to delete keyval: %v", rerr)
		return fmt.Errorf("unable to delete keyval: %w", rerr)
	}

	return tx.Commit()
}
