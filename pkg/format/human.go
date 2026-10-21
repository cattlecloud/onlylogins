// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package format

import (
	"time"

	"github.com/dustin/go-humanize"
)

func Ago(t time.Time) string {
	now := time.Now()
	return humanize.RelTime(t, now, "ago", "from now")
}
