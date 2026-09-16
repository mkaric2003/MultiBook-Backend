// Package application contains Recently Viewed use cases and ports.
package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	discoveryapp "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/application"
)

var (
	ErrBusinessNotFound = errors.New("business not found")
	ErrForbidden        = errors.New("forbidden")
	ErrValidation       = errors.New("validation failed")
)

const (
	defaultListLimit = 10
	maximumListLimit = 30
)

type Actor struct {
	ID   string
	Role string
}

type Service struct {
	repository Repository
	queries    Queries
	summaries  discoveryapp.CustomerBusinessSummaryReader
}

func NewService(repository Repository, queries Queries, summaries discoveryapp.CustomerBusinessSummaryReader) *Service {
	return &Service{repository: repository, queries: queries, summaries: summaries}
}

func (s *Service) Record(ctx context.Context, actor Actor, businessID uuid.UUID) error {
	if err := authorizeCustomer(actor); err != nil {
		return err
	}
	if businessID == uuid.Nil {
		return ErrValidation
	}
	return s.repository.Record(ctx, actor.ID, businessID)
}

func (s *Service) List(ctx context.Context, actor Actor, businessType domain.BusinessType, limit int) ([]discoveryapp.CustomerBusinessSummary, error) {
	if err := authorizeCustomer(actor); err != nil {
		return nil, err
	}
	if businessType != domain.BusinessTypeStay && businessType != domain.BusinessTypeService {
		return nil, ErrValidation
	}
	if limit < 1 {
		limit = defaultListLimit
	}
	if limit > maximumListLimit {
		limit = maximumListLimit
	}
	ids, err := s.queries.ListBusinessIDs(ctx, actor.ID, businessType, limit)
	if err != nil {
		return nil, err
	}
	return s.summaries.ListActiveByIDs(ctx, ids)
}

func authorizeCustomer(actor Actor) error {
	if actor.ID == "" || actor.Role != "customer" {
		return ErrForbidden
	}
	return nil
}
