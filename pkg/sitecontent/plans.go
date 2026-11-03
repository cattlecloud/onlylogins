// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package sitecontent

import (
	"fmt"

	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/users"
)

type Plan struct {
	ID             records.PlansEnum
	Label          string
	MonthlyPrice   int // cents
	DocumentsLimit int // account wide
}

func (p *Plan) PriceTag() string {
	if p.MonthlyPrice == 0 {
		return "$0 - free!"
	}
	dollars := float64(p.MonthlyPrice) / 100
	s := fmt.Sprintf("$%.2f / month", dollars)
	return s
}

type Usage struct {
	User      users.Identity
	Plan      Plan
	Documents int
}

// Percent returns the percentage of the plan's document limit that has been
// used, clamped to the range 0-100 for display purposes.
func (u *Usage) Percent() int {
	if u == nil || u.Plan.DocumentsLimit <= 0 {
		return 0
	}
	percent := (u.Documents * 100) / u.Plan.DocumentsLimit
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}

// Remaining returns the number of documents remaining on the plan. The result
// is never negative.
func (u *Usage) Remaining() int {
	if u == nil {
		return 0
	}
	remaining := u.Plan.DocumentsLimit - u.Documents
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Level returns a style level for the current utilization of the plan.
func (u *Usage) Level() string {
	percent := u.Percent()
	switch {
	case percent >= 100:
		return "overage"
	case percent >= 90:
		return "redline"
	case percent >= 75:
		return "veryhigh"
	case percent >= 40:
		return "high"
	default:
		return "normal"
	}
}
