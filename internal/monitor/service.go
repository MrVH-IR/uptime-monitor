package monitor

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

var ErrInvalidInput = errors.New("invalid monitor input")

type repository interface {
	Create(context.Context, CreateMonitorInput) (*Monitor, error)
	GetByID(context.Context, int64) (*Monitor, error)
	List(context.Context) ([]Monitor, error)
	Update(context.Context, int64, UpdateMonitorInput) (*Monitor, error)
	Delete(context.Context, int64) error
}

type Service struct {
	repository repository
}

func NewService(repository repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(
	ctx context.Context,
	input CreateMonitorInput,
) (*Monitor, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.URL = strings.TrimSpace(input.URL)
	input.HTTPMethod = strings.ToUpper(strings.TrimSpace(input.HTTPMethod))

	if input.HTTPMethod == "" {
		input.HTTPMethod = "GET"
	}

	if input.IntervalSeconds == 0 {
		input.IntervalSeconds = 60
	}

	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 10
	}

	if err := validateMonitor(
		input.Name,
		input.URL,
		input.HTTPMethod,
		input.IntervalSeconds,
		input.TimeoutSeconds,
	); err != nil {
		return nil, err
	}

	return s.repository.Create(ctx, input)
}

func (s *Service) GetByID(
	ctx context.Context,
	id int64,
) (*Monitor, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: invalid monitor ID", ErrInvalidInput)
	}

	return s.repository.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Monitor, error) {
	return s.repository.List(ctx)
}

func (s *Service) Update(
	ctx context.Context,
	id int64,
	input UpdateMonitorInput,
) (*Monitor, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: invalid monitor ID", ErrInvalidInput)
	}

	input.Name = strings.TrimSpace(input.Name)
	input.URL = strings.TrimSpace(input.URL)
	input.HTTPMethod = strings.ToUpper(strings.TrimSpace(input.HTTPMethod))

	if err := validateMonitor(
		input.Name,
		input.URL,
		input.HTTPMethod,
		input.IntervalSeconds,
		input.TimeoutSeconds,
	); err != nil {
		return nil, err
	}

	return s.repository.Update(ctx, id, input)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: invalid monitor ID", ErrInvalidInput)
	}

	return s.repository.Delete(ctx, id)
}

func validateMonitor(
	name string,
	rawURL string,
	method string,
	interval int,
	timeout int,
) error {
	if name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}

	if utf8.RuneCountInString(name) > 150 {
		return fmt.Errorf("%w: name must not exceed 150 characters", ErrInvalidInput)
	}

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("%w: invalid URL", ErrInvalidInput)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("%w: only HTTP and HTTPS URLs are supported", ErrInvalidInput)
	}

	if parsedURL.Hostname() == "" || parsedURL.User != nil {
		return fmt.Errorf("%w: invalid URL host or credentials", ErrInvalidInput)
	}

	if method != "GET" && method != "HEAD" {
		return fmt.Errorf("%w: HTTP method must be GET or HEAD", ErrInvalidInput)
	}

	if interval < 10 {
		return fmt.Errorf("%w: interval must be at least 10 seconds", ErrInvalidInput)
	}

	if timeout <= 0 {
		return fmt.Errorf("%w: timeout must be greater than zero", ErrInvalidInput)
	}

	return nil
}
