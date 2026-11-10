// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sitecontent

import (
	"cattlecloud.net/onlylogins/pkg/users"
)

type UserModeration struct {
	Profile *users.Profile
	Plan    Plan
	Usage   Usage
}
