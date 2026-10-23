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
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
)

var (
	ErrDocumentLabelTaken = errors.New("document name is already in use")
	ErrFolderNotFound     = errors.New("folder not found")
)

type Documents interface {
	// metrics
	Count(scope.C) (int, error)

	// document operations
	Create(scope.C, *sitecontent.CreateDocumentRequest) (records.DocumentsAuto, error)
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

// isFolderMissing reports whether err indicates that the subquery INSERT
// affected zero rows because no folder matched (users_id, label) for
// the requesting user.
func isFolderMissing(err error) bool {
	return err != nil && strings.Contains(
		err.Error(),
		"litesql: expected to affect 1 row, actual was 0",
	)
}
