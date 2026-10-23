// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package sitecontent

import (
	"errors"
	"regexp"

	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

type Weblink struct {
	ID records.WeblinksAuto
	V  *conceal.Bytes
}

type Username struct {
	ID records.UsernamesAuto
	V  *conceal.Bytes
}

type Password struct {
	ID records.PasswordsAuto
	V  *conceal.Bytes
}

type Keyval struct {
	ID records.KeyvalsAuto
	K  *conceal.Text
	V  *conceal.Bytes
}

type Note struct {
	ID records.NotesAuto
	V  *conceal.Bytes
}

type Document struct {
	ID records.DocumentsAuto
}

type CreateDocumentRequest struct {
	User   users.Identity
	Folder string
	Label  string
}

// docRe restricts document labels to alphanumeric characters so as to be
// URL-safe (but with uppercase enabled)
var docRe = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

func (r *CreateDocumentRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if r.Folder == "" {
		return errors.New("folder must be set")
	}

	if !docRe.MatchString(r.Label) {
		return errors.New("document label must contain alphanumeric characters")
	}

	return nil
}
