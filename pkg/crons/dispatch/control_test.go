// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package dispatch

import (
	"sync"
	"testing"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/crons"
	"github.com/hashicorp/cronexpr"
	"github.com/shoenig/test/must"
	"github.com/shoenig/test/wait"
)

type mockCron struct {
	wg *sync.WaitGroup
}

func (mc *mockCron) Name() string {
	return "mock-cron"
}

func (mc *mockCron) Schedule() *cronexpr.Expression {
	return cronexpr.MustParse("0 */2 * * *")
}

func (mc *mockCron) Start() *crons.RunMetric {
	start := time.Now()
	time.Sleep(123 * time.Millisecond)
	stop := time.Now()
	mc.wg.Done()
	return &crons.RunMetric{
		Code:  0,
		Start: start,
		Stop:  stop,
	}
}

func TestControl_registration(t *testing.T) {
	t.Parallel()

	ctx, cancel := scope.WithTTL(t.Context(), timeout)
	defer cancel()

	db := mockdb.Open(t)
	store := db.Crons()

	c := New(ctx, store)
	must.NotNil(t, c)

	// prepare our waitgroup
	wg := new(sync.WaitGroup)
	wg.Add(1)

	// register our mock cron
	mc := &mockCron{wg: wg}
	must.NoError(t, c.Register(mc))

	// wait for mock cron to complete
	wg.Wait()

	// ensure cron becomes stopped in database
	must.Wait(t, wait.InitialSuccess(
		wait.Timeout(3*time.Second),
		wait.TestFunc(func() (bool, error) {
			status, err := store.Get(ctx, mc.Name())
			return status.State == crons.StateStopped, err
		}),
	))
}
