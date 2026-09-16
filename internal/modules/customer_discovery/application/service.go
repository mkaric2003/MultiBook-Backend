// Package application contains customer discovery queries and validation.
package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

var (
	ErrBusinessNotFound = errors.New("business not found")
	ErrValidation       = errors.New("validation failed")
)

type Service struct{ queries Queries }

func NewService(queries Queries) *Service { return &Service{queries: queries} }

func (s *Service) GetActiveDetail(ctx context.Context, id uuid.UUID) (CustomerBusinessDetail, error) {
	return s.queries.GetActiveDetail(ctx, id)
}

func (s *Service) ListPopular(ctx context.Context, businessType domain.BusinessType, city string, limit, offset int) ([]CustomerBusinessSummary, error) {
	if err := validateBusinessType(businessType); err != nil {
		return nil, err
	}
	city = domain.NormalizeSearchText(city)
	if limit <= 0 || limit > 30 || offset < 0 {
		return nil, fmt.Errorf("%w: invalid pagination", ErrValidation)
	}
	return s.queries.ListPopular(ctx, businessType, city, limit, offset)
}

func (s *Service) ListCities(ctx context.Context) ([]string, error) {
	return s.queries.ListCities(ctx)
}

func (s *Service) ListFeaturedCollections(ctx context.Context, businessType domain.BusinessType) ([]FeaturedCollection, error) {
	if err := validateBusinessType(businessType); err != nil {
		return nil, err
	}
	return s.queries.ListFeaturedCollections(ctx, businessType)
}

func (s *Service) Search(ctx context.Context, businessType domain.BusinessType, query string, limit int) ([]CustomerBusinessSummary, error) {
	if err := validateBusinessType(businessType); err != nil {
		return nil, err
	}
	query = domain.NormalizeSearchText(query)
	if query == "" {
		return nil, fmt.Errorf("%w: query is required", ErrValidation)
	}
	if limit <= 0 || limit > 30 {
		return nil, fmt.Errorf("%w: invalid pagination", ErrValidation)
	}
	return s.queries.Search(ctx, businessType, query, limit)
}

func (s *Service) ListRecommendedStays(ctx context.Context, city string) ([]RecommendedStay, error) {
	return s.queries.ListRecommendedStays(ctx, domain.NormalizeSearchText(city), 3)
}

func validateBusinessType(value domain.BusinessType) error {
	if value != domain.BusinessTypeStay && value != domain.BusinessTypeService {
		return fmt.Errorf("%w: type must be stays or services", ErrValidation)
	}
	return nil
}
