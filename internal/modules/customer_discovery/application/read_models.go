package application

import (
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

// CustomerBusinessBase contains the fields shared by customer-facing business
// summaries and details. Transport naming remains an HTTP adapter concern.
type CustomerBusinessBase struct {
	ID                    uuid.UUID
	OwnerID               string
	Type                  domain.BusinessType
	Status                domain.BusinessStatus
	Name                  string
	CategoryID            string
	Location              LocationSummary
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

type LocationSummary struct {
	City      string
	Address   string
	Latitude  float64
	Longitude float64
}

type StaySummary struct {
	PricePerNight *int64
	InventoryType string
}

type StayDetail struct {
	StaySummary
	Amenities []string
	Extras    []StayExtra
	Rooms     []StayRoom
}

type StayExtra struct {
	Type        string
	Price       int64
	PricingUnit string
}

type StayRoom struct {
	ID               uuid.UUID
	Name             string
	MaxGuests        int16
	SizeSquareMeters int
	PricePerNight    int64
	Quantity         int16
	IsActive         bool
}

type ServiceSummary struct {
	Offerings []ServiceOffering
}

type ServiceDetail struct {
	Offerings []ServiceOffering
	Providers []ServiceProvider
}

type ServiceOffering struct {
	ID              uuid.UUID
	Name            string
	Description     *string
	DurationMinutes int16
	Price           int64
	IsActive        bool
}

type ServiceProvider struct {
	ID             uuid.UUID
	Name           string
	Title          *string
	CommissionRate float64
	IsActive       bool
	Availability   []WeeklyAvailability
}

type WeeklyAvailability struct {
	ID           uuid.UUID
	Weekday      int16
	StartMinutes int16
	EndMinutes   int16
}

// CustomerBusinessSummary is the lightweight list projection used by
// discovery feeds. It is not a persisted snapshot or a UI widget model.
type CustomerBusinessSummary struct {
	CustomerBusinessBase
	Stay    *StaySummary
	Service *ServiceSummary
}

// CustomerBusinessDetail is the complete active customer-facing projection.
type CustomerBusinessDetail struct {
	CustomerBusinessBase
	Stay    *StayDetail
	Service *ServiceDetail
}

type FeaturedCollection struct {
	ID          string
	TitleKey    string
	SubtitleKey string
	ImageURL    string
}

type RecommendedStay struct {
	CustomerBusinessBase
	Stay StaySummary
}
