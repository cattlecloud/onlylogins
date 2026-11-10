// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sitecontent

import (
	"errors"
	"regexp"

	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/users"
)

// A Folder is the categorization unit for each user to store
// bits of data into.
type Folder struct {
	ID    records.FoldersAuto
	Label string
	Icon  Icon
}

// Icon is an enum associated with an SVG image.
type Icon int

var FolderIcons = map[Icon]string{
	0:  "attention.svg",
	1:  "heart.svg",
	2:  "house.svg",
	3:  "save.svg",
	4:  "search.svg",
	5:  "share.svg",
	6:  "square.svg",
	7:  "stack.svg",
	8:  "stock.svg",
	9:  "tag.svg",
	10: "thumbup.svg",
	11: "tower.svg",
	12: "branch.svg",
	13: "clock.svg",
	14: "cplus.svg",
	15: "cup.svg",
	16: "ghost.svg",
	17: "mail.svg",
}

// labelRe restricts folder and document label names to alphanumeric characters and dashes.
// this way they can be used in url paths without encoding
var labelRe = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

type CreateFolderRequest struct {
	User  users.Identity
	Label string
	Icon  Icon
}

func (r *CreateFolderRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !labelRe.MatchString(r.Label) {
		return errors.New("label must contain only alphanumeric characters")
	}

	if r.Icon < 0 || r.Icon > 17 {
		return errors.New("icon must be between 0 and 17")
	}

	return nil
}

type DeleteFolderRequest struct {
	User  users.Identity
	Label string
}

func (r *DeleteFolderRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !labelRe.MatchString(r.Label) {
		return errors.New("folder label must contain only alphanumeric characters")
	}

	return nil
}
