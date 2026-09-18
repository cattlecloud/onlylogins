package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"cattlecloud.net/onlylogins/pkg/users"
	"cattlecloud.net/onlylogins/pkg/users/oauths"
)

type Users interface {
	// metrics
	Count(scope.C) (int, error)

	// user management
	Create(scope.C, *users.CreateUserRequest) (users.Identity, error)
	Exists(scope.C, oauths.IDP, oauths.Subject) (users.Identity, bool, error)
	Info(scope.C, users.Identity) (*sitecontent.UserInfo, error)
	List(scope.C) ([]*sitecontent.UserInfo, error)
	SetUsername(scope.C, users.Identity, string) error
	ListUsernameChanges(scope.C, users.Identity) ([]*sitecontent.NameChange, error)
	Profile(scope.C, users.Identity) (*sitecontent.UserProfile, error)
	ProfileSetShowEmail(scope.C, users.Identity, bool) error
	ProfileSetLocation(scope.C, users.Identity, string) error
	ProfileSetLink(scope.C, users.Identity, string) error
	ProfileSetAbout(scope.C, users.Identity, string) error
}

type usersql struct {
	log *ulog.Log
	ldb *litesql.LiteDB
}

func (s *store) Users() Users {
	return &usersql{
		log: ulog.New("database/users"),
		ldb: s.ldb,
	}
}

func scanUserInfo(sf litesql.ScanFunc) (*sitecontent.UserInfo, error) {
	u := new(sitecontent.UserInfo)
	err := sf(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.Standing,
		&u.IDP,
	)
	return u, err
}

func (s *usersql) Count(ctx scope.C) (int, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return 0, xerr
	}
	defer done()

	const statement = `select count(id) from users`

	f := func(sf litesql.ScanFunc) (int, error) {
		var i int
		err := sf(&i)
		return i, err
	}

	count, err := litesql.QueryRow(ctx, tx, f, statement)
	return count, err
}

func (s *usersql) Create(ctx scope.C, request *users.CreateUserRequest) (users.Identity, error) {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return 0, fmt.Errorf("unable to create user: %w", xerr)
	}
	defer done()

	const statement = `
	insert into users (
		sub, oauths_id, username, email
	) values (
		?, ?, ?, ?
	)`

	auto, ierr := s.ldb.ExecID(ctx, tx, statement,
		request.Subject, request.IDP, request.Username, request.Email,
	)
	if ierr != nil {
		return 0, fmt.Errorf("unable to create user: %w", ierr)
	}

	return users.Identity(auto), tx.Commit()
}

func (s *usersql) Exists(ctx scope.C, idp oauths.IDP, sub oauths.Subject) (users.Identity, bool, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return 0, false, fmt.Errorf("unable to lookup user: %w", xerr)
	}
	defer done()

	const statement = `
	select id
	from users
	where
		oauths_id = ? and
		sub = ?`

	var id users.Identity

	row := s.ldb.QueryRow(ctx, tx, statement, idp, sub)
	if err := row.Scan(&id); err != nil {
		// special case no rows -> no such user (not an error)
		if errors.Is(err, sql.ErrNoRows) {
			return users.Inactive, false, nil
		}
		return users.Inactive, false, fmt.Errorf("unable to scan lookup user: %w", err)
	}

	return id, true, nil
}

func (s *usersql) Info(ctx scope.C, uid users.Identity) (*sitecontent.UserInfo, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to lookup user info: %w", xerr)
	}
	defer done()

	const statement = `
	select
		users.id,
		users.username,
		users.email,
		standings.id,
		oauths.id
	from users
		left join standings on users.standings_id = standings.id
		left join oauths    on users.oauths_id    = oauths.id
	where
		users.id = ?`

	return litesql.QueryRow(ctx, tx, scanUserInfo, statement, uid)
}

func (s *usersql) List(ctx scope.C) ([]*sitecontent.UserInfo, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to list users: %w", xerr)
	}
	defer done()

	const statement = `
	select
		users.id,
		users.username,
		users.email,
		standings.id,
		oauths.id
	from users
		left join standings on users.standings_id = standings.id
		left join oauths    on users.oauths_id    = oauths.id
	where
		oauths.id != ? and
		users.id   > 1
	order by users.id asc`

	return litesql.QueryRows(ctx, tx, scanUserInfo, statement, oauths.Testing)
}

