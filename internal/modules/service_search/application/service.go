package application

import (
	"context"
	"fmt"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_search/domain"
	"sort"
	"strings"
	"time"
)

var ErrValidation = fmt.Errorf("validation failed")

type ServiceSearchQueries interface {
	Search(context.Context, domain.Input) ([]domain.Item, error)
}

type Page struct {
	Items      []domain.Item
	NextCursor *string
}

type Service struct{ queries ServiceSearchQueries }

func NewService(queries ServiceSearchQueries) *Service { return &Service{queries: queries} }
func (s *Service) Search(ctx context.Context, input domain.Input) (Page, error) {
	input.City = strings.TrimSpace(input.City)
	input.City = normalizeCity(input.City)
	input.CategoryID = strings.TrimSpace(input.CategoryID)
	input.CollectionID = strings.TrimSpace(input.CollectionID)
	if input.MinPriceMinor < 0 || input.MaxPriceMinor < 0 || input.MinPriceMinor > input.MaxPriceMinor {
		return Page{}, fmt.Errorf("%w: price range is invalid", ErrValidation)
	}
	if input.AppointmentDate != "" {
		if _, err := time.Parse(time.DateOnly, input.AppointmentDate); err != nil {
			return Page{}, fmt.Errorf("%w: appointment_date must be an ISO date", ErrValidation)
		}
		if input.StartMinutes == nil {
			return Page{}, fmt.Errorf("%w: start_minutes is required with appointment_date", ErrValidation)
		}
	}
	if input.StartMinutes != nil && (*input.StartMinutes < 0 || *input.StartMinutes >= 1440 || *input.StartMinutes%30 != 0) {
		return Page{}, fmt.Errorf("%w: start_minutes is invalid", ErrValidation)
	}
	if input.SortOption == "" {
		input.SortOption = "recommended"
	}
	if input.SortOption != "recommended" && input.SortOption != "priceLowToHigh" && input.SortOption != "priceHighToLow" && input.SortOption != "rating" {
		return Page{}, fmt.Errorf("%w: sort_option is invalid", ErrValidation)
	}
	if input.PageSize < 1 {
		input.PageSize = 8
	}
	if input.PageSize > 20 {
		input.PageSize = 20
	}
	items, err := s.queries.Search(ctx, input)
	if err != nil {
		return Page{}, err
	}
	sortItems(items, input.SortOption)
	start := cursorIndex(items, input.Cursor)
	end := min(start+input.PageSize, len(items))
	page := Page{Items: items[start:end]}
	if end < len(items) {
		cursor := items[end-1].ID.String()
		page.NextCursor = &cursor
	}
	return page, nil
}

func normalizeCity(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer("č", "c", "ć", "c", "š", "s", "ž", "z", "đ", "d").Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func sortItems(items []domain.Item, option string) {
	sort.Slice(items, func(i, j int) bool {
		first, second := items[i], items[j]
		if option == "priceLowToHigh" && first.LowestPriceMinor != second.LowestPriceMinor {
			return first.LowestPriceMinor < second.LowestPriceMinor
		}
		if option == "priceHighToLow" && first.LowestPriceMinor != second.LowestPriceMinor {
			return first.LowestPriceMinor > second.LowestPriceMinor
		}
		if option == "recommended" && first.AverageRating != second.AverageRating {
			return first.AverageRating > second.AverageRating
		}
		if option == "recommended" && first.ReviewCount != second.ReviewCount {
			return first.ReviewCount > second.ReviewCount
		}
		if option == "rating" && first.AverageRating != second.AverageRating {
			return first.AverageRating > second.AverageRating
		}
		if first.Name != second.Name {
			return first.Name < second.Name
		}
		return first.ID.String() < second.ID.String()
	})
}

func cursorIndex(items []domain.Item, cursor string) int {
	if cursor == "" {
		return 0
	}
	for i, item := range items {
		if item.ID.String() == cursor {
			return i + 1
		}
	}
	return 0
}
