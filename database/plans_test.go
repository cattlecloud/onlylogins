// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package database

import (
	"testing"

	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/hack/mocks"
	"github.com/shoenig/test/must"
)

func TestPlans_Get(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	p := db.Plans()

	t.Run("free", func(t *testing.T) {
		plan, err := p.Get(ctx, records.PlansEnum(1))
		must.NoError(t, err)
		must.NotNil(t, plan)
		must.Eq(t, records.PlansEnum(1), plan.ID)
		must.Eq(t, "free", plan.Label)
		must.Eq(t, 0, plan.MonthlyPrice)
		must.Eq(t, 200, plan.DocumentsLimit)
	})

	t.Run("patreon", func(t *testing.T) {
		plan, err := p.Get(ctx, records.PlansEnum(2))
		must.NoError(t, err)
		must.NotNil(t, plan)
		must.Eq(t, records.PlansEnum(2), plan.ID)
		must.Eq(t, "patreon", plan.Label)
		must.Eq(t, 199, plan.MonthlyPrice)
		must.Eq(t, 999, plan.DocumentsLimit)
	})

	t.Run("not found", func(t *testing.T) {
		plan, err := p.Get(ctx, records.PlansEnum(999))
		must.Error(t, err)
		must.Nil(t, plan)
	})
}

func TestPlans_List(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	p := db.Plans()

	list, err := p.List(ctx)
	must.NoError(t, err)
	must.Len(t, 2, list)

	// ordered ascending by id: free (1), then patreon (2)
	must.Eq(t, records.PlansEnum(1), list[0].ID)
	must.Eq(t, "free", list[0].Label)
	must.Eq(t, 0, list[0].MonthlyPrice)
	must.Eq(t, 200, list[0].DocumentsLimit)

	must.Eq(t, records.PlansEnum(2), list[1].ID)
	must.Eq(t, "patreon", list[1].Label)
	must.Eq(t, 199, list[1].MonthlyPrice)
	must.Eq(t, 999, list[1].DocumentsLimit)
}

func TestPlans_Usage(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	p := db.Plans()

	t.Run("nobody empty", func(t *testing.T) {
		usage, err := p.Usage(ctx, mocks.Nobody)
		must.NoError(t, err)
		must.NotNil(t, usage)
		must.Eq(t, mocks.Nobody, usage.User)
		must.Eq(t, records.PlansEnum(1), usage.Plan.ID)
		must.Eq(t, "free", usage.Plan.Label)
		must.Eq(t, 0, usage.Plan.MonthlyPrice)
		must.Eq(t, 200, usage.Plan.DocumentsLimit)
		must.Zero(t, usage.Documents)
	})

	t.Run("nobody with documents", func(t *testing.T) {
		insertDocument(t, ctx, db, mocks.Nobody, "github")
		insertDocument(t, ctx, db, mocks.Nobody, "gitlab")

		usage, err := p.Usage(ctx, mocks.Nobody)
		must.NoError(t, err)
		must.NotNil(t, usage)
		must.Eq(t, mocks.Nobody, usage.User)
		must.Eq(t, 2, usage.Documents)
	})

	t.Run("other user", func(t *testing.T) {
		alice := insertUser(t, ctx, db, "alice")

		usage, err := p.Usage(ctx, alice)
		must.NoError(t, err)
		must.NotNil(t, usage)
		must.Eq(t, alice, usage.User)
		must.Eq(t, records.PlansEnum(1), usage.Plan.ID)
		must.Eq(t, "free", usage.Plan.Label)
		must.Zero(t, usage.Documents)
	})
}
