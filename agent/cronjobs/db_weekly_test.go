package cronjobs

import (
	"testing"

	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/crons"
	"github.com/shoenig/test/must"
)

func TestWeeklyDBM_Start(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	weekly := NewWeeklyDBM(
		&crons.Config{Expression: "* * * * *"},
		store,
	)

	result := weekly.Start()
	must.NotNil(t, result)
	must.Eq(t, crons.ExitSuccess, result.Code)
}
