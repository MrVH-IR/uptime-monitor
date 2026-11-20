package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"uptime-monitor/internal/monitor"
)

type CheckResult struct {
	ID             int64
	MonitorID      int64
	StatusCode     *int
	ResponseTimeMS *int64
	IsSuccess      bool
	ErrorType      string
	ErrorMessage   string
	StartedAt      time.Time
	FinishedAt     time.Time
}

func New(db *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	repo := monitor.NewRepository(db)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/monitors", func(w http.ResponseWriter, r *http.Request) {
		page, ok := readPage(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		monitors, err := repo.ListPage(ctx, (page-1)*50)
		if err != nil {
			respond(w, nil, err)
			return
		}
		paginated(w, monitors, page)
	})
	mux.HandleFunc("GET /api/results", func(w http.ResponseWriter, r *http.Request) {
		var id int64
		if value := r.URL.Query().Get("monitor_id"); value != "" {
			var err error
			id, err = strconv.ParseInt(value, 10, 64)
			if err != nil || id <= 0 {
				http.Error(w, "invalid monitor_id", http.StatusBadRequest)
				return
			}
		}
		page, ok := readPage(w, r)
		if !ok {
			return
		}
		var ids []int64
		if raw := r.URL.Query().Get("monitor_ids"); raw != "" {
			values := strings.Split(raw, ",")
			if len(values) > 50 || id != 0 {
				http.Error(w, "invalid monitor_ids", 400)
				return
			}
			for _, value := range values {
				parsed, err := strconv.ParseInt(value, 10, 64)
				if err != nil || parsed <= 0 {
					http.Error(w, "invalid monitor_ids", 400)
					return
				}
				ids = append(ids, parsed)
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		// Without a filter return the latest result of every monitor; with one return its recent history.
		query := `SELECT DISTINCT ON (monitor_id) id, monitor_id, status_code, response_time_ms, is_success, error_type, error_message, started_at, finished_at FROM check_results ORDER BY monitor_id, started_at DESC, id DESC`
		query = `SELECT * FROM (` + query + `) latest ORDER BY started_at DESC, id DESC LIMIT 51 OFFSET $1`
		args := []any{(page - 1) * 50}
		if id != 0 {
			query = `SELECT id, monitor_id, status_code, response_time_ms, is_success, error_type, error_message, started_at, finished_at FROM check_results WHERE monitor_id=$1 ORDER BY started_at DESC, id DESC LIMIT 51 OFFSET $2`
			args = []any{id, (page - 1) * 50}
		}
		if len(ids) > 0 {
			query = `SELECT * FROM (SELECT DISTINCT ON (monitor_id) id, monitor_id, status_code, response_time_ms, is_success, error_type, error_message, started_at, finished_at FROM check_results WHERE monitor_id=ANY($1) ORDER BY monitor_id,started_at DESC,id DESC) latest ORDER BY started_at DESC,id DESC LIMIT 51 OFFSET $2`
			args = []any{ids, (page - 1) * 50}
		}
		rows, err := db.Query(ctx, query, args...)
		if err != nil {
			respond(w, nil, err)
			return
		}
		defer rows.Close()
		results := make([]CheckResult, 0)
		for rows.Next() {
			var c CheckResult
			if err := rows.Scan(&c.ID, &c.MonitorID, &c.StatusCode, &c.ResponseTimeMS, &c.IsSuccess, &c.ErrorType, &c.ErrorMessage, &c.StartedAt, &c.FinishedAt); err != nil {
				respond(w, nil, err)
				return
			}
			results = append(results, c)
		}
		if err := rows.Err(); err != nil {
			respond(w, nil, err)
			return
		}
		paginated(w, results, page)
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	mux.Handle("/", http.FileServer(http.Dir("frontend/dist")))
	return mux
}

func respond(w http.ResponseWriter, value any, err error) {
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("API request failed", "error", err)
		}
		http.Error(w, "unable to load data", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("encode API response", "error", err)
	}
}

func readPage(w http.ResponseWriter, r *http.Request) (int, bool) {
	value := r.URL.Query().Get("page")
	if value == "" {
		return 1, true
	}
	page, err := strconv.Atoi(value)
	if err != nil || page < 1 || page > 1000000 {
		http.Error(w, "page must be between 1 and 1000000", http.StatusBadRequest)
		return 0, false
	}
	return page, true
}

func paginated[T any](w http.ResponseWriter, items []T, page int) {
	more := len(items) > 50
	if more {
		items = items[:50]
	}
	w.Header().Set("X-Page", strconv.Itoa(page))
	w.Header().Set("X-Page-Size", "50")
	w.Header().Set("X-Has-More", strconv.FormatBool(more))
	respond(w, items, nil)
}
