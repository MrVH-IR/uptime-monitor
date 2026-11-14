package monitor

import "time"

type Monitor struct {
	ID              int64
	Name            string
	URL             string
	HTTPMethod      string
	IntervalSeconds int
	TimeoutSeconds  int
	IsActive        bool
	NextCheckAt     *time.Time
	LastCheckedAt   *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CreateMonitorInput struct {
	Name            string
	URL             string
	HTTPMethod      string
	IntervalSeconds int
	TimeoutSeconds  int
}

type UpdateMonitorInput struct {
	Name            string
	URL             string
	HTTPMethod      string
	IntervalSeconds int
	TimeoutSeconds  int
	IsActive        bool
}
