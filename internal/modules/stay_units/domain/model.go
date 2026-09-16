// Package domain contains stay unit type models. A unit type describes a
// provider-visible room configuration; its quantity is materialized as units.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type UnitType struct {
	ID                 uuid.UUID `json:"id"`
	BusinessID         uuid.UUID `json:"business_id"`
	Name               string    `json:"name"`
	MaxGuests          int16     `json:"max_guests"`
	SizeSquareMeters   int       `json:"size_square_meters"`
	PricePerNightMinor int64     `json:"price_per_night_minor"`
	Quantity           int16     `json:"quantity"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CreateInput struct {
	Name               string `json:"name"`
	MaxGuests          int16  `json:"max_guests"`
	SizeSquareMeters   int    `json:"size_square_meters"`
	PricePerNightMinor int64  `json:"price_per_night_minor"`
	Quantity           int16  `json:"quantity"`
	IsActive           *bool  `json:"is_active"`
}

type UpdateInput struct {
	Name               *string `json:"name"`
	MaxGuests          *int16  `json:"max_guests"`
	SizeSquareMeters   *int    `json:"size_square_meters"`
	PricePerNightMinor *int64  `json:"price_per_night_minor"`
	Quantity           *int16  `json:"quantity"`
	IsActive           *bool   `json:"is_active"`
}
