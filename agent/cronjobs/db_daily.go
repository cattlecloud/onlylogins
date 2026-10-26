package cronjobs

import (
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/crons"
	"github.com/hashicorp/cronexpr"
)

func NewDailyDBM(
	config *crons.Config,
	store database.Store,
) *DailyDBM {
	return &DailyDBM{
		expression: cronexpr.MustParse(config.Expression),
		store:      store,
		log:        ulog.New("cron/db-daily"),
	}
}

type DailyDBM struct {
	store      database.Store
	expression *cronexpr.Expression
	log        *ulog.Log
}

func (c *DailyDBM) Name() string {
	return "db-daily"
}

func (c *DailyDBM) Schedule() *cronexpr.Expression {
	return c.expression
}

func (c *DailyDBM) Start() *crons.RunMetric {
	start := time.Now()

	ctx, cancel := scope.TTL(1 * time.Minute)
	defer cancel()

	if err := c.store.Maint().Optimize(ctx); err != nil {
		c.log.E.Fmt("failed to optimize: %v", err)
		return &crons.RunMetric{
			Code:  crons.ExitFailure,
			Start: start,
			Stop:  time.Now(),
		}
	}

	return &crons.RunMetric{
		Code:  crons.ExitSuccess,
		Start: start,
		Stop:  time.Now(),
	}
}
