// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package database

import (
	"fmt"
	"testing"

	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/pkg/sitecontent"
	"github.com/shoenig/test/must"
)

func TestEnforcement_quotas(t *testing.T) {
	t.Parallel()

	t.Run("within limit", func(t *testing.T) {
		db, ctx := openTestDB(t)

		// under the free plan limit of 256
		for i := range 5 {
			insertDocument(t, ctx, db, mocks.Nobody, fmt.Sprintf("doc-%02d", i))
		}

		// creating another document still succeeds
		_, err := db.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User:   mocks.Nobody,
			Folder: "miscellaneous",
			Label:  "another",
			Slug:   "another",
		})
		must.NoError(t, err)
	})

	t.Run("at limit", func(t *testing.T) {
		db, ctx := openTestDB(t)

		// exhaust the free-plan document limit
		for i := range 256 {
			insertDocument(t, ctx, db, mocks.Nobody, fmt.Sprintf("doc-%03d", i))
		}

		// the 257th create must be rejected by the enforcement.quotas check
		_, err := db.Documents().Create(ctx, &sitecontent.CreateDocumentRequest{
			User:   mocks.Nobody,
			Folder: "miscellaneous",
			Label:  "over-the-limit",
			Slug:   "over-the-limit",
		})
		must.ErrorIs(t, err, ErrQuotaExceeded)
	})
}
