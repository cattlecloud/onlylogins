// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package web

import (
	"fmt"
	"net/http"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/sitemaps"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/go/webtools"
	"cattlecloud.net/go/webtools/runtime"
)

type sitemapAPI struct {
	log         *ulog.Log
	environment runtime.Environment
}

func (h *sitemapAPI) Path() string {
	return "/sitemap.xml"
}

func (h *sitemapAPI) Methods() []string {
	return []string{http.MethodGet}
}

func (h *sitemapAPI) Consumer() consumer {
	return robotic
}

func (h *sitemapAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ua := webtools.Origins(r)
	h.log.I.Fmt("serving to %s", ua.From())

	ctx, cancel := scope.WithTTL(r.Context(), timeout)
	defer cancel()

	serve(
		ctx, h.log, w, r, methods{
			http.MethodGet: h.get,
		},
	)
}

func (h *sitemapAPI) get(ctx scope.C, w http.ResponseWriter, _ *http.Request) error {
	// 1) generate fresh sitemap
	site, serr := h.create(ctx)
	if serr != nil {
		return abort(h.log, w, fmt.Errorf("unable to create sitemap.xml: %w", serr))
	}

	// 2) write sitemap to response
	if err := site.Write(w); err != nil {
		return abort(h.log, w, fmt.Errorf("unable to write sitemap.xml into response: %w", err))
	}

	return nil
}

func (h *sitemapAPI) create(ctx scope.C) (sitemaps.Site, error) {
	_ = ctx

	const basics = 5
	today := sitemaps.DayFrom(time.Now())
	origin := sitemaps.CalendarDay{Year: 2026, Month: 11, Day: 13}
	urls := make(sitemaps.Site, 0, basics)

	// home page

	urls = append(urls, &sitemaps.URL{
		Location:  h.url("/"),
		Modified:  today,
		Frequency: sitemaps.Monthly,
		Priority:  sitemaps.Normal,
	})

	// navigation pages

	urls = append(urls, &sitemaps.URL{
		Location:  h.url("/export"),
		Modified:  today,
		Frequency: sitemaps.Monthly,
		Priority:  sitemaps.Normal,
	})

	urls = append(urls, &sitemaps.URL{
		Location:  h.url("/share"),
		Modified:  today,
		Frequency: sitemaps.Monthly,
		Priority:  sitemaps.Normal,
	})

	urls = append(urls, &sitemaps.URL{
		Location:  h.url("/patreon"),
		Modified:  today,
		Frequency: sitemaps.Monthly,
		Priority:  sitemaps.Normal,
	})

	// basic pages

	urls = append(urls, &sitemaps.URL{
		Location:  h.url("/about"),
		Modified:  origin,
		Frequency: sitemaps.Yearly,
		Priority:  sitemaps.Low,
	})

	urls = append(urls, &sitemaps.URL{
		Location:  h.url("/terms"),
		Modified:  origin,
		Frequency: sitemaps.Yearly,
		Priority:  sitemaps.Low,
	})

	urls = append(urls, &sitemaps.URL{
		Location:  h.url("/privacy"),
		Modified:  origin,
		Frequency: sitemaps.Yearly,
		Priority:  sitemaps.Low,
	})

	urls = append(urls, &sitemaps.URL{
		Location:  h.url("/help"),
		Modified:  origin,
		Frequency: sitemaps.Yearly,
		Priority:  sitemaps.Low,
	})

	return urls, nil
}

func (h *sitemapAPI) url(page string) string {
	u := h.environment.Canonical("%s", page)
	return string(u)
}
