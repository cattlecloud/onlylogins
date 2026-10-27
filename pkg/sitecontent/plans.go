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
