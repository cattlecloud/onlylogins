// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"time"

	"cattlecloud.net/go/webtools/middles"
	"cattlecloud.net/onlylogins/pkg/sessions"
	"cattlecloud.net/onlylogins/pkg/sessions/cookies"
	"cattlecloud.net/onlylogins/pkg/users"
)

// slidingSession re-issues the session and encryption cookies with a fresh
// expiry on each authenticated request, implementing sliding (idle-based)
// expiration.
type slidingSession struct {
	next   http.Handler
	slider sessions.Slider
	secure bool
	now    func() time.Time
}

// sliding wraps next with the sliding-session middleware.
func (a *Application) sliding(next http.Handler) *slidingSession {
	return &slidingSession{
		next:   next,
		slider: a.scache,
		secure: a.environment.NonLocal(),
		now:    time.Now,
	}
}

func (s *slidingSession) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// only authenticated requests slide the cookies
	if sess := middles.GetSession[users.Identity](r); sess.Active() {
		// slide the session cookie
		http.SetCookie(w, s.slider.Slide(sess.Identity(), sess.Token(), sessions.IdleTimeout))

		// slide the encryption cookie
		if enc, err := r.Cookie(cookies.Encryption); err == nil {
			enc.Path = "/"
			enc.HttpOnly = true
			enc.Expires = s.now().Add(sessions.IdleTimeout)
			enc.SameSite = http.SameSiteStrictMode
			enc.Secure = s.secure
			http.SetCookie(w, enc)
		}
	}

	// continue to the next handlers
	s.next.ServeHTTP(w, r)
}
