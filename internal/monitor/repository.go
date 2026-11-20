package monitor

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, input CreateMonitorInput) (*Monitor, error) {
	query := `
        INSERT INTO monitors (
            name,
            url,
            http_method,
            interval_seconds,
            timeout_seconds
        )
        VALUES ($1, $2, $3, $4, $5)
        RETURNING
            id,
            name,
            url,
            http_method,
            interval_seconds,
            timeout_seconds,
            is_active,
            next_check_at,
            last_checked_at,
            created_at,
            updated_at
    `

	var m Monitor

	err := r.db.QueryRow(
		ctx,
		query,
		input.Name,
		input.URL,
		input.HTTPMethod,
		input.IntervalSeconds,
		input.TimeoutSeconds,
	).Scan(
		&m.ID,
		&m.Name,
		&m.URL,
		&m.HTTPMethod,
		&m.IntervalSeconds,
		&m.TimeoutSeconds,
		&m.IsActive,
		&m.NextCheckAt,
		&m.LastCheckedAt,
		&m.CreatedAt,
		&m.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Monitor, error) {
	query := `
        SELECT
            id,
            name,
            url,
            http_method,
            interval_seconds,
            timeout_seconds,
            is_active,
            next_check_at,
            last_checked_at,
            created_at,
            updated_at
        FROM monitors
        WHERE id = $1
    `

	var m Monitor

	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.Name,
		&m.URL,
		&m.HTTPMethod,
		&m.IntervalSeconds,
		&m.TimeoutSeconds,
		&m.IsActive,
		&m.NextCheckAt,
		&m.LastCheckedAt,
		&m.CreatedAt,
		&m.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (r *Repository) List(ctx context.Context) ([]Monitor, error) { return r.list(ctx, 0, 0) }
func (r *Repository) ListPage(ctx context.Context, offset int) ([]Monitor, error) {
	return r.list(ctx, 51, offset)
}

func (r *Repository) list(ctx context.Context, limit, offset int) ([]Monitor, error) {
	query := `
        SELECT
            id,
            name,
            url,
            http_method,
            interval_seconds,
            timeout_seconds,
            is_active,
            next_check_at,
            last_checked_at,
            created_at,
            updated_at
        FROM monitors
        ORDER BY id DESC LIMIT NULLIF($1, 0) OFFSET $2
    `

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	monitors := make([]Monitor, 0)

	for rows.Next() {
		var m Monitor

		err := rows.Scan(
			&m.ID,
			&m.Name,
			&m.URL,
			&m.HTTPMethod,
			&m.IntervalSeconds,
			&m.TimeoutSeconds,
			&m.IsActive,
			&m.NextCheckAt,
			&m.LastCheckedAt,
			&m.CreatedAt,
			&m.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		monitors = append(monitors, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return monitors, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(
		ctx,
		`DELETE FROM monitors WHERE id = $1`,
		id,
	)

	return err
}

func (r *Repository) UpdateNextCheck(
	ctx context.Context,
	id int64,
	nextCheckAt time.Time,
) error {
	_, err := r.db.Exec(
		ctx,
		`
        UPDATE monitors
        SET next_check_at = $1,
            updated_at = NOW()
        WHERE id = $2
        `,
		nextCheckAt,
		id,
	)

	return err
}

func (r *Repository) ClaimDue(ctx context.Context, limit int, leaseDuration time.Duration) ([]Monitor, error) {
	query := `WITH due_monitors AS (SELECT id FROM monitors WHERE is_active = TRUE AND next_check_at <= NOW()
				AND (locked_until IS NULL OR locked_until < NOW()) ORDER BY next_check_at ASC FOR UPDATE SKIP LOCKED LIMIT $1)
				UPDATE monitors AS m SET locked_until = NOW() + (($2::double precision + (SELECT COALESCE(SUM(timeout_seconds),0) FROM monitors WHERE id IN (SELECT id FROM due_monitors))) * INTERVAL '1 second') FROM due_monitors AS d WHERE m.id = d.id
				RETURNING m.id, m.name, m.url, m.http_method, m.interval_seconds, m.timeout_seconds, m.is_active, m.next_check_at,
				m.last_checked_at, m.created_at, m.updated_at
			`
	rows, err := r.db.Query(ctx, query, limit, leaseDuration.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	monitors := make([]Monitor, 0)

	for rows.Next() {
		var m Monitor
		err := rows.Scan(
			&m.ID,
			&m.Name,
			&m.URL,
			&m.HTTPMethod,
			&m.IntervalSeconds,
			&m.TimeoutSeconds,
			&m.IsActive,
			&m.NextCheckAt,
			&m.LastCheckedAt,
			&m.CreatedAt,
			&m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return monitors, nil
}

// Complete records the result and reschedules the monitor atomically.
func (r *Repository) Complete(ctx context.Context, id int64, statusCode int, responseTime time.Duration, success bool, errorType, errorMessage string, startedAt, finishedAt time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO check_results (monitor_id, status_code, response_time_ms, is_success, error_type, error_message, started_at, finished_at)
 VALUES ($1, NULLIF($2, 0), $3, $4, $5, $6, $7, $8)`, id, statusCode, responseTime.Milliseconds(), success, errorType, errorMessage, startedAt, finishedAt)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE monitors SET last_checked_at = $2::timestamptz, next_check_at = $2::timestamptz + interval_seconds * INTERVAL '1 second', locked_until = NULL, updated_at = NOW() WHERE id = $1`, id, finishedAt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Update(ctx context.Context, id int64, input UpdateMonitorInput) (*Monitor, error) {
	var m Monitor
	err := r.db.QueryRow(ctx, `UPDATE monitors SET name=$2, url=$3, http_method=$4, interval_seconds=$5, timeout_seconds=$6, is_active=$7, updated_at=NOW() WHERE id=$1
 RETURNING id, name, url, http_method, interval_seconds, timeout_seconds, is_active, next_check_at, last_checked_at, created_at, updated_at`, id, input.Name, input.URL, input.HTTPMethod, input.IntervalSeconds, input.TimeoutSeconds, input.IsActive).Scan(&m.ID, &m.Name, &m.URL, &m.HTTPMethod, &m.IntervalSeconds, &m.TimeoutSeconds, &m.IsActive, &m.NextCheckAt, &m.LastCheckedAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
