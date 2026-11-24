// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package users

import (
	"testing"

	"cattlecloud.net/go/scope"
	"github.com/hashicorp/go-set/v3"
	"github.com/shoenig/test/must"
)

type testStore struct {
	existing set.Collection[Identity]
	add      set.Collection[Identity]
	remove   set.Collection[Identity]
}

func (s *testStore) Sync(_ scope.C, add, remove set.Collection[Identity]) error {
	s.add = add
	s.remove = remove
	return nil
}

func (s *testStore) ListStanding(_ scope.C, _ Standing) (set.Collection[Identity], error) {
	if s.existing == nil {
		return set.New[Identity](0), nil
	}
	return s.existing, nil
}

func TestCouncil_IsAdmin(t *testing.T) {
	t.Parallel()

	c := NewCouncil([]Identity{2, 98}, &testStore{})

	must.True(t, c.IsAdmin(2))
	must.True(t, c.IsAdmin(98))
	must.False(t, c.IsAdmin(1))
	must.False(t, c.IsAdmin(101))
}

func TestCouncil_Sync(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	store := &testStore{existing: set.From([]Identity{2, 3})}
	c := NewCouncil([]Identity{2, 98}, store)

	c.Sync(ctx)

	// config admin 98 is missing from the db, so it must be added
	must.Eq(t, 1, store.add.Size())
	must.True(t, store.add.Contains(98))

	// db admin 3 is missing from the config, so it must be removed
	must.Eq(t, 1, store.remove.Size())
	must.True(t, store.remove.Contains(3))
}
