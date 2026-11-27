// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/VictoriaMetrics/metrics"
	"github.com/shoenig/test/must"
)

func TestMetricsAPI(t *testing.T) {
	t.Parallel()

	h := &metricsAPI{
		mset: metrics.NewSet(),
	}

	request, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/metrics", nil)
	must.NoError(t, rerr)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, request)

	must.Eq(t, 200, w.Code)
	must.StrContains(t, w.Body.String(), "go_gc_cpu_seconds_total")
}
