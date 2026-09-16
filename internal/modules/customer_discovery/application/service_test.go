package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

type queriesStub struct {
	businessType domain.BusinessType
	city         string
	query        string
	limit        int
	offset       int
}

func (*queriesStub) GetActiveDetail(context.Context, uuid.UUID) (CustomerBusinessDetail, error) {
	return CustomerBusinessDetail{}, nil
}
func (q *queriesStub) ListPopular(_ context.Context, businessType domain.BusinessType, city string, limit, offset int) ([]CustomerBusinessSummary, error) {
	q.businessType, q.city, q.limit, q.offset = businessType, city, limit, offset
	return nil, nil
}
func (*queriesStub) ListCities(context.Context) ([]string, error) { return nil, nil }
func (*queriesStub) ListFeaturedCollections(context.Context, domain.BusinessType) ([]FeaturedCollection, error) {
	return nil, nil
}
func (q *queriesStub) Search(_ context.Context, businessType domain.BusinessType, query string, limit int) ([]CustomerBusinessSummary, error) {
	q.businessType, q.query, q.limit = businessType, query, limit
	return nil, nil
}
func (q *queriesStub) ListRecommendedStays(_ context.Context, city string, limit int) ([]RecommendedStay, error) {
	q.city, q.limit = city, limit
	return nil, nil
}

func TestListPopularNormalizesInput(t *testing.T) {
	queries := &queriesStub{}
	_, err := NewService(queries).ListPopular(context.Background(), domain.BusinessTypeStay, "  Čapljina  ", 12, 4)
	if err != nil {
		t.Fatal(err)
	}
	if queries.businessType != domain.BusinessTypeStay || queries.city != "čapljina" || queries.limit != 12 || queries.offset != 4 {
		t.Fatalf("unexpected query input: %#v", queries)
	}
}

func TestDiscoveryValidationStopsInvalidQueries(t *testing.T) {
	service := NewService(&queriesStub{})
	if _, err := service.ListPopular(context.Background(), "invalid", "", 10, 0); !errors.Is(err, ErrValidation) {
		t.Fatalf("business type error = %v", err)
	}
	if _, err := service.ListPopular(context.Background(), domain.BusinessTypeStay, "", 31, 0); !errors.Is(err, ErrValidation) {
		t.Fatalf("pagination error = %v", err)
	}
	if _, err := service.Search(context.Background(), domain.BusinessTypeService, "  ", 20); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty search error = %v", err)
	}
}

func TestSearchAndRecommendationsNormalizeText(t *testing.T) {
	queries := &queriesStub{}
	service := NewService(queries)
	if _, err := service.Search(context.Background(), domain.BusinessTypeService, "  Žuti Salon  ", 8); err != nil {
		t.Fatal(err)
	}
	if queries.query != "žuti salon" || queries.limit != 8 {
		t.Fatalf("unexpected search input: %#v", queries)
	}
	if _, err := service.ListRecommendedStays(context.Background(), " Sarajevo "); err != nil {
		t.Fatal(err)
	}
	if queries.city != "sarajevo" || queries.limit != 3 {
		t.Fatalf("unexpected recommendation input: %#v", queries)
	}
}
