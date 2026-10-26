package database

import (
	"testing"
	"time"

	"cattlecloud.net/onlylogins/pkg/crons"
	"github.com/shoenig/test/must"
)

func TestCrons_Register(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	err := c.Register(ctx, "test-cron")
	must.NoError(t, err)

	list, lerr := c.List(ctx)
	must.NoError(t, lerr)
	must.Len(t, 1, list)

	must.Eq(t, "test-cron", list[0].Label)
	must.Eq(t, "pending", list[0].State)
}

func TestCrons_RegisterMultiple(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	err := c.Register(ctx, "cron-one")
	must.NoError(t, err)
	err = c.Register(ctx, "cron-two")
	must.NoError(t, err)

	list, lerr := c.List(ctx)
	must.NoError(t, lerr)
	must.Len(t, 2, list)

	must.Eq(t, "cron-one", list[0].Label)
	must.Eq(t, "cron-two", list[1].Label)
}

func TestCrons_ListEmpty(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	list, err := c.List(ctx)
	must.NoError(t, err)
	must.Len(t, 0, list)
}

func TestCrons_Latest(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	err := c.Register(ctx, "my-cron")
	must.NoError(t, err)

	err = c.Start(ctx, "my-cron")
	must.NoError(t, err)

	result, lerr := c.Latest(ctx, "my-cron")
	must.NoError(t, lerr)
	must.Eq(t, "my-cron", result.Label)
	must.Eq(t, "running", result.State)
}

func TestCrons_Get(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	err := c.Register(ctx, "my-cron")
	must.NoError(t, err)

	status, gerr := c.Get(ctx, "my-cron")
	must.NoError(t, gerr)
	must.Eq(t, "my-cron", status.Label)
	must.Eq(t, "pending", status.State)
}

func TestCrons_GetNotFound(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	status, err := c.Get(ctx, "nonexistent")
	must.Error(t, err)
	must.Nil(t, status)
}

func TestCrons_Start(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	err := c.Register(ctx, "my-cron")
	must.NoError(t, err)

	err = c.Start(ctx, "my-cron")
	must.NoError(t, err)

	status, gerr := c.Get(ctx, "my-cron")
	must.NoError(t, gerr)
	must.Eq(t, "my-cron", status.Label)
	must.Eq(t, "running", status.State)
}

func TestCrons_StartNotFound(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	err := c.Start(ctx, "nonexistent")
	must.Error(t, err)
}

func TestCrons_Stop(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	err := c.Register(ctx, "my-cron")
	must.NoError(t, err)

	err = c.Start(ctx, "my-cron")
	must.NoError(t, err)

	metric := &crons.RunMetric{
		Code:  crons.ExitSuccess,
		Start: time.Now().Add(-time.Minute),
		Stop:  time.Now(),
	}
	err = c.Stop(ctx, "my-cron", metric)
	must.NoError(t, err)

	status, gerr := c.Get(ctx, "my-cron")
	must.NoError(t, gerr)
	must.Eq(t, "my-cron", status.Label)
	must.Eq(t, "stopped", status.State)

	list, lerr := c.List(ctx)
	must.NoError(t, lerr)
	must.Len(t, 1, list)
	must.Eq(t, crons.ExitSuccess, list[0].Code)
}

func TestCrons_StopNotFound(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	err := c.Stop(ctx, "nonexistent", &crons.RunMetric{})
	must.Error(t, err)
}

func TestCrons_Reset(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	err := c.Register(ctx, "my-cron")
	must.NoError(t, err)

	err = c.Start(ctx, "my-cron")
	must.NoError(t, err)

	err = c.Reset(ctx, "my-cron")
	must.NoError(t, err)

	status, gerr := c.Get(ctx, "my-cron")
	must.NoError(t, gerr)
	must.Eq(t, "my-cron", status.Label)
	must.Eq(t, "stopped", status.State)
}

func TestCrons_ResetNotFound(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	err := c.Reset(ctx, "nonexistent")
	must.Error(t, err)
}

func TestCrons_Prune(t *testing.T) {
	t.Parallel()

	db, ctx := openTestDB(t)
	c := db.Crons()

	err := c.Register(ctx, "my-cron")
	must.NoError(t, err)

	oldMetric := &crons.RunMetric{
		Code:  crons.ExitSuccess,
		Start: time.Now().Add(-48 * time.Hour),
		Stop:  time.Now().Add(-48 * time.Hour).Add(time.Minute),
	}
	err = c.Stop(ctx, "my-cron", oldMetric)
	must.NoError(t, err)

	err = c.Start(ctx, "my-cron")
	must.NoError(t, err)

	recentMetric := &crons.RunMetric{
		Code:  crons.ExitSuccess,
		Start: time.Now(),
		Stop:  time.Now(),
	}
	err = c.Stop(ctx, "my-cron", recentMetric)
	must.NoError(t, err)

	listBefore, lerr := c.List(ctx)
	must.NoError(t, lerr)
	must.Len(t, 1, listBefore)

	pruned, perr := c.Prune(ctx, 24*time.Hour)
	must.NoError(t, perr)
	must.Eq(t, 2, pruned)

	listAfter, aerr := c.List(ctx)
	must.NoError(t, aerr)
	must.Len(t, 1, listAfter)
	must.Eq(t, crons.ExitSuccess, listAfter[0].Code)
}
