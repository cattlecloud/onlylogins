// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sitecontent

import (
	"testing"

	"cattlecloud.net/onlylogins/pkg/users"
	"github.com/shoenig/test/must"
)

func TestCreateFolderRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("no user", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  0,
			Label: "work",
			Icon:  Icon(0),
		}
		err := r.Validate()
		must.ErrorContains(t, err, "user must be set")
	})

	t.Run("negative user", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(-1),
			Label: "work",
			Icon:  Icon(0),
		}
		err := r.Validate()
		must.ErrorContains(t, err, "user must be set")
	})

	t.Run("empty label", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "",
			Icon:  Icon(0),
		}
		err := r.Validate()
		must.ErrorContains(t, err, "label must contain only letters, numbers, and dashes")
	})

	t.Run("uppercase label", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "Work",
			Slug:  "work",
			Icon:  Icon(0),
		}
		err := r.Validate()
		must.NoError(t, err)
	})

	t.Run("label with dash", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "my-folder",
			Slug:  "my-folder",
			Icon:  Icon(0),
		}
		err := r.Validate()
		must.NoError(t, err)
	})

	t.Run("label with space", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "my folder",
			Icon:  Icon(0),
		}
		err := r.Validate()
		must.ErrorContains(t, err, "label must contain only letters, numbers, and dashes")
	})

	t.Run("label with accent", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "café",
			Slug:  "cafe",
			Icon:  Icon(0),
		}
		err := r.Validate()
		must.NoError(t, err)
	})

	t.Run("label with Jalapeño", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "Jalapeño",
			Slug:  "jalapeno",
			Icon:  Icon(0),
		}
		err := r.Validate()
		must.NoError(t, err)
	})

	t.Run("crazy letters only", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "日本語",
			Slug:  "",
			Icon:  Icon(0),
		}
		err := r.Validate()
		must.ErrorContains(t, err, "label must contain at least one letter or number")
	})

	t.Run("label of only dashes", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "---",
			Slug:  "",
			Icon:  Icon(0),
		}
		err := r.Validate()
		must.ErrorContains(t, err, "label must contain at least one letter or number")
	})

	t.Run("label with punctuation", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "***",
			Icon:  Icon(0),
		}
		err := r.Validate()
		must.ErrorContains(t, err, "label must contain only letters, numbers, and dashes")
	})

	t.Run("icon too low", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "work",
			Slug:  "work",
			Icon:  Icon(-1),
		}
		err := r.Validate()
		must.ErrorContains(t, err, "icon must be between 0 and 17")
	})

	t.Run("icon too high", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "work",
			Slug:  "work",
			Icon:  Icon(18),
		}
		err := r.Validate()
		must.ErrorContains(t, err, "icon must be between 0 and 17")
	})

	t.Run("ok", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "work",
			Slug:  "work",
			Icon:  Icon(0),
		}
		err := r.Validate()
		must.NoError(t, err)
	})

	t.Run("ok boundary icon high", func(t *testing.T) {
		r := &CreateFolderRequest{
			User:  users.Identity(1),
			Label: "work",
			Slug:  "work",
			Icon:  Icon(17),
		}
		err := r.Validate()
		must.NoError(t, err)
	})
}

func TestRenameFolderRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("no user", func(t *testing.T) {
		r := &RenameFolderRequest{
			User:    0,
			Slug:    "work",
			Label:   "home",
			NewSlug: "home",
		}
		err := r.Validate()
		must.ErrorContains(t, err, "user must be set")
	})

	t.Run("bad current slug", func(t *testing.T) {
		r := &RenameFolderRequest{
			User:    users.Identity(1),
			Slug:    "Work",
			Label:   "home",
			NewSlug: "home",
		}
		err := r.Validate()
		must.ErrorContains(t, err, "folder slug must contain only lowercase alphanumeric characters")
	})

	t.Run("empty label", func(t *testing.T) {
		r := &RenameFolderRequest{
			User:    users.Identity(1),
			Slug:    "work",
			Label:   "",
			NewSlug: "",
		}
		err := r.Validate()
		must.ErrorContains(t, err, "label must contain only letters, numbers, and dashes")
	})

	t.Run("label with space", func(t *testing.T) {
		r := &RenameFolderRequest{
			User:    users.Identity(1),
			Slug:    "work",
			Label:   "my folder",
			NewSlug: "my-folder",
		}
		err := r.Validate()
		must.ErrorContains(t, err, "label must contain only letters, numbers, and dashes")
	})

	t.Run("label with accent", func(t *testing.T) {
		r := &RenameFolderRequest{
			User:    users.Identity(1),
			Slug:    "work",
			Label:   "café",
			NewSlug: "cafe",
		}
		err := r.Validate()
		must.NoError(t, err)
	})

	t.Run("label slugifies empty", func(t *testing.T) {
		r := &RenameFolderRequest{
			User:    users.Identity(1),
			Slug:    "work",
			Label:   "---",
			NewSlug: "",
		}
		err := r.Validate()
		must.ErrorContains(t, err, "label must contain at least one letter or number")
	})

	t.Run("ok", func(t *testing.T) {
		r := &RenameFolderRequest{
			User:    users.Identity(1),
			Slug:    "work",
			Label:   "home",
			NewSlug: "home",
		}
		err := r.Validate()
		must.NoError(t, err)
	})
}
