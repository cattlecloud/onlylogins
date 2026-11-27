// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sitecontent

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"cattlecloud.net/onlylogins/database/records"
	"cattlecloud.net/onlylogins/pkg/encrypt"
	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/go-conceal"
)

var (
	ErrNoUnlock = "(decryption-failure)"
)

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
	ID           records.DocumentsAuto
	Label        string
	Slug         string
	HasWeblinks  bool
	HasPasswords bool
}

type Document struct {
	ID           records.DocumentsAuto
	Label        string
	Slug         string
	Folder       string
	FolderSlug   string
	Modification time.Time
	Weblinks     []*Weblink
	Usernames    []*Username
	Passwords    []*Password
	Keyvals      []*Keyval
	Note         *Note
}

type CreateDocumentRequest struct {
	User   users.Identity
	Folder string
	Label  string
	Slug   string
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

func (r *CreateDocumentRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateName(r.Label) {
		return errors.New("document must contain at least one letter or number")
	}

	return nil
}

func (r *ListDocumentsRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	return nil
}

func (r *GetDocumentRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
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

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
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

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
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

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	return nil
}

type GetWeblinkRequest struct {
	ID       records.WeblinksAuto
	User     users.Identity
	Folder   string
	Document string
}

func (r *GetWeblinkRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("weblink must be set")
	}
	if r.User <= 0 {
		return errors.New("user must be set")
	}
	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}
	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}
	return nil
}

type ListWeblinksRequest struct {
	User     users.Identity
	Folder   string
	Document string
}

func (r *ListWeblinksRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}
	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}
	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}
	return nil
}

type ListPasswordsRequest struct {
	User     users.Identity
	Folder   string
	Document string
}

func (r *ListPasswordsRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}
	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}
	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}
	return nil
}

// maxUsernameLength bounds how long a username value may be
const maxUsernameLength = 256

// maxPasswordLength bounds how long a password value may be
const maxPasswordLength = 256

type CreateUsernameRequest struct {
	User      users.Identity
	Folder    string
	Document  string
	Username  string
	Encrypted *conceal.Bytes
}

func (r *CreateUsernameRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	r.Username = strings.TrimSpace(r.Username)
	if r.Username == "" {
		return errors.New("username must be set")
	}
	if len(r.Username) > maxUsernameLength {
		return fmt.Errorf("username must not exceed %d characters", maxUsernameLength)
	}

	return nil
}

type UpdateUsernameRequest struct {
	ID        records.UsernamesAuto
	User      users.Identity
	Folder    string
	Document  string
	Username  string
	Encrypted *conceal.Bytes
}

func (r *UpdateUsernameRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("username must be set")
	}

	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	r.Username = strings.TrimSpace(r.Username)
	if r.Username == "" {
		return errors.New("username must be set")
	}
	if len(r.Username) > maxUsernameLength {
		return fmt.Errorf("username must not exceed %d characters", maxUsernameLength)
	}

	return nil
}

type DeleteUsernameRequest struct {
	ID       records.UsernamesAuto
	User     users.Identity
	Folder   string
	Document string
}

func (r *DeleteUsernameRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("username must be set")
	}

	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	return nil
}

type GetUsernameRequest struct {
	ID       records.UsernamesAuto
	User     users.Identity
	Folder   string
	Document string
}

func (r *GetUsernameRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("username must be set")
	}
	if r.User <= 0 {
		return errors.New("user must be set")
	}
	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}
	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}
	return nil
}

type CreatePasswordRequest struct {
	User      users.Identity
	Folder    string
	Document  string
	Encrypted *conceal.Bytes
}

func (r *CreatePasswordRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	return nil
}

// ValidatePassword checks the supplied plaintext against the password
// length and non-empty rules. It deliberately does not trim whitespace.
func ValidatePassword(raw string) error {
	if raw == "" {
		return errors.New("password must be set")
	}
	if len(raw) > maxPasswordLength {
		return fmt.Errorf("password must not exceed %d characters", maxPasswordLength)
	}
	return nil
}

type UpdatePasswordRequest struct {
	ID        records.PasswordsAuto
	User      users.Identity
	Folder    string
	Document  string
	Encrypted *conceal.Bytes
}

func (r *UpdatePasswordRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("password must be set")
	}

	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	return nil
}

type DeletePasswordRequest struct {
	ID       records.PasswordsAuto
	User     users.Identity
	Folder   string
	Document string
}

func (r *DeletePasswordRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("password must be set")
	}

	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	return nil
}

type GetPasswordRequest struct {
	ID       records.PasswordsAuto
	User     users.Identity
	Folder   string
	Document string
}

func (r *GetPasswordRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("password must be set")
	}
	if r.User <= 0 {
		return errors.New("user must be set")
	}
	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}
	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}
	return nil
}

