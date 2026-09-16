package domain

import (
	"time"

	"github.com/google/uuid"
)

type Item struct {
	ID                    uuid.UUID
	OwnerID               string
	Name                  string
	CategoryID            string
	City                  string
	Address               string
	Latitude              float64
	Longitude             float64
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
	LowestPriceMinor      int64
	Offerings             []Offering
	Providers             []Provider
}

type Offering struct {
	ID              uuid.UUID
	Name            string
	Description     *string
	DurationMinutes int16
	PriceMinor      int64
	IsActive        bool
}

type Provider struct {
	ID             uuid.UUID
	Name           string
	Title          *string
	CommissionRate float64
	IsActive       bool
	Availability   []AvailabilitySlot
}

type AvailabilitySlot struct {
	ID           uuid.UUID
	Weekday      int16
	StartMinutes int16
	EndMinutes   int16
}
type Input struct {
	City            string
	CategoryID      string
	CollectionID    string
	MinPriceMinor   int64
	MaxPriceMinor   int64
	AppointmentDate string
	StartMinutes    *int16
	SortOption      string
	Cursor          string
	PageSize        int
}
