// Package application contains Reviews use cases and ports.
package application

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/reviews/domain"
)

var (
	ErrNotFound          = errors.New("review source or business not found")
	ErrForbidden         = errors.New("forbidden")
	ErrValidation        = errors.New("validation failed")
	ErrAlreadyReviewed   = errors.New("business already reviewed")
	ErrSourceNotFinished = errors.New("review source is not finished")
)

const (
	defaultListLimit     = 20
	maximumListLimit     = 100
	maximumCommentLength = 1000
)

type Actor struct {
	ID   string
	Role string
}

type Page struct {
	Items      []domain.Review
	NextOffset *int
}

type Service struct {
	repository Repository
	queries    Queries
}

func NewService(repository Repository, queries Queries) *Service {
	return &Service{repository: repository, queries: queries}
}

func (s *Service) Create(ctx context.Context, actor Actor, businessID uuid.UUID, input domain.CreateInput) (domain.Review, error) {
	if err := authorizeCustomer(actor); err != nil {
		return domain.Review{}, err
	}
	if businessID == uuid.Nil || input.SourceID == uuid.Nil || !input.Type.Valid() || input.Rating < 1 || input.Rating > 5 {
		return domain.Review{}, ErrValidation
	}
	input.Comment = normalizeComment(input.Comment)
	return s.repository.Create(ctx, actor.ID, businessID, input)
}

func (s *Service) HasReview(ctx context.Context, actor Actor, businessID uuid.UUID) (bool, error) {
	if err := authorizeCustomer(actor); err != nil {
		return false, err
	}
	if businessID == uuid.Nil {
		return false, ErrValidation
	}
	return s.queries.HasReview(ctx, actor.ID, businessID)
}

func (s *Service) List(ctx context.Context, actor Actor, businessID uuid.UUID, limit, offset int) (Page, error) {
	if actor.ID == "" {
		return Page{}, ErrForbidden
	}
	if businessID == uuid.Nil || offset < 0 {
		return Page{}, ErrValidation
	}
	if limit < 1 {
		limit = defaultListLimit
	}
	if limit > maximumListLimit {
		limit = maximumListLimit
	}
	items, err := s.queries.List(ctx, businessID, limit+1, offset)
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		next := offset + limit
		page.NextOffset = &next
	}
	return page, nil
}

func authorizeCustomer(actor Actor) error {
	if actor.ID == "" || actor.Role != "customer" {
		return ErrForbidden
	}
	return nil
}

func normalizeComment(comment *string) *string {
	if comment == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*comment)
	if trimmed == "" {
		return nil
	}
	runes := []rune(trimmed)
	if len(runes) > maximumCommentLength {
		trimmed = string(runes[:maximumCommentLength])
	}
	return &trimmed
}
