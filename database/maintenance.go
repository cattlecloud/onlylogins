package database

import (
	"cattlecloud.net/go/litesql"
	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
)

type Maint interface {
	Optimize(scope.C) error // daily
	Analyze(scope.C) error  // weekly
	Compact(scope.C) error  // weekly
	Vacuum(scope.C) error   // monthly
}

type maintsql struct {
	log *ulog.Log
	ldb *litesql.LiteDB
}

func (s *store) Maint() Maint {
	return &maintsql{
		log: ulog.New("database/maintenance"),
		ldb: s.ldb,
	}
}

func (s *maintsql) Optimize(ctx scope.C) error {
	return s.ldb.Optimize(ctx)
}

func (s *maintsql) Analyze(ctx scope.C) error {
	return s.ldb.Analyze(ctx)
}

func (s *maintsql) Vacuum(ctx scope.C) error {
	return s.ldb.Vacuum(ctx)
}

func (s *maintsql) Compact(_ scope.C) error {
	// TODO see ubs for example on compaction of search tables.
	return nil
}
