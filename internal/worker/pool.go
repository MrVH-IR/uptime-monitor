package worker

import (
	"context"
	"errors"
	"sync"
	"uptime-monitor/internal/checker"
	"uptime-monitor/internal/monitor"
)

var (
	ErrInvalidWorkerCount = errors.New("worker count must be greater than zero")
	ErrNilChecker         = errors.New("checker cannot be nil")
)

type Result struct {
	Monitor monitor.Monitor
	Check   checker.Result
}

type Pool struct {
	workersCount int
	checker      checker.Checker
}

func NewPool(workerCount int, checker checker.Checker) (*Pool, error) {
	if workerCount <= 0 {
		return nil, ErrInvalidWorkerCount
	}

	if checker == nil {
		return nil, ErrNilChecker
	}

	return &Pool{
		workersCount: workerCount,
		checker:      checker,
	}, nil
}

func (pool *Pool) Run(ctx context.Context, jobs <-chan monitor.Monitor) <-chan Result {
	results := make(chan Result)

	var wg sync.WaitGroup

	for range pool.workersCount {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case target, ok := <-jobs:
					if !ok {
						return
					}
					if ctx.Err() != nil {
						return
					}

					checkResult := pool.checker.Check(ctx, target)

					result := Result{Monitor: target, Check: checkResult}

					select {
					case results <- result:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}
