// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

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
}

func newFakeMemcache() *fakeMemcache {
	return &fakeMemcache{
		sessions: make(map[string]*memcache.Item[users.Identity]),
	}
}

func (f *fakeMemcache) SessionPut(id users.Identity, token *conceal.Text, _ time.Duration) {
	f.sessions[token.Unveil()] = &memcache.Item[users.Identity]{
		Value:     id,
		Insertion: time.Now(),
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

func Test_remainingTTL(t *testing.T) {
	t.Parallel()

	s := &Store{
		now: func() time.Time {
			return time.Date(2026, 2, 2, 9, 30, 0, 0, time.UTC)
		},
	}

	t.Run("normal", func(t *testing.T) {
		insertion := time.Date(2026, 2, 2, 9, 29, 0, 0, time.UTC)
		result := s.remainingTTL(insertion)
		must.Eq(t, result, 9*time.Minute) // insert was 1 minute ago
	})

	t.Run("grace", func(t *testing.T) {
		insertion := time.Date(2025, 2, 2, 9, 30, 0, 0, time.UTC)
		result := s.remainingTTL(insertion)
		must.Eq(t, result, 1*time.Minute) // grace period
	})
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("invalidates both caches", func(t *testing.T) {
		fake := newFakeMemcache()
		s := New(fake)

		token := conceal.UUIDv4()
		s.Put(token, users.Identity(7), TTL)

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
		s.Put(token, users.Identity(7), TTL)

		err := s.Delete(token)
		must.Error(t, err)
		must.ErrorContains(t, err, "memcache down")
	})
}
