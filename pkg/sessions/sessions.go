package sessions

import (
	"time"

	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools/middles/oauth"
	"cattlecloud.net/onlylogins/pkg/memcache"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

// YearTTL is a duration representing one full non-leap year.
const YearTTL = 365 * 24 * time.Hour

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
// up to one year, with three hours of grace period.
func (s *Store) remainingTTL(insertion time.Time) time.Duration {
	now := s.now()
	elapsed := now.Sub(insertion)
	remaining := YearTTL - elapsed
	return max(3*time.Hour, remaining)
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
