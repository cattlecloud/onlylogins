// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package database

import "cattlecloud.net/go/litesql"

func scanCount(scan litesql.ScanFunc) (int, error) {
	var i int
	err := scan(&i)
	return i, err
}
