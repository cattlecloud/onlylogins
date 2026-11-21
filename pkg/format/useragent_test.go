// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package format

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestUserAgent(t *testing.T) {
	t.Parallel()

	t.Run("chrome desktop", func(t *testing.T) {
		ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
		result := UserAgent(ua)
		must.StrContains(t, result, "Chrome")
		must.StrContains(t, result, "macOS")
		must.StrContains(t, result, "desktop")
	})

	t.Run("android phone", func(t *testing.T) {
		ua := "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"
		result := UserAgent(ua)
		must.StrContains(t, result, "Android")
		must.StrContains(t, result, "phone")
	})

	t.Run("bot", func(t *testing.T) {
		result := UserAgent("Googlebot/2.1 (+http://www.google.com/bot.html)")
		must.StrContains(t, result, "bot")
	})

	t.Run("empty", func(t *testing.T) {
		must.Eq(t, "unknown", UserAgent(""))
	})
}
