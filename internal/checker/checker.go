package checker

import (
	"context"
	"time"
	"uptime-monitor/internal/monitor"
)

type Result struct {
	StatusCode   int
	ResponseTime time.Duration
	IsSuccess    bool
	ErrorType    string
	ErrorMessage string
	StartedAt    time.Time
	FinishedAt   time.Time
}

type Checker interface {
	Check(ctx context.Context, target monitor.Monitor) Result
}
