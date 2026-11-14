package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"uptime-monitor/internal/monitor"
	"uptime-monitor/internal/worker"
)

type repository interface {
	ClaimDue(ctx context.Context, limit int, leaseDuration time.Duration) ([]monitor.Monitor, error)
}

type workerPool interface {
	Run(
		ctx context.Context,
		jobs <-chan monitor.Monitor,
	) <-chan worker.Result
}

type resultHandler func(context.Context, worker.Result) error

type Config struct {
	PollInterval  time.Duration
	BatchSize     int
	LeaseDuration time.Duration
}

type Scheduler struct {
	repository repository
	pool       workerPool
	config     Config
	handler    resultHandler
}

var (
	ErrInvalidRepository    = errors.New("repository is required")
	ErrInvalidPool          = errors.New("pool is required")
	ErrInvalidPoll          = errors.New("poll interval must be positive")
	ErrInvalidBatchSize     = errors.New("batch size must be positive")
	ErrInvalidLeaseDuration = errors.New("lease duration must be positive")
	ErrInvalidResultHandler = errors.New("result handler is required")
)

func New(repo repository, pool workerPool, config Config, handler resultHandler) (*Scheduler, error) {
	err := Check(repo, pool, config, handler)
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		repository: repo,
		pool:       pool,
		config:     config,
		handler:    handler,
	}, nil
}

func (s *Scheduler) Run(ctx context.Context) error {
	jobs := make(chan monitor.Monitor)

	results := s.pool.Run(ctx, jobs)

	var resultWG sync.WaitGroup
	resultWG.Add(1)

	go func() {
		defer resultWG.Done()
		for result := range results {
			if err := s.handler(ctx, result); err != nil {
				// Result persistence and error handling Will Be Added In The Next Stage
				continue
			}
		}
	}()

	defer func() {
		close(jobs)
		resultWG.Wait()
	}()

	tiker := time.NewTicker(s.config.PollInterval)
	defer tiker.Stop()

	if err := s.poll(ctx, jobs); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tiker.C:
			if err := s.poll(ctx, jobs); err != nil {
				return err
			}
		}
	}
}

func (s *Scheduler) poll(ctx context.Context, jobs chan<- monitor.Monitor) error {
	monitors, err := s.repository.ClaimDue(ctx, s.config.BatchSize, s.config.LeaseDuration)

	if err != nil {
		return fmt.Errorf("claim due monitors %w: ", err)
	}

	for _, m := range monitors {
		select {
		case <-ctx.Done():
			return nil

		case jobs <- m:
		}
	}
	return nil
}

func Check(repo repository, pool workerPool, config Config, handler resultHandler) error {
	if repo == nil {
		return ErrInvalidRepository
	}

	if pool == nil {
		return ErrInvalidPool
	}

	if config.PollInterval <= 0 {
		return ErrInvalidPoll
	}

	if config.BatchSize <= 0 {
		return ErrInvalidBatchSize
	}

	if config.LeaseDuration <= 0 {
		return ErrInvalidLeaseDuration
	}

	if handler == nil {
		return ErrInvalidResultHandler
	}

	return nil
}
