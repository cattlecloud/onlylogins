// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package sitecontent

import (
	"strings"
	"testing"

	"github.com/shoenig/test/must"
)

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	t.Run("empty is rejected", func(t *testing.T) {
		err := ValidatePassword("")
		must.Error(t, err)
	})

	t.Run("whitespace only is accepted", func(t *testing.T) {
		// Whitespace can legitimately be part of a password secret and
		// must not be dropped silently — a literal whitespace-only
		// password is therefore a valid (if unlikely) value.
		err := ValidatePassword("   ")
		must.NoError(t, err)
	})

	t.Run("normal value is accepted", func(t *testing.T) {
		err := ValidatePassword("s3cret!")
		must.NoError(t, err)
	})

	t.Run("max length is accepted", func(t *testing.T) {
		err := ValidatePassword(strings.Repeat("a", maxPasswordLength))
		must.NoError(t, err)
	})

	t.Run("over max length is rejected", func(t *testing.T) {
		err := ValidatePassword(strings.Repeat("a", maxPasswordLength+1))
		must.Error(t, err)
	})
}
func TestValidateKeyvalValue(t *testing.T) {
	t.Parallel()

	t.Run("empty is rejected", func(t *testing.T) {
		err := ValidateKeyvalValue("")
		must.Error(t, err)
	})

	t.Run("whitespace only is accepted", func(t *testing.T) {
		// whitespace is valid in values, anywhere
		err := ValidateKeyvalValue("   ")
		must.NoError(t, err)
	})

	t.Run("normal value is accepted", func(t *testing.T) {
		err := ValidateKeyvalValue("s3cret!")
		must.NoError(t, err)
	})

	t.Run("max length is accepted", func(t *testing.T) {
		err := ValidateKeyvalValue(strings.Repeat("a", maxKeyvalValueLength))
		must.NoError(t, err)
	})

	t.Run("over max length is rejected", func(t *testing.T) {
		err := ValidateKeyvalValue(strings.Repeat("a", maxKeyvalValueLength+1))
		must.Error(t, err)
	})
}

func TestListDocumentsRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("missing user is rejected", func(t *testing.T) {
		r := &ListDocumentsRequest{Folder: "f"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing folder is rejected", func(t *testing.T) {
		r := &ListDocumentsRequest{User: 1}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("happy path", func(t *testing.T) {
		r := &ListDocumentsRequest{User: 1, Folder: "f"}
		err := r.Validate()
		must.NoError(t, err)
	})
}

func TestListWeblinksRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("missing user is rejected", func(t *testing.T) {
		r := &ListWeblinksRequest{Folder: "f", Document: "d"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing folder is rejected", func(t *testing.T) {
		r := &ListWeblinksRequest{User: 1, Document: "d"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing document is rejected", func(t *testing.T) {
		r := &ListWeblinksRequest{User: 1, Folder: "f"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("happy path", func(t *testing.T) {
		r := &ListWeblinksRequest{User: 1, Folder: "f", Document: "d"}
		err := r.Validate()
		must.NoError(t, err)
	})
}

func TestDeleteDocumentRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("missing id is rejected", func(t *testing.T) {
		r := &DeleteDocumentRequest{User: 1}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing user is rejected", func(t *testing.T) {
		r := &DeleteDocumentRequest{ID: 1}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("happy path", func(t *testing.T) {
		r := &DeleteDocumentRequest{ID: 1, User: 1}
		err := r.Validate()
		must.NoError(t, err)
	})
}

func TestRenameDocumentRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("missing user is rejected", func(t *testing.T) {
		r := &RenameDocumentRequest{Folder: "f", Document: "d", Label: "x", NewSlug: "x"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing folder is rejected", func(t *testing.T) {
		r := &RenameDocumentRequest{User: 1, Document: "d", Label: "x", NewSlug: "x"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing document is rejected", func(t *testing.T) {
		r := &RenameDocumentRequest{User: 1, Folder: "f", Label: "x", NewSlug: "x"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("empty label is rejected", func(t *testing.T) {
		r := &RenameDocumentRequest{User: 1, Folder: "f", Document: "d", Label: "", NewSlug: ""}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("whitespace only label is rejected", func(t *testing.T) {
		r := &RenameDocumentRequest{User: 1, Folder: "f", Document: "d", Label: "   ", NewSlug: ""}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("overlong label is rejected", func(t *testing.T) {
		r := &RenameDocumentRequest{User: 1, Folder: "f", Document: "d",
			Label: strings.Repeat("a", 32), NewSlug: strings.Repeat("a", 32)}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("label that slugifies empty is rejected", func(t *testing.T) {
		r := &RenameDocumentRequest{User: 1, Folder: "f", Document: "d", Label: "---", NewSlug: ""}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("empty new slug is rejected", func(t *testing.T) {
		r := &RenameDocumentRequest{User: 1, Folder: "f", Document: "d", Label: "Git Hub", NewSlug: ""}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("happy path", func(t *testing.T) {
		r := &RenameDocumentRequest{User: 1, Folder: "f", Document: "d", Label: "Git Hub", NewSlug: "git-hub"}
		err := r.Validate()
		must.NoError(t, err)
	})
}

func TestMigrateDocumentRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("missing user is rejected", func(t *testing.T) {
		r := &MigrateDocumentRequest{Folder: "f", Document: "d", Target: "t"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing folder is rejected", func(t *testing.T) {
		r := &MigrateDocumentRequest{User: 1, Document: "d", Target: "t"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing document is rejected", func(t *testing.T) {
		r := &MigrateDocumentRequest{User: 1, Folder: "f", Target: "t"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing target is rejected", func(t *testing.T) {
		r := &MigrateDocumentRequest{User: 1, Folder: "f", Document: "d"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("same-folder target is rejected", func(t *testing.T) {
		r := &MigrateDocumentRequest{User: 1, Folder: "f", Document: "d", Target: "f"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("happy path", func(t *testing.T) {
		r := &MigrateDocumentRequest{User: 1, Folder: "f", Document: "d", Target: "t"}
		err := r.Validate()
		must.NoError(t, err)
	})
}

func TestCreateKeyvalRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("missing user is rejected", func(t *testing.T) {
		r := &CreateKeyvalRequest{Folder: "f", Document: "d", Key: "k"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing folder is rejected", func(t *testing.T) {
		r := &CreateKeyvalRequest{User: 1, Document: "d", Key: "k"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing document is rejected", func(t *testing.T) {
		r := &CreateKeyvalRequest{User: 1, Folder: "f", Key: "k"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("empty key is rejected", func(t *testing.T) {
		r := &CreateKeyvalRequest{User: 1, Folder: "f", Document: "d", Key: "  "}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("whitespacey key is trimmed", func(t *testing.T) {
		r := &CreateKeyvalRequest{User: 1, Folder: "f", Document: "d", Key: "  api_key  "}
		err := r.Validate()
		must.NoError(t, err)
		must.Eq(t, "api_key", r.Key)
	})

	t.Run("overlong key is rejected", func(t *testing.T) {
		r := &CreateKeyvalRequest{User: 1, Folder: "f", Document: "d",
			Key: strings.Repeat("a", maxKeyLength+1)}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("happy path", func(t *testing.T) {
		r := &CreateKeyvalRequest{User: 1, Folder: "f", Document: "d",
			Key: "api_endpoint"}
		err := r.Validate()
		must.NoError(t, err)
	})
}

func TestUpdateKeyvalRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("missing id is rejected", func(t *testing.T) {
		r := &UpdateKeyvalRequest{User: 1, Folder: "f", Document: "d", Key: "k"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("empty key is rejected", func(t *testing.T) {
		r := &UpdateKeyvalRequest{ID: 1, User: 1, Folder: "f", Document: "d",
			Key: "   "}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("happy path trims key", func(t *testing.T) {
		r := &UpdateKeyvalRequest{ID: 1, User: 1, Folder: "f", Document: "d",
			Key: "  k  "}
		err := r.Validate()
		must.NoError(t, err)
		must.Eq(t, "k", r.Key)
	})
}

func TestDeleteKeyvalRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("missing id is rejected", func(t *testing.T) {
		r := &DeleteKeyvalRequest{User: 1, Folder: "f", Document: "d"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing user is rejected", func(t *testing.T) {
		r := &DeleteKeyvalRequest{ID: 1, Folder: "f", Document: "d"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing folder is rejected", func(t *testing.T) {
		r := &DeleteKeyvalRequest{ID: 1, User: 1, Document: "d"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("missing document is rejected", func(t *testing.T) {
		r := &DeleteKeyvalRequest{ID: 1, User: 1, Folder: "f"}
		err := r.Validate()
		must.Error(t, err)
	})

	t.Run("happy path", func(t *testing.T) {
		r := &DeleteKeyvalRequest{ID: 1, User: 1, Folder: "f", Document: "d"}
		err := r.Validate()
		must.NoError(t, err)
	})
}

func TestGetKeyvalRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		r := &GetKeyvalRequest{ID: 1, User: 1, Folder: "f", Document: "d"}
		err := r.Validate()
		must.NoError(t, err)
	})

	for _, tc := range []struct {
		name string
		r    *GetKeyvalRequest
	}{
		{"missing id", &GetKeyvalRequest{User: 1, Folder: "f", Document: "d"}},
		{"missing user", &GetKeyvalRequest{ID: 1, Folder: "f", Document: "d"}},
		{"missing folder", &GetKeyvalRequest{ID: 1, User: 1, Document: "d"}},
		{"missing document", &GetKeyvalRequest{ID: 1, User: 1, Folder: "f"}},
	} {
		t.Run(tc.name+" is rejected", func(t *testing.T) {
			err := tc.r.Validate()
			must.Error(t, err)
		})
	}
}
