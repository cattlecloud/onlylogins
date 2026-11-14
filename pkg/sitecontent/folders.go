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
	ID        records.FoldersAuto
	Label     string
	Slug      string
	Icon      Icon
	Protected bool
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

// folderLabelRe restricts folder names to alphanumeric characters and dashes.
// Unlike document names, folder names may not contain spaces.
var folderLabelRe = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

type CreateFolderRequest struct {
	User  users.Identity
	Label string
	Slug  string
	Icon  Icon
}

func (r *CreateFolderRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !folderLabelRe.MatchString(r.Label) {
		return errors.New("label must contain only letters, numbers, and dashes")
	}

	if r.Icon < 0 || r.Icon > 17 {
		return errors.New("icon must be between 0 and 17")
	}

	return nil
}

type RenameFolderRequest struct {
	User    users.Identity
	Slug    string
	Label   string
	NewSlug string
}

func (r *RenameFolderRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Slug) {
		return errors.New("folder slug must contain only lowercase alphanumeric characters")
	}

	if !folderLabelRe.MatchString(r.Label) {
		return errors.New("label must contain only letters, numbers, and dashes")
	}

	if !ValidateSlug(r.NewSlug) {
		return errors.New("label must contain at least one letter or number")
	}

	return nil
}

type DeleteFolderRequest struct {
	User users.Identity
	Slug string
}

func (r *DeleteFolderRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Slug) {
		return errors.New("folder slug must contain only lowercase alphanumeric characters")
	}

	return nil
}
