package users

import (
	"regexp"
	"testing"
	"testing/synctest"
	"time"

	"github.com/hashicorp/go-set/v3"
	"github.com/shoenig/test/must"
)

func TestProfile_Active(t *testing.T) {
	t.Parallel()

	t.Run("yes", func(t *testing.T) {
		p := &Profile{
			ID: 1,
		}
		must.True(t, p.Active())
	})

	t.Run("no", func(t *testing.T) {
		p := (*Profile)(nil)
		must.False(t, p.Active())
	})
}

func TestProfile_Identity(t *testing.T) {
	t.Parallel()

	t.Run("active", func(t *testing.T) {
		p := &Profile{
			ID: 3,
		}
		must.Eq(t, 3, p.Identity().(Identity))
	})

	t.Run("inactive", func(t *testing.T) {
		p := (*Profile)(nil)
		must.Eq(t, Inactive, p.Identity())
	})
}

func TestGenerateUsername(t *testing.T) {
	t.Parallel()

	t.Run("format", func(t *testing.T) {
		now := time.Date(2025, 1, 21, 5, 30, 0, 0, time.UTC)
		name := GenerateUsername(now)
		var nameRe = regexp.MustCompile(`user_o7cyoj2[a-z]{2}`)
		must.RegexMatch(t, nameRe, name)
	})

	t.Run("duplicates", func(t *testing.T) {
		used := set.New[string](3)
		synctest.Test(t, func(t *testing.T) {
			for i := range 100 {
				now := time.Now()
				name := GenerateUsername(now)
				if used.Contains(name) {
					t.Fatalf("conflict i: %d, name: %s", i, name)
				}
				used.Insert(name)
				time.Sleep(1 * time.Second)
			}
		})
	})
}
