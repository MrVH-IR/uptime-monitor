package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"uptime-monitor/internal/monitor"
)

func TestHTTPChecker_CheckSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHttpChecker()

	result := checker.Check(context.Background(), monitor.Monitor{
		URL:            server.URL,
		HTTPMethod:     http.MethodGet,
		TimeoutSeconds: 5,
	})

	if !result.IsSuccess {
		t.Fatalf("expected success, got error: %s", result.ErrorMessage)
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.StatusCode)
	}

	if result.ResponseTime <= 0 {
		t.Errorf("expected positive response time, got %v", result.ResponseTime)
	}

	if result.StartedAt.IsZero() || result.FinishedAt.IsZero() {
		t.Error("expected StartedAt and FinishedAt to be set")
	}
}

func TestHTTPChecker_CheckHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	checker := NewHttpChecker()

	result := checker.Check(context.Background(), monitor.Monitor{
		URL:            server.URL,
		HTTPMethod:     http.MethodGet,
		TimeoutSeconds: 5,
	})

	if result.IsSuccess {
		t.Fatal("expected failure for HTTP 503")
	}

	if result.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", result.StatusCode)
	}

	if result.ErrorType != "http_status" {
		t.Errorf("expected error type http_status, got %s", result.ErrorType)
	}
}

func TestHTTPChecker_CheckTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	checker := NewHttpChecker()

	result := checker.Check(context.Background(), monitor.Monitor{
		URL:            server.URL,
		HTTPMethod:     http.MethodGet,
		TimeoutSeconds: 1,
	})

	if result.IsSuccess {
		t.Fatal("expected timeout failure")
	}

	if result.ErrorType != "timeout" {
		t.Errorf("expected error type timeout, got %s", result.ErrorType)
	}

	if result.FinishedAt.IsZero() {
		t.Error("expected FinishedAt to be set")
	}
}
