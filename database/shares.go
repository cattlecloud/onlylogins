// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"fmt"
	"time"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"github.com/shoenig/go-conceal"
)

type Shares interface {
	// metrics
	Count(scope.C) (int, error)

	// share operations
	Create(scope.C, *sitecontent.CreateShareRequest) (records.SharesUnique, error)
	Consume(scope.C, *sitecontent.ConsumeShareRequest) (*sitecontent.Share, error)
	Prune(scope.C, time.Duration) (int, error)
}

type sharesql struct {
	log    *ulog.Log
	ldb    *litesql.LiteDB
	cypher encrypt.Cypher
}

func (s *store) Shares(cypher encrypt.Cypher) Shares {
	return &sharesql{
		log:    ulog.New("database/shares"),
		ldb:    s.ldb,
		cypher: cypher,
	}
}

func (s *sharesql) Count(ctx scope.C) (int, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return litesql.TxFailure, xerr
	}
	defer done()

	const statement = `select count(uid) from shares`

	return litesql.QueryRow(ctx, tx, scanCount, statement)
}

func (s *sharesql) Create(ctx scope.C, request *sitecontent.CreateShareRequest) (records.SharesUnique, error) {
	uid := records.Unique[records.SharesUnique]()

	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return records.SharesUnique(records.None), xerr
	}
	defer done()

	const statement = `
	insert into shares (
		uid, users_id, content, ttl, creation
	) values (
		?, ?, ?, ?, ?
	)`

	seconds := request.SecondsTTL()
	now := request.Now.Unix()

	ciphertext, berr := s.cypher.Encrypt(request.Secret)
	if berr != nil {
		return "", fmt.Errorf("unable to encrypt share: %w", berr)
	}

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, statement,
		uid, request.User, ciphertext.Unveil(), seconds, now,
	); err != nil {
		return "", fmt.Errorf("unable to create share: %w", err)
	}

	return uid, tx.Commit()
}

func (s *sharesql) Consume(ctx scope.C, request *sitecontent.ConsumeShareRequest) (*sitecontent.Share, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return nil, xerr
	}
	defer done()

	const statement = `
	select
		uid,
		users_id,
		content,
		ttl,
		creation
	from shares
	where uid = ?`

	f := func(scan litesql.ScanFunc) (*sitecontent.Share, error) {
		share := new(sitecontent.Share)
		var ciphertext []byte
		var ttl, creation int64

		if err := scan(
			&share.UUID,
			&share.User,
			&ciphertext,
			&ttl,
			&creation,
		); err != nil {
			return nil, err
		}

		secret, serr := s.cypher.Decrypt(conceal.NewBytes(ciphertext))
		if serr != nil {
			return nil, fmt.Errorf("unable to decrypt share: %w", serr)
		}

		share.Secret = secret
		share.TTL = time.Duration(ttl) * time.Second
		share.Creation = time.Unix(creation, 0)

		return share, nil
	}

	share, qerr := litesql.QueryRow(ctx, tx, f, statement, request.UUID)
	if qerr != nil {
		return nil, fmt.Errorf("unable to get share: %w", qerr)
	}

	const deletion = `delete from shares where uid = ?`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOne, deletion, request.UUID); err != nil {
		return nil, fmt.Errorf("unable to delete share: %w", err)
	}

	return share, tx.Commit()
}

func (s *sharesql) Prune(ctx scope.C, age time.Duration) (int, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return litesql.TxFailure, xerr
	}
	defer done()

	cutoff := time.Now().Add(-age).Unix()

	const countSQL = `select count(uid) from shares where creation < ?`
	count, cerr := litesql.QueryRow(ctx, tx, scanCount, countSQL, cutoff)
	if cerr != nil {
		return litesql.ExecFailure, fmt.Errorf("unable to count shares to prune: %w", cerr)
	}

	const deleteSQL = `delete from shares where creation < ?`
	if err := s.ldb.Exec(ctx, tx, count, deleteSQL, cutoff); err != nil {
		return litesql.ExecFailure, fmt.Errorf("unable to prune shares: %w", err)
	}

	return count, tx.Commit()
}
