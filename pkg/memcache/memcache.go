package memcache

import (
	"errors"
	"time"

	"cattlecloud.net/go/memc"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/VictoriaMetrics/metrics"
	"github.com/shoenig/go-conceal"
)

// Item wraps something we put in the cache with the insertion timestamp.
type Item[T any] struct {
	Value     T
	Insertion time.Time
}

type Client interface {
	// many more things to store ...

	SessionPut(users.Identity, *conceal.Text, time.Duration)
	SessionGet(*conceal.Text) (*Item[users.Identity], bool)
}

func New(cache *memc.Client, metset *metrics.Set) Client {
	return &client{
		log:   ulog.New("memcache"),
		cache: cache,
		now:   time.Now,

		// setup metrics counters
		missSessions: metset.NewCounter("sb_memc_sessions_miss"),
		hitsSessions: metset.NewCounter("sb_memc_sessions_hits"),
		putsSessions: metset.NewCounter("sb_memc_sessions_puts"),
	}
}

type Config struct {
	IdleConnections int           `toml:"idle-connections"`
	DialTimeout     time.Duration `toml:"dial-timeout"`
	DefaultTTL      time.Duration `toml:"default-ttl"`
	Instances       []string      `toml:"instances"`
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("memcache config must be set")
	}

	if c.IdleConnections < 1 {
		return errors.New("memcache idle-connections must be >= 1")
	}

	if c.DialTimeout < 100*time.Millisecond {
		return errors.New("memcache dial-timeout must be >= 100ms")
	}

	if c.DefaultTTL <= 1*time.Second {
		return errors.New("memcache default-ttl must be >= 1s")
	}

	if len(c.Instances) < 1 {
		return errors.New("memcache instances must specify at least one")
	}

	return nil
}

type client struct {
	log   *ulog.Log
	cache *memc.Client
	now   func() time.Time

	// counters for hit/miss metrics
	missSessions *metrics.Counter
	hitsSessions *metrics.Counter
	putsSessions *metrics.Counter
}

func wrap[T any](c *client, value T) *Item[T] {
	return &Item[T]{
		Value:     value,
		Insertion: c.now(),
	}
}

func (c *client) sessionKey(token *conceal.Text) string {
	key := "onlylogins/sessions/" + token.Unveil()
	return key
}

func (c *client) SessionPut(id users.Identity, token *conceal.Text, ttl time.Duration) {
	c.putsSessions.Inc()

	key := c.sessionKey(token)
	item := wrap(c, id)

	if err := c.cache.Set(key, item, memc.TTL(ttl)); err != nil {
		c.log.W.Fmt("unable to put session: %v", err)
	}
}

func (c *client) SessionGet(token *conceal.Text) (*Item[users.Identity], bool) {
	key := c.sessionKey(token)

	if v, err := c.cache.Get[*Item[users.Identity]](key); err == nil {
		c.hitsSessions.Inc()
		return v, true
	}

	c.missSessions.Inc()
	return nil, false
}
