// Copyright (c) 2026 Cattle Cloud LLC
// SPDX-License-Identifier: AGPL-3.0

package crons

import (
	"time"

	"github.com/hashicorp/cronexpr"
)

type State string

const (
	StateDisable State = "disable"
	StateRunning State = "running"
	StateStopped State = "stopped"
	StatePending State = "pending"
)

type Cronjob interface {
	Name() string
	Start() *RunMetric
	Schedule() *cronexpr.Expression
}

type RunResult struct {
	CurrentStatus
	RunMetric
}

type CurrentStatus struct {
	ID    int
	Label string
	State State
}

type RunMetric struct {
	Code  ExitCode
	Start time.Time
	Stop  time.Time
}

type ExitCode int

const ExitSuccess = 0
const ExitFailure = 1

func Success(start time.Time) *RunMetric {
	return &RunMetric{
		Code:  ExitSuccess,
		Start: start,
		Stop:  time.Now(),
	}
}

func Failure(start time.Time) *RunMetric {
	return &RunMetric{
		Code:  ExitFailure,
		Start: start,
		Stop:  time.Now(),
	}
}
