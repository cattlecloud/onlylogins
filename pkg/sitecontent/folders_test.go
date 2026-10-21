package sitecontent

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestValidateLabel(t *testing.T) {
	t.Parallel()

	t.Run("letters", func(t *testing.T) {
		must.NoError(t, ValidateLabel("FooBar"))
	})

	t.Run("digits", func(t *testing.T) {
		must.NoError(t, ValidateLabel("12345"))
	})

	t.Run("mixed", func(t *testing.T) {
		must.NoError(t, ValidateLabel("R2D2"))
	})

	t.Run("empty", func(t *testing.T) {
		must.Error(t, ValidateLabel(""))
	})

	t.Run("space", func(t *testing.T) {
		must.Error(t, ValidateLabel("foo bar"))
	})

	t.Run("hyphen", func(t *testing.T) {
		must.Error(t, ValidateLabel("foo-bar"))
	})

	t.Run("underscore", func(t *testing.T) {
		must.Error(t, ValidateLabel("foo_bar"))
	})

	t.Run("slash", func(t *testing.T) {
		must.Error(t, ValidateLabel("foo/bar"))
	})

	t.Run("unicode", func(t *testing.T) {
		must.Error(t, ValidateLabel("café"))
	})
}
