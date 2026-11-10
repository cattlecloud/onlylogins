// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sitecontent

import (
	"errors"
	"time"

	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

type CreateShareRequest struct {
	User   users.Identity
	Secret *conceal.Text
	TTL    time.Duration
	Now    time.Time
}

func (c *CreateShareRequest) SecondsTTL() int {
	return int(c.TTL.Seconds())
}

func (c *CreateShareRequest) Validate() error {
	switch {
	case c == nil:
		return errors.New("must not be nil")
	case c.User <= 0:
		return errors.New("user must be set")
	case c.Secret.IsZero():
		return errors.New("content must be set")
	case c.Secret.Len() > 8000:
		return errors.New("content is too long (8k)")
	default:
		return nil
	}
}

type Share struct {
	UUID     records.SharesUnique
	User     users.Identity
	Secret   *conceal.Text
	Creation time.Time
	TTL      time.Duration
}

func (s *Share) Expired(now time.Time) bool {
	if s == nil {
		return true
	}
	expiration := s.Creation.Add(s.TTL)
	return now.After(expiration)
}

type ConsumeShareRequest struct {
	UUID records.SharesUnique
	IP   string
}
