package application

import (
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

// BusinessReadBase is shared data for business read-side query models. It is
// intentionally transport-agnostic: HTTP naming and JSON shape belong to the
// HTTP adapter.
type BusinessReadBase struct {
	ID                    uuid.UUID
	OwnerID               string
	Type                  domain.BusinessType
	Status                domain.BusinessStatus
	Name                  string
	CategoryID            string
	Location              LocationRead
	Currency              string
	ShortDescription      *string
	LogoPath              *string
	CoverPath             *string
	PhotoPaths            []string
	FeaturedCollectionIDs []string
	AverageRating         float64
	ReviewCount           int
	IsPromotionActive     bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type LocationRead struct {
	City      string
	Address   string
	Latitude  float64
	Longitude float64
}

type StaySummaryRead struct {
	PricePerNight *int64
	InventoryType string
}

type StayDetailRead struct {
	StaySummaryRead
	Amenities []string
	Extras    []StayExtraRead
	Rooms     []StayRoomRead
}

type StayExtraRead struct {
	Type        string
	Price       int64
	PricingUnit string
}

type StayRoomRead struct {
	ID               uuid.UUID
	Name             string
	MaxGuests        int16
	SizeSquareMeters int
	PricePerNight    int64
	Quantity         int16
	IsActive         bool
}

type ServiceDetailRead struct {
	Offerings []ServiceOfferingRead
	Providers []ServiceProviderRead
}

type ServiceOfferingRead struct {
	ID              uuid.UUID
	Name            string
	Description     *string
	DurationMinutes int16
	Price           int64
	IsActive        bool
}

type ServiceProviderRead struct {
	ID             uuid.UUID
	Name           string
	Title          *string
	CommissionRate float64
	IsActive       bool
	Availability   []WeeklyAvailabilityRead
}

type WeeklyAvailabilityRead struct {
	ID           uuid.UUID
	Weekday      int16
	StartMinutes int16
	EndMinutes   int16
}

// OwnedSummary is the provider's lightweight list model.
type OwnedSummary struct {
	BusinessReadBase
	Stay *StaySummaryRead
}

// OwnedDetail is the full provider edit model.
type OwnedDetail struct {
	BusinessReadBase
	Stay    *StayDetailRead
	Service *ServiceDetailRead
}
