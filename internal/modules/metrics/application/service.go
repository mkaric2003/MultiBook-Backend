// Package application contains provider Metrics read use cases.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrBusinessNotFound = errors.New("business not found")
	ErrStaffNotFound    = errors.New("staff not found")
	ErrForbidden        = errors.New("forbidden")
	ErrValidation       = errors.New("validation failed")
)

type EarningsFilter struct {
	StartDate        time.Time
	EndDate          time.Time
	UTCOffsetMinutes int
	StaffID          *uuid.UUID
}

type Actor struct {
	ID   string
	Role string
}

type Service struct {
	queries Queries
	updates MetricsUpdates
	now     func() time.Time
}

func NewService(queries Queries, updates MetricsUpdates) *Service {
	return &Service{queries: queries, updates: updates, now: time.Now}
}

func (s *Service) GetDashboardMetrics(ctx context.Context, actor Actor, businessID uuid.UUID) (DashboardMetrics, error) {
	monthStart, err := s.validateDashboardRequest(actor, businessID)
	if err != nil {
		return DashboardMetrics{}, err
	}
	return s.queries.GetDashboardMetrics(ctx, actor.ID, businessID, monthStart)
}

// OpenDashboardMetricsStream subscribes before loading the initial metrics so a
// reservation commit cannot be lost between those two operations.
func (s *Service) OpenDashboardMetricsStream(ctx context.Context, actor Actor, businessID uuid.UUID) (DashboardMetrics, <-chan struct{}, func(), error) {
	monthStart, err := s.validateDashboardRequest(actor, businessID)
	if err != nil {
		return DashboardMetrics{}, nil, nil, err
	}
	updates, unsubscribe := s.updates.SubscribeChanges(businessID)
	metrics, err := s.queries.GetDashboardMetrics(ctx, actor.ID, businessID, monthStart)
	if err != nil {
		unsubscribe()
		return DashboardMetrics{}, nil, nil, err
	}
	return metrics, updates, unsubscribe, nil
}

func (s *Service) GetEarningsMetrics(ctx context.Context, actor Actor, businessID uuid.UUID, filter EarningsFilter) (EarningsMetrics, error) {
	if err := validateEarningsRequest(actor, businessID, filter); err != nil {
		return EarningsMetrics{}, err
	}
	return s.queries.GetEarningsMetrics(ctx, actor.ID, businessID, filter)
}

func (s *Service) OpenEarningsMetricsStream(ctx context.Context, actor Actor, businessID uuid.UUID, filter EarningsFilter) (EarningsMetrics, <-chan struct{}, func(), error) {
	if err := validateEarningsRequest(actor, businessID, filter); err != nil {
		return EarningsMetrics{}, nil, nil, err
	}
	updates, unsubscribe := s.updates.SubscribeChanges(businessID)
	metrics, err := s.queries.GetEarningsMetrics(ctx, actor.ID, businessID, filter)
	if err != nil {
		unsubscribe()
		return EarningsMetrics{}, nil, nil, err
	}
	return metrics, updates, unsubscribe, nil
}

func (s *Service) validateDashboardRequest(actor Actor, businessID uuid.UUID) (time.Time, error) {
	if err := authorizeProvider(actor); err != nil {
		return time.Time{}, err
	}
	if businessID == uuid.Nil {
		return time.Time{}, ErrValidation
	}
	now := s.now().UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func authorizeProvider(actor Actor) error {
	if actor.ID == "" || actor.Role != "provider" {
		return ErrForbidden
	}
	return nil
}

func validateEarningsRequest(actor Actor, businessID uuid.UUID, filter EarningsFilter) error {
	if err := authorizeProvider(actor); err != nil {
		return err
	}
	if businessID == uuid.Nil || filter.StartDate.IsZero() || filter.EndDate.IsZero() || filter.EndDate.Before(filter.StartDate) {
		return ErrValidation
	}
	if filter.StaffID != nil && *filter.StaffID == uuid.Nil {
		return ErrValidation
	}
	return nil
}
