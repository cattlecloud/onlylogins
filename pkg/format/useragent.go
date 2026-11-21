// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package format

import (
	"strings"

	"github.com/mileusna/useragent"
)

// UserAgent parses a raw user agent string into a compact, human-readable
// description of the client, including the browser, operating system, device,
// and device class (desktop, phone, tablet, or bot).
func UserAgent(s string) string {
	if s == "" {
		return "unknown"
	}

	ua := useragent.Parse(s)

	if ua.Bot {
		if ua.Name != "" {
			return ua.Name + " (bot)"
		}
		return "bot"
	}

	parts := make([]string, 0, 4)

	name := ua.Name
	if name == "" {
		name = "unknown"
	}
	if ua.Version != "" {
		name += " " + ua.Version
	}
	parts = append(parts, name)

	if ua.OS != "" {
		os := ua.OS
		if ua.OSVersion != "" {
			os += " " + ua.OSVersion
		}
		parts = append(parts, "on "+os)
	}

	if ua.Device != "" {
		parts = append(parts, ua.Device)
	}

	switch {
	case ua.Tablet:
		parts = append(parts, "(tablet)")
	case ua.Mobile:
		parts = append(parts, "(phone)")
	case ua.Desktop:
		parts = append(parts, "(desktop)")
	}

	return strings.Join(parts, " ")
}
