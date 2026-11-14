package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"uptime-monitor/internal/checker"
	"uptime-monitor/internal/monitor"
)

type mockChecker struct {
	active atomic.Int32
	max    atomic.Int32
}

func (m *mockChecker) Check(
	ctx context.Context,
	target monitor.Monitor,
) checker.Result {
	current := m.active.Add(1)
	defer m.active.Add(-1)

	for {
		max := m.max.Load()

		if current <= max {
			break
		}

		if m.max.CompareAndSwap(max, current) {
			break
		}
	}

	select {
	case <-ctx.Done():
	case <-time.After(50 * time.Millisecond):
	}

	return checker.Result{
		IsSuccess: true,
	}
}

func TestPool_Run(t *testing.T) {
	const (
		workerCount = 3
		jobCount    = 10
	)

	mock := &mockChecker{}

	pool, err := NewPool(workerCount, mock)
	if err != nil {
		t.Fatal(err)
	}

	jobs := make(chan monitor.Monitor, jobCount)

	for i := 0; i < jobCount; i++ {
		jobs <- monitor.Monitor{
			ID: int64(i + 1),
		}
	}

	close(jobs)

	results := pool.Run(context.Background(), jobs)

	var completed int

	for result := range results {
		completed++

		if !result.Check.IsSuccess {
			t.Error("expected successful check")
		}
	}

	if completed != jobCount {
		t.Errorf(
			"expected %d results, got %d",
			jobCount,
			completed,
		)
	}

	if got := mock.max.Load(); got > workerCount {
		t.Errorf(
			"expected maximum %d concurrent workers, got %d",
			workerCount,
			got,
		)
	}

	if got := mock.max.Load(); got < 2 {
		t.Errorf(
			"expected concurrent execution, maximum concurrency was %d",
			got,
		)
	}
}

func TestNewPool_InvalidWorkerCount(t *testing.T) {
	_, err := NewPool(0, &mockChecker{})

	if !errors.Is(err, ErrInvalidWorkerCount) {
		t.Fatalf("expected ErrInvalidWorkerCount, got %v", err)
	}
}
