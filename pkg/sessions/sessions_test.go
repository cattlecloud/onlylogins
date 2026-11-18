// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sessions

import (
	"errors"
	"testing"
	"time"

	"cattlecloud.net/onlylogins/pkg/memcache"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

type fakeMemcache struct {
	sessions map[string]*memcache.Item[users.Identity]
	err      error
	now      func() time.Time
}

func newFakeMemcache() *fakeMemcache {
	return &fakeMemcache{
		sessions: make(map[string]*memcache.Item[users.Identity]),
		now:      time.Now,
	}
}

func (f *fakeMemcache) SessionPut(id users.Identity, token *conceal.Text, _ time.Duration) {
	now := f.now()
	f.sessions[token.Unveil()] = &memcache.Item[users.Identity]{
		Value:     id,
		Created:   now,
		Insertion: now,
	}
}

func (f *fakeMemcache) SessionTouch(item *memcache.Item[users.Identity], token *conceal.Text, _ time.Duration) {
	f.sessions[token.Unveil()] = &memcache.Item[users.Identity]{
		Value:     item.Value,
		Created:   item.Created,
		Insertion: f.now(),
	}
}

func (f *fakeMemcache) SessionGet(token *conceal.Text) (*memcache.Item[users.Identity], bool) {
	item, exists := f.sessions[token.Unveil()]
	return item, exists
}

func (f *fakeMemcache) SessionDelete(token *conceal.Text) error {
	if f.err != nil {
		return f.err
	}
	delete(f.sessions, token.Unveil())
	return nil
}

// newTestStore builds a Store whose clock is driven by the given mutable now
// pointer, and whose memcache backend shares the same clock. The volatile cache
// is left empty so Get always exercises the authoritative memcache path.
func newTestStore(t *testing.T, fake *fakeMemcache, now *time.Time) *Store {
	t.Helper()
	s := New(fake)
	s.now = func() time.Time { return *now }
	fake.now = func() time.Time { return *now }
	return s
}

func TestStore_Get_idle_expiration(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 2, 2, 9, 0, 0, 0, time.UTC)
	now := start

	fake := newFakeMemcache()
	s := newTestStore(t, fake, &now)

	token := conceal.UUIDv4()
	fake.sessions[token.Unveil()] = &memcache.Item[users.Identity]{
		Value:     users.Identity(7),
		Created:   start,
		Insertion: start,
	}

	// advance past the idle window
	now = start.Add(IdleTimeout + time.Minute)

	id, exists := s.Get(token)
	must.False(t, exists)
	must.Eq(t, users.Identity(0), id)

	// the expired session must be purged from memcache
	_, exists = fake.SessionGet(token)
	must.False(t, exists)
}

func TestStore_Get_touch_refreshes_last_activity(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 2, 2, 9, 0, 0, 0, time.UTC)
	now := start

	fake := newFakeMemcache()
	s := newTestStore(t, fake, &now)

	token := conceal.UUIDv4()
	fake.sessions[token.Unveil()] = &memcache.Item[users.Identity]{
		Value:     users.Identity(7),
		Created:   start,
		Insertion: start,
	}

	// advance beyond the touch threshold but still within the idle window
	now = start.Add(TouchThreshold + time.Minute)

	id, exists := s.Get(token)
	must.True(t, exists)
	must.Eq(t, users.Identity(7), id)

	// the last-activity timestamp must have been refreshed
	item, exists := fake.SessionGet(token)
	must.True(t, exists)
	must.True(t, item.Insertion.Equal(now))
}

func TestStore_Get_within_threshold_does_not_touch(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 2, 2, 9, 0, 0, 0, time.UTC)
	now := start

	fake := newFakeMemcache()
	s := newTestStore(t, fake, &now)

	token := conceal.UUIDv4()
	fake.sessions[token.Unveil()] = &memcache.Item[users.Identity]{
		Value:     users.Identity(7),
		Created:   start,
		Insertion: start,
	}

	// advance but stay within the touch threshold
	now = start.Add(TouchThreshold - time.Minute)

	id, exists := s.Get(token)
	must.True(t, exists)
	must.Eq(t, users.Identity(7), id)

	// the last-activity timestamp must be unchanged
	item, exists := fake.SessionGet(token)
	must.True(t, exists)
	must.True(t, item.Insertion.Equal(start))
}

func TestStore_Get_hard_cap_expiration(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 2, 2, 9, 0, 0, 0, time.UTC)
	now := start.Add(MaxLifetime + time.Minute)

	fake := newFakeMemcache()
	s := newTestStore(t, fake, &now)

	token := conceal.UUIDv4()
	fake.sessions[token.Unveil()] = &memcache.Item[users.Identity]{
		Value:     users.Identity(7),
		Created:   start,
		Insertion: now.Add(-5 * time.Minute), // recent activity, so not idle-expired
	}

	id, exists := s.Get(token)
	must.False(t, exists)
	must.Eq(t, users.Identity(0), id)

	// the expired session must be purged from memcache
	_, exists = fake.SessionGet(token)
	must.False(t, exists)
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("invalidates both caches", func(t *testing.T) {
		fake := newFakeMemcache()
		s := New(fake)

		token := conceal.UUIDv4()
		s.Put(token, users.Identity(7), IdleTimeout)

		err := s.Delete(token)
		must.NoError(t, err)

		// session must not be found, including via memcache fallback
		_, exists := s.Get(token)
		must.False(t, exists)

		// memcache must not retain the session either
		_, exists = fake.SessionGet(token)
		must.False(t, exists)
	})

	t.Run("memcache failure is returned", func(t *testing.T) {
		fake := newFakeMemcache()
		fake.err = errors.New("memcache down")
		s := New(fake)

		token := conceal.UUIDv4()
		s.Put(token, users.Identity(7), IdleTimeout)

		err := s.Delete(token)
		must.Error(t, err)
		must.ErrorContains(t, err, "memcache down")
	})
}