func (s *usersql) SetUsername(ctx scope.C, id users.Identity, chosen string) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("uanble to set username: %w", xerr)
	}
	defer done()

	const statement = `
	update users set
		username = ?
	where
		id = ?`

	if _, uerr := s.ldb.ExecID(ctx, tx, statement, chosen, id); uerr != nil {
		return fmt.Errorf("unable to set username: %w", xerr)
	}

	return tx.Commit()
}

func (s *usersql) ListUsernameChanges(ctx scope.C, id users.Identity) ([]*sitecontent.NameChange, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to list username changes: %w", xerr)
	}
	defer done()

	const statement = `
	select
		usernamechanges.chosen,
		usernamechanges.previous,
		usernamechanges.modification
	from
		usernamechanges
	where
		usernamechanges.users_id = ?
	order by usernamechanges.id desc`

	f := func(sf litesql.ScanFunc) (*sitecontent.NameChange, error) {
		nc := new(sitecontent.NameChange)
		var modification int64
		err := sf(
			&nc.Chosen,
			&nc.Previous,
			&modification,
		)
		nc.Timestamp = time.Unix(modification, 0)
		return nc, err
	}

	return litesql.QueryRows(ctx, tx, f, statement, id)
}

func (s *usersql) Profile(ctx scope.C, uid users.Identity) (*sitecontent.UserProfile, error) {
	tx, done, xerr := s.ldb.StartRead(ctx)
	if xerr != nil {
		return nil, fmt.Errorf("unable to lookup user profile: %w", xerr)
	}
	defer done()

	const statement = `
	select
		users.id,
		users.username,
		users.email,
		standings.id,
		oauths.id,
		userprofiles.company,
		userprofiles.loc,
		userprofiles.link,
		userprofiles.showemail,
		userprofiles.about
	from users
		left join standings    on users.standings_id    = standings.id
		left join oauths       on users.oauths_id       = oauths.id
		left join userprofiles on userprofiles.users_id = users.id
	where
		users.id = ?`

	f := func(sf litesql.ScanFunc) (*sitecontent.UserProfile, error) {
		profile := new(sitecontent.UserProfile)
		err := sf(
			&profile.ID,
			&profile.Username,
			&profile.Email,
			&profile.Standing,
			&profile.IDP,
			&profile.Company,
			&profile.Location,
			&profile.Link,
			&profile.ShowEmail,
			&profile.About,
		)
		return profile, err
	}

	return litesql.QueryRow(ctx, tx, f, statement, uid)
}

func (s *usersql) ProfileSetShowEmail(ctx scope.C, uid users.Identity, show bool) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to edit user profile: %w", xerr)
	}
	defer done()

	const statement = `
	update userprofiles set
		showemail = ?
	where
		users_id = ?`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOneOrZero, statement, show, uid); err != nil {
		return fmt.Errorf("unable to edit user profile showemail: %w", err)
	}

	return tx.Commit()
}

func (s *usersql) ProfileSetLocation(ctx scope.C, uid users.Identity, loc string) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to edit user profile: %w", xerr)
	}
	defer done()

	const statement = `
	update userprofiles set
		loc = ?
	where
		users_id = ?`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOneOrZero, statement, loc, uid); err != nil {
		return fmt.Errorf("unable to edit user profile location: %w", err)
	}

	return tx.Commit()
}

func (s *usersql) ProfileSetLink(ctx scope.C, uid users.Identity, link string) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to set user link: %w", xerr)
	}
	defer done()

	const statement = `
	update userprofiles set
		link = ?
	where
		users_id = ?`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOneOrZero, statement, link, uid); err != nil {
		return fmt.Errorf("unable to edit user profile location: %w", err)
	}

	return tx.Commit()
}

func (s *usersql) ProfileSetAbout(ctx scope.C, uid users.Identity, about string) error {
	tx, done, xerr := s.ldb.StartWrite(ctx)
	if xerr != nil {
		return fmt.Errorf("unable to set user about: %w", xerr)
	}
	defer done()

	const statement = `
	update userprofiles set
		about = ?
	where
		users_id = ?`

	if err := s.ldb.Exec(ctx, tx, litesql.ExpectOneOrZero, statement, about, uid); err != nil {
		return fmt.Errorf("unable to edit user profile about: %w", err)
	}

	return tx.Commit()
}
