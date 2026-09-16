package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mkaric2003/multibook-backend/internal/modules/stay_search/domain"
)

var ErrValidation = fmt.Errorf("validation failed")

type Queries interface {
	Search(context.Context, domain.Input) ([]domain.Item, error)
}
type Service struct{ queries Queries }

func NewService(queries Queries) *Service { return &Service{queries: queries} }

func (s *Service) Search(ctx context.Context, input domain.Input) ([]domain.Item, error) {
	input.City = normalize(input.City)
	if input.Adults < 1 || input.Children < 0 {
		return nil, fmt.Errorf("%w: guest counts are invalid", ErrValidation)
	}
	if input.MinPriceMinor < 0 || input.MaxPriceMinor < 0 || input.MinPriceMinor > input.MaxPriceMinor {
		return nil, fmt.Errorf("%w: price range is invalid", ErrValidation)
	}
	if input.MinimumRating < 0 {
		return nil, fmt.Errorf("%w: minimum_rating is invalid", ErrValidation)
	}
	var checkIn, checkOut time.Time
	var err error
	if input.CheckIn != "" {
		checkIn, err = time.Parse(time.DateOnly, input.CheckIn)
		if err != nil {
			return nil, fmt.Errorf("%w: check_in must be an ISO date", ErrValidation)
		}
	}
	if input.CheckOut != "" {
		checkOut, err = time.Parse(time.DateOnly, input.CheckOut)
		if err != nil {
			return nil, fmt.Errorf("%w: check_out must be an ISO date", ErrValidation)
		}
	}
	if !checkIn.IsZero() && !checkOut.IsZero() && !checkOut.After(checkIn) {
		return nil, fmt.Errorf("%w: check_out must be after check_in", ErrValidation)
	}
	if input.InventoryType != "" && input.InventoryType != "singleUnit" && input.InventoryType != "multipleUnits" {
		return nil, fmt.Errorf("%w: inventory_type is invalid", ErrValidation)
	}
	if input.PageSize < 1 {
		input.PageSize = 8
	}
	if input.PageSize > 20 {
		input.PageSize = 20
	}
	if input.Offset < 0 {
		return nil, fmt.Errorf("%w: cursor is invalid", ErrValidation)
	}
	return s.queries.Search(ctx, input)
}

func normalize(v string) string { return strings.ToLower(strings.TrimSpace(v)) }
