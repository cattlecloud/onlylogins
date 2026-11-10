// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package format

import (
	"time"

	"github.com/dustin/go-humanize"
)

func Ago(t time.Time) string {
	now := time.Now()
	return humanize.RelTime(t, now, "ago", "from now")
}

// DateTime renders the given time as a human-readable, fixed-format timestamp.
//
// Example: "Sunday 02, Nov 2026 13:04 PST".
func DateTime(t time.Time) string {
	return t.Format("Monday 02, Jan 2006 15:04 MST")
}
