package cronjobs

import (
	"testing"

	"cattlecloud.net/onlylogins/hack/mocks/mockdb"
	"cattlecloud.net/onlylogins/pkg/crons"
	"github.com/shoenig/test/must"
)

func TestDailyDBM_Start(t *testing.T) {
	t.Parallel()

	store := mockdb.Open(t)

	daily := NewDailyDBM(
		&crons.Config{Expression: "* * * * *"},
		store,
	)

	result := daily.Start()
	must.NotNil(t, result)
	must.Eq(t, crons.ExitSuccess, result.Code)
}
