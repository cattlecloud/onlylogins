// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sitecontent

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestUsage_Percent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		documents int
		limit     int
		expect    int
	}{
		{name: "empty", documents: 0, limit: 256, expect: 0},
		{name: "half", documents: 128, limit: 256, expect: 50},
		{name: "over limit is clamped", documents: 300, limit: 256, expect: 100},
		{name: "no limit", documents: 10, limit: 0, expect: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			usage := &Usage{
				Documents: tc.documents,
				Plan:      Plan{DocumentsLimit: tc.limit},
			}
			must.Eq(t, tc.expect, usage.Percent())
		})
	}
}

func TestUsage_Remaining(t *testing.T) {
	t.Parallel()

	t.Run("remaining", func(t *testing.T) {
		usage := &Usage{
			Documents: 100,
			Plan:      Plan{DocumentsLimit: 256},
		}
		must.Eq(t, 156, usage.Remaining())
	})

	t.Run("over limit", func(t *testing.T) {
		usage := &Usage{
			Documents: 300,
			Plan:      Plan{DocumentsLimit: 256},
		}
		must.Eq(t, 0, usage.Remaining())
	})
}

func TestUsage_Level(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		documents int
		expect    string
	}{
		{name: "normal", documents: 0, expect: "normal"},
		{name: "high", documents: 103, expect: "high"},
		{name: "very high", documents: 192, expect: "veryhigh"},
		{name: "redline", documents: 231, expect: "redline"},
		{name: "overage", documents: 256, expect: "overage"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			usage := &Usage{
				Documents: tc.documents,
				Plan:      Plan{DocumentsLimit: 256},
			}
			must.Eq(t, tc.expect, usage.Level())
		})
	}
}
