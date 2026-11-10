// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"cattlecloud.net/go/ulog"
	"github.com/shoenig/test/must"
)

func Test_abort(t *testing.T) {
	t.Parallel()

	log := ulog.New("test")
	w := httptest.NewRecorder()
	err := errors.New("oops")

	result := abort(log, w, err)

	must.ErrorIs(t, result, ErrPageProblem)
	must.Eq(t, http.StatusInternalServerError, w.Code)
	must.Eq(t, "text/plain; charset=utf8", w.Header().Get("Content-Type"))
	must.Eq(t, "unable to process request", w.Body.String())
}

func Test_deny(t *testing.T) {
	t.Parallel()

	log := ulog.New("test")
	w := httptest.NewRecorder()
	err := errors.New("oops")

	result := deny(log, w, err)

	must.NoError(t, result)
	must.Eq(t, http.StatusForbidden, w.Code)
	must.Eq(t, "text/plain; charset=utf8", w.Header().Get("Content-Type"))
	must.Eq(t, "potentially malicious request denied", w.Body.String())
}

func Test_lost(t *testing.T) {
	t.Parallel()

	log := ulog.New("test")
	w := httptest.NewRecorder()
	r, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/nothing", nil)
	must.NoError(t, rerr)

	err := lost(log, w, r)

	must.NoError(t, err)
	must.Eq(t, http.StatusNotFound, w.Code)
	must.Eq(t, "text/plain; charset=utf8", w.Header().Get("Content-Type"))
	must.Eq(t, "page not found", w.Body.String())
}
