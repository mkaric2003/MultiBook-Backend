// Package application contains Saved businesses use cases and ports.
package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	discoveryapp "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/application"
)

var (
	ErrBusinessNotFound = errors.New("business not found")
	ErrForbidden        = errors.New("forbidden")
	ErrValidation       = errors.New("validation failed")
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

func (s *Service) Save(ctx context.Context, actor Actor, businessID uuid.UUID) error {
	if err := authorize(actor, businessID); err != nil {
		return err
	}
	return s.repository.Save(ctx, actor.ID, businessID)
}

func (s *Service) Remove(ctx context.Context, actor Actor, businessID uuid.UUID) error {
	if err := authorize(actor, businessID); err != nil {
		return err
	}
	return s.repository.Remove(ctx, actor.ID, businessID)
}

func (s *Service) IsSaved(ctx context.Context, actor Actor, businessID uuid.UUID) (bool, error) {
	if err := authorize(actor, businessID); err != nil {
		return false, err
	}
	return s.queries.IsSaved(ctx, actor.ID, businessID)
}

func (s *Service) List(ctx context.Context, actor Actor) ([]discoveryapp.CustomerBusinessSummary, error) {
	if err := authorizeCustomer(actor); err != nil {
		return nil, err
	}
	ids, err := s.queries.ListBusinessIDs(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	return s.summaries.ListActiveByIDs(ctx, ids)
}

func authorize(actor Actor, businessID uuid.UUID) error {
	if err := authorizeCustomer(actor); err != nil {
		return err
	}
	if businessID == uuid.Nil {
		return ErrValidation
	}
	return nil
}

func authorizeCustomer(actor Actor) error {
	if actor.ID == "" || actor.Role != "customer" {
		return ErrForbidden
	}
	return nil
}
