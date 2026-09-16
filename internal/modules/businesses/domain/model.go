// Package domain contains the MultiBook businesses domain model. It is free
// from HTTP, Firebase, and PostgreSQL dependencies.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type BusinessType string

const (
	BusinessTypeStay    BusinessType = "stay"
	BusinessTypeService BusinessType = "service"
)

type BusinessStatus string

const (
	BusinessStatusDraft    BusinessStatus = "draft"
	BusinessStatusActive   BusinessStatus = "active"
	BusinessStatusInactive BusinessStatus = "inactive"
	BusinessStatusArchived BusinessStatus = "archived"
)

type Business struct {
	ID                uuid.UUID      `json:"id"`
	OwnerID           string         `json:"owner_id"`
	Type              BusinessType   `json:"type"`
	Status            BusinessStatus `json:"status"`
	Name              string         `json:"name"`
	CategoryID        string         `json:"category_id"`
	Currency          string         `json:"currency"`
	ShortDescription  *string        `json:"short_description"`
	AverageRating     float64        `json:"average_rating"`
	ReviewCount       int            `json:"review_count"`
	IsPromotionActive bool           `json:"is_promotion_active"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type CreateInput struct {
	Type                  BusinessType `json:"type"`
	Name                  string       `json:"name"`
	CategoryID            string       `json:"category_id"`
	ShortDescription      *string      `json:"short_description"`
	Location              Location     `json:"location"`
	Media                 []Media      `json:"media"`
	FeaturedCollectionIDs []string     `json:"featured_collection_ids"`
	Stay                  *Stay        `json:"stay"`
	Service               *Service     `json:"service"`
}

type Location struct {
	City        string  `json:"city"`
	Address     string  `json:"address"`
	CountryCode *string `json:"country_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}
type Media struct {
	MediaType   string `json:"media_type"`
	StoragePath string `json:"storage_path"`
	Position    *int16 `json:"position"`
}
type Stay struct {
	InventoryType  string     `json:"inventory_type"`
	BasePriceMinor *int64     `json:"base_price_minor"`
	Amenities      []string   `json:"amenities"`
	UnitTypes      []UnitType `json:"unit_types"`
	Extras         []Extra    `json:"extras"`
}
type UnitType struct {
	ID                 uuid.UUID `json:"-"`
	ClientID           string    `json:"-"`
	Name               string    `json:"name"`
	MaxGuests          int16     `json:"max_guests"`
	SizeSquareMeters   int       `json:"size_square_meters"`
	PricePerNightMinor int64     `json:"price_per_night_minor"`
	Quantity           int16     `json:"quantity"`
	IsActive           *bool     `json:"is_active"`
}
type Extra struct {
	Type        string `json:"type"`
	PriceMinor  int64  `json:"price_minor"`
	PricingUnit string `json:"pricing_unit"`
}
type Service struct {
	TimeZone  string     `json:"time_zone"`
	Offerings []Offering `json:"offerings"`
	Staff     []Staff    `json:"staff"`
}
type Offering struct {
	ID              uuid.UUID `json:"-"`
	ClientID        string    `json:"client_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	DurationMinutes int16     `json:"duration_minutes"`
	PriceMinor      int64     `json:"price_minor"`
	IsActive        *bool     `json:"is_active"`
}
type Staff struct {
	ID                 uuid.UUID            `json:"-"`
	ClientID           string               `json:"-"`
	Name               string               `json:"name"`
	Title              *string              `json:"title"`
	CommissionRate     *float64             `json:"commission_rate"`
	IsActive           *bool                `json:"is_active"`
	OfferingClientIDs  []string             `json:"offering_client_ids"`
	WeeklyAvailability []WeeklyAvailability `json:"weekly_availability"`
}
type WeeklyAvailability struct {
	ID           uuid.UUID `json:"-"`
	ClientID     string    `json:"-"`
	Weekday      int16     `json:"weekday"`
	StartMinutes int16     `json:"start_minutes"`
	EndMinutes   int16     `json:"end_minutes"`
}

type UpdateInput struct {
	Name             *string         `json:"name"`
	CategoryID       *string         `json:"category_id"`
	ShortDescription *string         `json:"short_description"`
	Status           *BusinessStatus `json:"status"`
}
