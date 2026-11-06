package mocks

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"cattlecloud.net/go/webtools/middles/oauth"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

type Session struct {
	SetIdentity users.Identity
	SetActive   bool
}

func (s *Session) Identity() users.Identity {
	return s.SetIdentity
}

func (s *Session) Active() bool {
	return s.SetActive
}

// Sessions is an implementation of middles.Sessions.
//
// This implementation does not expire sessions.
type Sessions struct {
	lock   *sync.Mutex
	active map[users.Identity]*conceal.Text
}

func NewSessions() *Sessions {
	return &Sessions{
		lock:   new(sync.Mutex),
		active: make(map[users.Identity]*conceal.Text),
	}
}

func (s *Sessions) Create(id users.Identity, ttl time.Duration) *http.Cookie {
	factory := &oauth.CookieFactory[users.Identity]{
		Name:  "onlylogins-session",
		Clock: time.Now,
	}
	token := conceal.UUIDv4()
	cookie := factory.Create(id, token, ttl)

	s.lock.Lock()
	s.active[id] = token
	s.lock.Unlock()

	return cookie
}

func (s *Sessions) Match(id users.Identity, token *conceal.Text) error {
	s.lock.Lock()
	t, exists := s.active[id]
	s.lock.Unlock()

	if !exists {
		return errors.New("no such session")
	}

	if !t.Equal(token) {
		return errors.New("session token does not match")
	}

	return nil
}

func (s *Sessions) Delete(token *conceal.Text) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	for k, v := range s.active {
		if token.Equal(v) {
			delete(s.active, k)
			break
		}
	}

	return nil
}
