package format

import (
	"testing"

	"github.com/shoenig/test/must"
)

func TestBase32(t *testing.T) {
	t.Parallel()

	t.Run("100", func(t *testing.T) {
		result := Base32(100)
		must.Eq(t, "54", result)
	})

	t.Run("1737458569", func(t *testing.T) {
		result := Base32(1737458569)
		must.Eq(t, "0s1voj2", result)
	})

	t.Run("1737458628", func(t *testing.T) {
		result := Base32(1737458628)
		must.Eq(t, "5y1voj2", result)
	})
}
