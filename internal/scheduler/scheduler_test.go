package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"uptime-monitor/internal/monitor"
	"uptime-monitor/internal/worker"
)

type fakeRepository struct {
	claimFn func(
		context.Context,
		int,
		time.Duration,
	) ([]monitor.Monitor, error)
}

func (f *fakeRepository) ClaimDue(
	ctx context.Context,
	limit int,
	leaseDuration time.Duration,
) ([]monitor.Monitor, error) {
	return f.claimFn(ctx, limit, leaseDuration)
}

type fakeWorkerPool struct{}

func (f *fakeWorkerPool) Run(
	ctx context.Context,
	jobs <-chan monitor.Monitor,
) <-chan worker.Result {
	results := make(chan worker.Result)

	go func() {
		defer close(results)

		for range jobs {
		}
	}()

	return results
}

func TestScheduler_RunPollsImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	calls := 0

	repo := &fakeRepository{
		claimFn: func(
			ctx context.Context,
			limit int,
			leaseDuration time.Duration,
		) ([]monitor.Monitor, error) {
			calls++
			cancel()

			return nil, nil
		},
	}

	scheduler, err := New(
		repo,
		&fakeWorkerPool{},
		Config{
			PollInterval:  time.Hour,
			BatchSize:     10,
			LeaseDuration: time.Minute,
		},
		func(context.Context, worker.Result) error {
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := scheduler.Run(ctx); err != nil {
		t.Fatal(err)
	}

	if calls != 1 {
		t.Fatalf("expected 1 poll, got %d", calls)
	}
}

func TestScheduler_RunPollsPeriodically(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	calls := 0

	repo := &fakeRepository{
		claimFn: func(
			ctx context.Context,
			limit int,
			leaseDuration time.Duration,
		) ([]monitor.Monitor, error) {
			calls++

			if calls == 2 {
				cancel()
			}

			return nil, nil
		},
	}

	scheduler, err := New(
		repo,
		&fakeWorkerPool{},
		Config{
			PollInterval:  10 * time.Millisecond,
			BatchSize:     10,
			LeaseDuration: time.Minute,
		},
		func(context.Context, worker.Result) error {
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := scheduler.Run(ctx); err != nil {
		t.Fatal(err)
	}

	if calls != 2 {
		t.Fatalf("expected 2 polls, got %d", calls)
	}
}

func TestScheduler_RunReturnsClaimError(t *testing.T) {
	expectedErr := errors.New("database unavailable")

	repo := &fakeRepository{
		claimFn: func(
			context.Context,
			int,
			time.Duration,
		) ([]monitor.Monitor, error) {
			return nil, expectedErr
		},
	}

	scheduler, err := New(
		repo,
		&fakeWorkerPool{},
		Config{
			PollInterval:  time.Minute,
			BatchSize:     10,
			LeaseDuration: time.Minute,
		},
		func(context.Context, worker.Result) error {
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = scheduler.Run(context.Background())

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}
