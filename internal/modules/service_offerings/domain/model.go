package domain

import (
	"github.com/google/uuid"
	"time"
)

type Offering struct {
	ID              uuid.UUID `json:"id"`
	BusinessID      uuid.UUID `json:"business_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	DurationMinutes int16     `json:"duration_minutes"`
	PriceMinor      int64     `json:"price_minor"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
type CreateInput struct {
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	DurationMinutes int16   `json:"duration_minutes"`
	PriceMinor      int64   `json:"price_minor"`
	IsActive        *bool   `json:"is_active"`
}
type UpdateInput struct {
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	DurationMinutes *int16  `json:"duration_minutes"`
	PriceMinor      *int64  `json:"price_minor"`
	IsActive        *bool   `json:"is_active"`
}