// maxNoteLength bounds how long a note value may be. The notes table column
// is capped at 8192 bytes; encryption overhead (50 bytes for an authenticated
// block) gives us a comfortable plaintext ceiling around 6000 characters
const maxNoteLength = 6000

type CreateNoteRequest struct {
	User      users.Identity
	Folder    string
	Document  string
	Encrypted *conceal.Bytes
}

func (r *CreateNoteRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	return nil
}

type GetNoteRequest struct {
	User     users.Identity
	Folder   string
	Document string
}

func (r *GetNoteRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	return nil
}

func ValidateNote(raw string) error {
	if raw == "" {
		return errors.New("note must be set")
	}

	if len(raw) > maxNoteLength {
		return fmt.Errorf("note must not exceed %d characters", maxNoteLength)
	}

	return nil
}

type UpdateNoteRequest struct {
	ID        records.NotesAuto
	User      users.Identity
	Folder    string
	Document  string
	Encrypted *conceal.Bytes
}

func (r *UpdateNoteRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("note must be set")
	}

	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	return nil
}

type DeleteNoteRequest struct {
	ID       records.NotesAuto
	User     users.Identity
	Folder   string
	Document string
}

func (r *DeleteNoteRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("note must be set")
	}

	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	return nil
}

// maxKeyLength bounds how long the plaintext "k" of a keyval pair may be.
// Mirrors the username/password length, which is large enough for any
// realistic label and well within the database column < 1024 cap.
const maxKeyLength = 256

// maxKeyvalValueLength bounds how long the encrypted "v" of a keyval pair
// may be (plaintext, before encryption). Matches the password value cap.
const maxKeyvalValueLength = 1024

type CreateKeyvalRequest struct {
	User      users.Identity
	Folder    string
	Document  string
	Key       string
	Encrypted *conceal.Bytes
}

func (r *CreateKeyvalRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	r.Key = strings.TrimSpace(r.Key)
	if r.Key == "" {
		return errors.New("key must be set")
	}
	if len(r.Key) > maxKeyLength {
		return fmt.Errorf("key must not exceed %d characters", maxKeyLength)
	}

	return nil
}

type GetKeyvalRequest struct {
	ID       records.KeyvalsAuto
	User     users.Identity
	Folder   string
	Document string
}

func (r *GetKeyvalRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("key value must be set")
	}
	if r.User <= 0 {
		return errors.New("user must be set")
	}
	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}
	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}
	return nil
}

func ValidateKeyvalValue(value string) error {
	if value == "" {
		return errors.New("value must be set")
	}
	if len(value) > maxKeyvalValueLength {
		return fmt.Errorf("value must not exceed %d characters", maxKeyvalValueLength)
	}
	return nil
}

type UpdateKeyvalRequest struct {
	ID        records.KeyvalsAuto
	User      users.Identity
	Folder    string
	Document  string
	Key       string
	Encrypted *conceal.Bytes
}

func (r *UpdateKeyvalRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("key value must be set")
	}

	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	r.Key = strings.TrimSpace(r.Key)
	if r.Key == "" {
		return errors.New("key must be set")
	}
	if len(r.Key) > maxKeyLength {
		return fmt.Errorf("key must not exceed %d characters", maxKeyLength)
	}

	return nil
}

type DeleteKeyvalRequest struct {
	ID       records.KeyvalsAuto
	User     users.Identity
	Folder   string
	Document string
}

func (r *DeleteKeyvalRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("key value must be set")
	}

	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	return nil
}

type RenameDocumentRequest struct {
	User     users.Identity
	Folder   string
	Document string
	Label    string
	NewSlug  string
}

func (r *RenameDocumentRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	if !ValidateName(r.Label) {
		return errors.New("document must contain at least one letter or number")
	}

	if !ValidateSlug(r.NewSlug) {
		return errors.New("document must contain at least one letter or number")
	}

	return nil
}

type DeleteDocumentRequest struct {
	ID   records.DocumentsAuto
	User users.Identity
}

func (r *DeleteDocumentRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("document must be set")
	}

	if r.User <= 0 {
		return errors.New("user must be set")
	}

	return nil
}

type MigrateDocumentRequest struct {
	User     users.Identity
	Folder   string
	Document string
	Target   string
}

func (r *MigrateDocumentRequest) Validate() error {
	if r.User <= 0 {
		return errors.New("user must be set")
	}

	if !ValidateSlug(r.Folder) {
		return errors.New("folder must be set")
	}

	if !ValidateSlug(r.Document) {
		return errors.New("document must be set")
	}

	if !ValidateSlug(r.Target) {
		return errors.New("target folder must be set")
	}

	if r.Folder == r.Target {
		return errors.New("document is already in the target folder")
	}

	return nil
}
