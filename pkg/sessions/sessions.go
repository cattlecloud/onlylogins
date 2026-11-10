// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sessions

import (
	"fmt"
	"time"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/middles/oauth"
	"cattlecloud.net/onlylogins/pkg/memcache"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

// TTL is the hard-coded lifespan of sessions.
//
// As a password manager we force users to re-login after a short period of time
// so that someone leaving their computer unattended is less of a security risk.
const TTL = 10 * time.Minute

// Store is an implementation of oauth.Cache that makes use of both the basic
// in-memory oauth.Volatile cache as well as a fallback memcached client for
// out of process lookups.
type Store struct {
	log *ulog.Log
	now func() time.Time
	vc  *oauth.VolatileCache[users.Identity] // internal-process memory
	mc  memcache.Client                      // external-process memory
}

func New(mc memcache.Client) *Store {
	return &Store{
		log: ulog.New("sessions"),
		now: time.Now,
		mc:  mc,
		vc:  oauth.NewVolatileCache[users.Identity](8),
	}
}

// remainingTTL computes the amount of time the item should remain in cache
// given its original insertion time and the fact that we store sessions for
// up to one year, with one minute of grace period.
func (s *Store) remainingTTL(insertion time.Time) time.Duration {
	now := s.now()
	elapsed := now.Sub(insertion)
	remaining := TTL - elapsed
	return max(1*time.Minute, remaining)
}

func (s *Store) Get(token *conceal.Text) (users.Identity, bool) {
	// check the fast key where the in-process cache knows the identity
	key := token.Unveil()
	if id, exists := s.vc.Get(key); exists {
		return id, true
	}

	// now check the memcache (going through our shim with metrics)
	if item, exists := s.mc.SessionGet(token); exists {
		// we found the session in memcache; restore into vc
		remaining := s.remainingTTL(item.Insertion)
		s.vc.Put(key, item.Value, remaining)
		return item.Value, true
	}

	// the item was in neither cache
	return 0, false
}

func (s *Store) Put(token *conceal.Text, id users.Identity, ttl time.Duration) {
	// update the external process cache
	s.mc.SessionPut(id, token, ttl)

	// update the internal process cache
	key := token.Unveil()
	s.vc.Put(key, id, ttl)
}

// Delete terminates the session associated with the given token.
//
// Memcache is the authoritative store and is deleted first; the local volatile
// cache is always cleared, even if the memcache deletion fails.
func (s *Store) Delete(token *conceal.Text) error {
	merr := s.mc.SessionDelete(token)

	// the volatile cache delete never fails, but must still be cleared
	_ = s.vc.Delete(token.Unveil())

	if merr != nil {
		return fmt.Errorf("unable to delete session from memcache: %w", merr)
	}

	return nil
}
