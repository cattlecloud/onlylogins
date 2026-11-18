// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sessions

import (
	"fmt"
	"net/http"
	"time"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/middles/oauth"
	"cattlecloud.net/onlylogins/pkg/memcache"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

const (
	// IdleTimeout is how long a session may remain inactive before it expires.
	//
	// As a password manager we force users to re-login after a short period of
	// inactivity so that someone leaving their computer unattended is less of a
	// security risk.
	IdleTimeout = 10 * time.Minute

	// MaxLifetime is the absolute upper bound on a session's lifetime. No matter
	// how active a user is, they must re-login after this much time.
	MaxLifetime = 12 * time.Hour

	// TouchThreshold controls how often the last-activity timestamp is refreshed
	// in memcache. Rather than writing on every request we only rewrite once the
	// threshold has elapsed. A smaller threshold approaches an exact idle window
	// at the cost of more memcache writes; with this value the effective idle
	// timeout falls between IdleTimeout-TouchThreshold and IdleTimeout.
	TouchThreshold = IdleTimeout / 2
)

// A Slider is used to re-issue a session cookie with fresh expiration.
type Slider interface {
	Slide(users.Identity, *conceal.Text, time.Duration) *http.Cookie
}

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

// Get returns the identity associated with the given token, enforcing sliding
// (idle-based) expiration. The memcached entry is authoritative: its insertion
// timestamp records the last activity, and is refreshed (throttled) as the
// session continues to be used. Its creation timestamp enforces the absolute
// lifetime cap.
func (s *Store) Get(token *conceal.Text) (users.Identity, bool) {
	key := token.Unveil()

	// fast path: in-process cache. Entries only live until the next touch is due
	// and never past the hard cap, so a hit here always means the session is
	// still valid.
	if id, exists := s.vc.Get(key); exists {
		return id, true
	}

	// slow path: authoritative memcache
	item, exists := s.mc.SessionGet(token)
	if !exists {
		return 0, false
	}

	now := s.now()
	age := now.Sub(item.Created)
	idle := now.Sub(item.Insertion)

	// hard idle expiration or absolute lifetime cap
	if idle > IdleTimeout || age > MaxLifetime {
		_ = s.mc.SessionDelete(token)
		_ = s.vc.Delete(key)
		return 0, false
	}

	if idle >= TouchThreshold {
		// refresh the last-activity timestamp (throttled), preserving the hard
		// cap anchor, and reset the memcache lifetime to the remaining cap
		s.mc.SessionTouch(item, token, MaxLifetime-age)
		idle = 0 // treat as freshly touched for the volatile TTL below
	}

	// mirror locally until the next touch is due, but never past the hard cap
	volatileTTL := TouchThreshold - idle
	if remaining := MaxLifetime - age; remaining < volatileTTL {
		volatileTTL = remaining
	}
	s.vc.Put(key, item.Value, volatileTTL)

	return item.Value, true
}

func (s *Store) Put(token *conceal.Text, id users.Identity, _ time.Duration) {
	// update the external process cache (authoritative) with the absolute cap
	s.mc.SessionPut(id, token, MaxLifetime)

	// update the internal process cache until the next touch is due
	key := token.Unveil()
	s.vc.Put(key, id, TouchThreshold)
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

// Cache combines the session cache with the cookie factory so callers can both
// validate sessions (Create, Match, Delete) and slide their cookies with a
// fresh expiry without rotating the underlying token.
type Cache struct {
	*oauth.Sessions[users.Identity]
}

func NewCache(factory *oauth.CookieFactory[users.Identity], store *Store) *Cache {
	return &Cache{
		Sessions: oauth.NewSessions(factory, store),
	}
}

// Slide re-issues the session cookie for the given identity and token with a
// fresh expiry, reusing the existing token.
func (c *Cache) Slide(id users.Identity, token *conceal.Text, ttl time.Duration) *http.Cookie {
	return c.CookieFactory.Create(id, token, ttl)
}
