// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package sitecontent

import (
	"errors"
	"regexp"
	"strings"

	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

var ErrNoUnlock = "(decryption-failure)"

func unlock(b *conceal.Bytes, v encrypt.Vault) string {
	s, err := v.Decrypt(b)
	if err != nil {
		return ErrNoUnlock
	}
	return s.Unveil()
}

type Weblink struct {
	ID records.WeblinksAuto
	V  *conceal.Bytes
}

func (w *Weblink) Unlock(v encrypt.Vault) string {
	return unlock(w.V, v)
}

type Username struct {
	ID records.UsernamesAuto
	V  *conceal.Bytes
}

func (u *Username) Unlock(v encrypt.Vault) string {
	return unlock(u.V, v)
}

type Password struct {
	ID records.PasswordsAuto
	V  *conceal.Bytes
}

func (p *Password) Unlock(v encrypt.Vault) string {
	return unlock(p.V, v)
}

type Keyval struct {
	ID records.KeyvalsAuto
	K  *conceal.Text
	V  *conceal.Bytes
}

func (k *Keyval) Unlock(v encrypt.Vault) string {
	return unlock(k.V, v)
}

type Note struct {
	ID records.NotesAuto
	V  *conceal.Bytes
}

func (n *Note) Unlock(v encrypt.Vault) string {
	return unlock(n.V, v)
}

type DocumentListing struct {
	ID    records.DocumentsAuto
	Label string
}

type Document struct {
	ID        records.DocumentsAuto
	Label     string
	Folder    string
	Weblinks  []*Weblink
	Usernames []*Username
	Passwords []*Password
	Keyvals   []*Keyval
	Notes     []*Note
}

type CreateDocumentRequest struct {
	User   users.Identity
	Folder string
	Label  string
}

type ListDocumentsRequest struct {
	User   users.Identity
	Folder string
}

type GetDocumentRequest struct {
	User     users.Identity
	Folder   string
	Document string
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

func (r *GetDocumentRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if r.Folder == "" {
		return errors.New("folder must be set")
	}

	if r.Document == "" {
		return errors.New("document must be set")
	}

	return nil
}

type CreateWeblinkRequest struct {
	User      users.Identity
	Folder    string
	Document  string
	URL       string
	Encrypted *conceal.Bytes
}

func (r *CreateWeblinkRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if r.Folder == "" {
		return errors.New("folder must be set")
	}

	if r.Document == "" {
		return errors.New("document must be set")
	}

	r.URL = strings.TrimSpace(r.URL)
	if r.URL == "" {
		return errors.New("url must be set")
	}

	if !strings.HasPrefix(r.URL, "http://") && !strings.HasPrefix(r.URL, "https://") {
		r.URL = "https://" + r.URL
	}

	return nil
}

type UpdateWeblinkRequest struct {
	ID        records.WeblinksAuto
	User      users.Identity
	Folder    string
	Document  string
	URL       string
	Encrypted *conceal.Bytes
}

func (r *UpdateWeblinkRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("weblink must be set")
	}

	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if r.Folder == "" {
		return errors.New("folder must be set")
	}

	if r.Document == "" {
		return errors.New("document must be set")
	}

	r.URL = strings.TrimSpace(r.URL)
	if r.URL == "" {
		return errors.New("url must be set")
	}

	if !strings.HasPrefix(r.URL, "http://") && !strings.HasPrefix(r.URL, "https://") {
		r.URL = "https://" + r.URL
	}

	return nil
}

type DeleteWeblinkRequest struct {
	ID       records.WeblinksAuto
	User     users.Identity
	Folder   string
	Document string
}

func (r *DeleteWeblinkRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("weblink must be set")
	}

	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if r.Folder == "" {
		return errors.New("folder must be set")
	}

	if r.Document == "" {
		return errors.New("document must be set")
	}

	return nil
}
