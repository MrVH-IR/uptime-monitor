package monitor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"uptime-monitor/internal/database"

	"github.com/joho/godotenv"
)

func setupTestRepository(t *testing.T) (*Repository, context.Context) {
	t.Helper()

	if err := godotenv.Load("../../.env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed to load .env: %v", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	t.Cleanup(cancel)

	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	t.Cleanup(pool.Close)

	return NewRepository(pool), ctx
}

func createTestMonitor(
	t *testing.T,
	ctx context.Context,
	repo *Repository,
) *Monitor {
	t.Helper()

	id := time.Now().UnixNano()

	created, err := repo.Create(ctx, CreateMonitorInput{
		Name:            fmt.Sprintf("test-monitor-%d", id),
		URL:             fmt.Sprintf("https://example.com/%d", id),
		HTTPMethod:      "GET",
		IntervalSeconds: 60,
		TimeoutSeconds:  10,
	})
	if err != nil {
		t.Fatalf("failed to create test monitor: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := repo.Delete(cleanupCtx, created.ID); err != nil {
			t.Errorf("failed to delete test monitor: %v", err)
		}
	})

	return created
}

func TestRepository_ClaimDue(t *testing.T) {
	repo, ctx := setupTestRepository(t)

	created := createTestMonitor(t, ctx, repo)

	first, err := repo.ClaimDue(ctx, 10, 30*time.Second)
	if err != nil {
		t.Fatalf("first ClaimDue failed: %v", err)
	}

	if len(first) == 0 {
		t.Fatal("expected at least one monitor")
	}

	found := false

	for _, m := range first {
		if m.ID == created.ID {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("created monitor was not claimed")
	}

	second, err := repo.ClaimDue(ctx, 10, 30*time.Second)
	if err != nil {
		t.Fatalf("second ClaimDue failed: %v", err)
	}

	for _, m := range second {
		if m.ID == created.ID {
			t.Fatal("monitor was claimed more than once")
		}
	}
}

func TestRepository_ClaimDueConcurrent(t *testing.T) {
	repo, ctx := setupTestRepository(t)

	firstMonitor := createTestMonitor(t, ctx, repo)
	secondMonitor := createTestMonitor(t, ctx, repo)

	type claimResult struct {
		monitors []Monitor
		err      error
	}

	start := make(chan struct{})
	results := make(chan claimResult, 2)

	for range 2 {
		go func() {
			<-start

			monitors, err := repo.ClaimDue(ctx, 1, 30*time.Second)

			results <- claimResult{
				monitors: monitors,
				err:      err,
			}
		}()
	}

	close(start)

	first := <-results
	second := <-results

	if first.err != nil {
		t.Fatalf("first concurrent ClaimDue failed: %v", first.err)
	}

	if second.err != nil {
		t.Fatalf("second concurrent ClaimDue failed: %v", second.err)
	}

	if len(first.monitors) != 1 || len(second.monitors) != 1 {
		t.Fatalf(
			"expected 1 monitor per request, got %d and %d",
			len(first.monitors),
			len(second.monitors),
		)
	}

	claimedIDs := map[int64]bool{
		first.monitors[0].ID: true,
	}

	if claimedIDs[second.monitors[0].ID] {
		t.Fatal("both requests claimed the same monitor")
	}

	if !claimedIDs[firstMonitor.ID] && !claimedIDs[secondMonitor.ID] {
		t.Fatal("first request did not claim either test monitor")
	}

	if second.monitors[0].ID != firstMonitor.ID &&
		second.monitors[0].ID != secondMonitor.ID {
		t.Fatal("second request did not claim either test monitor")
	}
}

func TestRepository_Complete(t *testing.T) {
	repo, ctx := setupTestRepository(t)
	m := createTestMonitor(t, ctx, repo)
	started := time.Now().UTC().Truncate(time.Microsecond)
	finished := started.Add(time.Second)
	if err := repo.Complete(ctx, m.ID, 200, time.Second, true, "", "", started, finished); err != nil {
		t.Fatal(err)
	}
	updated, err := repo.GetByID(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.LastCheckedAt.Equal(finished) || !updated.NextCheckAt.Equal(finished.Add(60*time.Second)) {
		t.Fatal("incorrect check timestamps")
	}
	var count int
	var unlocked bool
	if err := repo.db.QueryRow(ctx, `SELECT COUNT(*) FROM check_results WHERE monitor_id=$1`, m.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err := repo.db.QueryRow(ctx, `SELECT locked_until IS NULL FROM monitors WHERE id=$1`, m.ID).Scan(&unlocked); err != nil {
		t.Fatal(err)
	}
	if count != 1 || !unlocked {
		t.Fatalf("results=%d unlocked=%v", count, unlocked)
	}
}
