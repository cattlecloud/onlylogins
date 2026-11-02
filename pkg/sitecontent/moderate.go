// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package sitecontent

import (
	"cattlecloud.net/onlylogins/pkg/users"
)

type UserModeration struct {
	Profile *users.Profile
	Plan    Plan
	Usage   Usage
}
