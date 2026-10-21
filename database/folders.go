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

	f := func(sf litesql.ScanFunc) (int, error) {
		var i int
		err := sf(&i)
		return i, err
	}

	return litesql.QueryRow(ctx, tx, f, statement)
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
		var folder sitecontent.Folder
		err := sf(&folder.ID, &folder.Label, &folder.Icon)
		return &folder, err
	}

	return litesql.QueryRows(ctx, tx, f, statement, user)
}
