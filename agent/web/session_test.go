// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cattlecloud.net/onlylogins/hack/mocks"
	"cattlecloud.net/onlylogins/pkg/sessions"
	"cattlecloud.net/onlylogins/pkg/sessions/cookies"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
	"github.com/shoenig/test/must"
)

// mockSlider is a test double for the Slider capability.
type mockSlider struct {
	slideID    users.Identity
	slideToken *conceal.Text
	slideTTL   time.Duration
}

func (f *mockSlider) Slide(id users.Identity, token *conceal.Text, ttl time.Duration) *http.Cookie {
	f.slideID = id
	f.slideToken = token
	f.slideTTL = ttl
	return &http.Cookie{Name: cookies.Session, Value: "slid", Expires: time.Unix(1, 0)}
}

func getCookie(set []*http.Cookie, name string) *http.Cookie {
	for _, c := range set {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestSlidingSession_ServeHTTP_active(t *testing.T) {
	t.Parallel()

	scache := mocks.NewSessions()
	sessionCookie := scache.Create(mocks.Nobody, 1*time.Minute)

	now := time.Date(2026, 2, 2, 10, 0, 0, 0, time.UTC)

	slider := &mockSlider{}
	slide := &slidingSession{
		next:   http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
		slider: slider,
		secure: false,
		now:    func() time.Time { return now },
	}

	ss := session(slide, scache)

	req := requestGet(t, "/", sessionCookie)
	req.AddCookie(&http.Cookie{Name: cookies.Encryption, Value: "secret"})

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, req)

	must.Eq(t, http.StatusOK, w.Code)

	// the slider must have been invoked with the active identity and token
	must.Eq(t, mocks.Nobody, slider.slideID)
	must.NotNil(t, slider.slideToken)
	must.Eq(t, sessions.IdleTimeout, slider.slideTTL)

	// both cookies must be re-issued
	set := w.Result().Cookies()

	sessionSet := getCookie(set, cookies.Session)
	must.NotNil(t, sessionSet)
	must.Eq(t, "slid", sessionSet.Value)

	encryptionSet := getCookie(set, cookies.Encryption)
	must.NotNil(t, encryptionSet)
	must.Eq(t, "secret", encryptionSet.Value)
	must.Eq(t, now.Add(sessions.IdleTimeout), encryptionSet.Expires)
	must.True(t, encryptionSet.HttpOnly)
	must.Eq(t, http.SameSiteStrictMode, encryptionSet.SameSite)
	must.False(t, encryptionSet.Secure)
}

func TestSlidingSession_ServeHTTP_inactive(t *testing.T) {
	t.Parallel()

	scache := mocks.NewSessions()

	slider := &mockSlider{}
	slide := &slidingSession{
		next:   http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
		slider: slider,
		secure: false,
		now:    time.Now,
	}

	ss := session(slide, scache)

	req := requestGet(t, "/", nil)

	w := httptest.NewRecorder()
	ss.ServeHTTP(w, req)

	must.Eq(t, http.StatusOK, w.Code)

	// no cookies must be re-issued and the slider must not be invoked
	must.Eq(t, 0, len(w.Result().Cookies()))
	must.Eq(t, users.Identity(0), slider.slideID)
	must.Nil(t, slider.slideToken)
}
