package domain

import (
	"github.com/google/uuid"
	"time"
)

type Staff struct {
	ID             uuid.UUID   `json:"id"`
	BusinessID     uuid.UUID   `json:"business_id"`
	Name           string      `json:"name"`
	Title          *string     `json:"title"`
	CommissionRate float64     `json:"commission_rate"`
	IsActive       bool        `json:"is_active"`
	OfferingIDs    []uuid.UUID `json:"offering_ids"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}
type CreateInput struct {
	Name           string      `json:"name"`
	Title          *string     `json:"title"`
	CommissionRate *float64    `json:"commission_rate"`
	IsActive       *bool       `json:"is_active"`
	OfferingIDs    []uuid.UUID `json:"offering_ids"`
}
type UpdateInput struct {
	Name           *string      `json:"name"`
	Title          *string      `json:"title"`
	CommissionRate *float64     `json:"commission_rate"`
	IsActive       *bool        `json:"is_active"`
	OfferingIDs    *[]uuid.UUID `json:"offering_ids"`
}
