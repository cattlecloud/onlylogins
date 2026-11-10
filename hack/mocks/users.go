// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package mocks

import (
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

const Nobody = users.Identity(1)

var Pass = conceal.New("abcdefghijklmnopqrstuvwxyz")
