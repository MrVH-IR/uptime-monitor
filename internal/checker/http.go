package checker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
	"uptime-monitor/internal/monitor"
)

type HttpChecker struct {
	client *http.Client
}

func NewHttpChecker() *HttpChecker {
	return &HttpChecker{
		client: &http.Client{
			Transport: http.DefaultTransport,
		},
	}
}

func (c *HttpChecker) Check(ctx context.Context, target monitor.Monitor) Result {
	startedAt := time.Now()
	result := Result{
		StartedAt: startedAt,
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(target.TimeoutSeconds)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, target.HTTPMethod, target.URL, nil)

	if err != nil {
		return c.failure(result, "request", err)
	}

	response, err := c.client.Do(req)
	if err != nil {
		return c.failure(result, classifyError(err), err)
	}
	defer response.Body.Close()
	result.StatusCode = response.StatusCode
	result.ResponseTime = time.Since(startedAt)
	result.FinishedAt = time.Now()

	result.IsSuccess = response.StatusCode >= 200 && response.StatusCode < 400
	if !result.IsSuccess {
		result.ErrorType = "http_status"
		result.ErrorMessage = fmt.Sprintf("unexpected status code: %d", response.StatusCode)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64*1024))

	return result
}

func (c *HttpChecker) failure(result Result, errType string, err error) Result {
	result.ResponseTime = time.Since(result.StartedAt)
	result.FinishedAt = time.Now()
	result.IsSuccess = false
	result.ErrorType = errType
	result.ErrorMessage = truncateError(err.Error(), 500)

	return result
}

func classifyError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}

	return "network"
}

func truncateError(message string, maxLength int) string {
	message = strings.TrimSpace(message)

	if len(message) > maxLength {
		return message[:maxLength]
	}

	return message
}
