package database

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestMaint_Optimize(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	m := db.Maint()

	err := m.Optimize(ctx)
	must.NoError(t, err)
}

func TestMaint_Analyze(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	m := db.Maint()

	err := m.Analyze(ctx)
	must.NoError(t, err)
}

func TestMaint_Vacuum(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	m := db.Maint()

	err := m.Vacuum(ctx)
	must.NoError(t, err)
}

func TestMaint_Compact(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	m := db.Maint()

	err := m.Compact(ctx)
	must.NoError(t, err)
}
