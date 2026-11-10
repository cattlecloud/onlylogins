// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package dispatch

import (
	"fmt"
	"sync"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"cattlecloud.net/onlylogins/database"
	"cattlecloud.net/onlylogins/pkg/crons"
	"github.com/shoenig/lang"
)

const (
	// timeout is the general timeout policy for database operations
	// (must not include actual runtime of crons)
	timeout = 5 * time.Second
)

type Control struct {
	parent scope.C
	store  database.Crons
	log    *ulog.Log

	lock     *sync.Mutex
	cronjobs map[string]crons.Cronjob
}

func New(ctx scope.C, store database.Crons) *Control {
	return &Control{
		parent:   ctx,
		store:    store,
		log:      ulog.New("crons/dispatch"),
		lock:     new(sync.Mutex),
		cronjobs: make(map[string]crons.Cronjob),
	}
}

func (c *Control) Register(job crons.Cronjob) error {
	name := job.Name()

	// track the job; do not double register
	lang.Critical(c.lock, func() {
		c.cronjobs[name] = job
	})

	// establish context for database operations
	ctx, cancel := scope.WithTTL(c.parent, timeout)
	defer cancel()

	// register cron into database (initially pending state)
	if err := c.store.Register(ctx, name); err != nil {
		return err
	}

	c.log.I.Fmt("registration of %q complete", name)

	// start a goroutine monitoring this cronjob
	go c.monitor(job)

	return nil
}

func (c *Control) monitor(job crons.Cronjob) {
	// We must first get the cronjob into a consistent state - it could be
	// that we had an unexpected shutdown before leaving the cronjob in a
	// running state, etc.
	if err := c.restore(job); err != nil {
		c.log.E.Fmt("unable to restore cron: %v", err)
		return
	}

	// Start the normal wait loop
	for {
		previous := c.latest(job)
		next := job.Schedule().Next(previous)
		wait := max(0, time.Until(next))

		c.log.I.Fmt("next launch of %s in %v", job.Name(), wait.Truncate(time.Second))

		select {
		case <-c.parent.Done():
			c.log.I.Fmt("monitor for %q shutting down", job.Name())
			return
		case <-time.After(wait):
			c.log.D.Fmt("time to run %s", job.Name())
		}

		// make sure cron is in expected condition before starting
		if err := c.preflight(job); err != nil {
			c.log.W.Fmt("aborting cron run; preflight failed: %v", err)
			return
		}

		// ready set go!
		c.log.I.Fmt("executing cron %s", job.Name())
		c.execute(job)
		c.log.I.Fmt("finished cron %s", job.Name())
	}
}

func (c *Control) latest(job crons.Cronjob) time.Time {
	ctx, cancel := scope.WithTTL(c.parent, timeout)
	defer cancel()

	l, lerr := c.store.Latest(ctx, job.Name())
	if lerr != nil {
		c.log.E.Fmt("unable to get latest %s cron run: %v", job.Name(), lerr)
		return time.Now()
	}

	return l.Stop
}

func (c *Control) preflight(job crons.Cronjob) error {
	ctx, cancel := scope.WithTTL(c.parent, timeout)
	defer cancel()

	status, serr := c.store.Get(ctx, job.Name())
	if serr != nil {
		return fmt.Errorf("unable to get %s status from database: %w", job.Name(), serr)
	}

	switch status.State {
	case crons.StateStopped:
		return nil
	default:
		return fmt.Errorf("cron preflight: in unexpected state: %s", status.State)
	}
}

func (c *Control) restore(job crons.Cronjob) error {
	ctx, cancel := scope.WithTTL(c.parent, timeout)
	defer cancel()

	status, serr := c.store.Get(ctx, job.Name())
	if serr != nil {
		return fmt.Errorf("unable to get %s status from database: %w", job.Name(), serr)
	}

	switch status.State {
	case crons.StateRunning:
		c.reset(job)
	case crons.StatePending:
		c.log.I.Fmt("cron %s is pending; will run immediately", job.Name())
		c.execute(job)
	case crons.StateDisable:
		c.log.I.Fmt("cron %s is disabled; will not execute", job.Name())
	case crons.StateStopped:
		c.log.I.Fmt("cron %s is stopped; will run regularly", job.Name())
	default:
		return fmt.Errorf("no such cron state %q", status.State)
	}

	return nil
}

func (c *Control) reset(job crons.Cronjob) {
	ctx, cancel := scope.WithTTL(c.parent, timeout)
	defer cancel()

	c.log.I.Fmt("doing a reset of %s", job.Name())

	if err := c.store.Reset(ctx, job.Name()); err != nil {
		c.log.E.Fmt("unable to reset cron %s: %v", job.Name(), err)
	}
}

func (c *Control) execute(job crons.Cronjob) {
	ctx, cancel := scope.WithCancel(c.parent)
	defer cancel()

	// update database to set running status
	if err := c.store.Start(ctx, job.Name()); err != nil {
		c.log.E.Fmt("unable to start cron %s: %v", job.Name(), err)
		return
	}

	// actually run the cronjob
	metric := job.Start()

	// update the database to set stopped status
	if err := c.store.Stop(ctx, job.Name(), metric); err != nil {
		c.log.E.Fmt("unable to stop cron %s: %v", job.Name(), err)
		return
	}
}
