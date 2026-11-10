// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package cronjobs

import (
	"testing"

	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/crons"
	"github.com/shoenig/test/must"
)

func TestSyncSubscriptions_Start(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	sync := NewSyncSubscriptions(
		&crons.Config{Expression: "* * * * *"},
		store,
	)

	result := sync.Start()
	must.NotNil(t, result)
	must.Eq(t, crons.ExitSuccess, result.Code)
}
