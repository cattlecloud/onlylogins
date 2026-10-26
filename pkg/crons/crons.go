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
