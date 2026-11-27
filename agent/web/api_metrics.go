// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package web

import (
	"net/http"
	"time"

	"cattlecloud.net/go/scope"
	"github.com/VictoriaMetrics/metrics"
)

// metricsAPI serves Prometheus metrics.
type metricsAPI struct {
	mset *metrics.Set
}

func (h *metricsAPI) Path() string {
	return "/api/metrics"
}

func (h *metricsAPI) Methods() []string {
	return []string{http.MethodGet}
}

func (h *metricsAPI) Consumer() consumer {
	return robotic
}

func (h *metricsAPI) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.mset.WritePrometheus(w)
	metrics.WriteProcessMetrics(w)
}

// periodicMetrics wires up the metrics which need a periodic source of truth
// (e.g. database counts) as gauges.
func (a *Application) periodicMetrics() {
	a.gaugeCount("ol_users", a.store.Users().Count)
	a.gaugeCount("ol_documents", a.store.Documents().Count)
	a.gaugeCount("ol_folders", a.store.Folders().Count)
	a.gaugeCount("ol_shares", a.store.Shares(a.vault).Count)
}

// gaugeCount registers a gauge named name whose value is obtained by calling
// count against the database. Failures are reported as zero.
func (a *Application) gaugeCount(name string, count func(scope.C) (int, error)) {
	a.metrics.NewGauge(name, func() float64 {
		ctx, cancel := scope.TTL(2 * time.Second)
		defer cancel()

		n, err := count(ctx)
		if err != nil {
			a.log.W.Fmt("metrics: unable to read %s: %v", name, err)
			return 0
		}
		return float64(n)
	})
}
