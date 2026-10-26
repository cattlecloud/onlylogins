package cronjobs

import (
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/crons"
	"github.com/hashicorp/cronexpr"
)

func NewWeeklyDBM(
	config *crons.Config,
	store database.Store,
) *WeeklyDBM {
	return &WeeklyDBM{
		expression: cronexpr.MustParse(config.Expression),
		store:      store,
		log:        ulog.New("cron/db-daily"),
	}
}

type WeeklyDBM struct {
	store      database.Store
	expression *cronexpr.Expression
	log        *ulog.Log
}

func (c *WeeklyDBM) Name() string {
	return "db-weekly"
}

func (c *WeeklyDBM) Schedule() *cronexpr.Expression {
	return c.expression
}

func (c *WeeklyDBM) Start() *crons.RunMetric {
	start := time.Now()

	ctx, cancel := scope.TTL(1 * time.Minute)
	defer cancel()

	if err := c.store.Maint().Analyze(ctx); err != nil {
		c.log.E.Fmt("failed to analyze: %v", err)
		return &crons.RunMetric{
			Code:  crons.ExitFailure,
			Start: start,
			Stop:  time.Now(),
		}
	}

	if err := c.store.Maint().Compact(ctx); err != nil {
		c.log.E.Fmt("failed to compact: %v", err)
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
