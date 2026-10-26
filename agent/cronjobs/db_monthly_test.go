package cronjobs

import (
	"testing"

	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/crons"
	"github.com/shoenig/test/must"
)

func TestMonthlyDBM_Start(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	monthly := NewMonthlyDBM(
		&crons.Config{Expression: "* * * * *"},
		store,
	)

	result := monthly.Start()
	must.NotNil(t, result)
	must.Eq(t, crons.ExitSuccess, result.Code)
}
