package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

// Queries is the customer-facing business catalog read port.
type Queries interface {
	GetActiveDetail(context.Context, uuid.UUID) (CustomerBusinessDetail, error)
	ListPopular(context.Context, domain.BusinessType, string, int, int) ([]CustomerBusinessSummary, error)
	ListCities(context.Context) ([]string, error)
	ListFeaturedCollections(context.Context, domain.BusinessType) ([]FeaturedCollection, error)
	Search(context.Context, domain.BusinessType, string, int) ([]CustomerBusinessSummary, error)
	ListRecommendedStays(context.Context, string, int) ([]RecommendedStay, error)
}

// CustomerBusinessSummaryReader is the published read contract used by
// customer-owned relation modules such as Saved and Recently Viewed. Results
// preserve input ID order and omit businesses that are not active.
type CustomerBusinessSummaryReader interface {
	ListActiveByIDs(context.Context, []uuid.UUID) ([]CustomerBusinessSummary, error)
}
