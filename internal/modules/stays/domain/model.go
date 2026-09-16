// Package domain contains the stays domain model.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type InventoryType string

const (
	InventoryTypeSingleUnit    InventoryType = "single_unit"
	InventoryTypeMultipleUnits InventoryType = "multiple_units"
)

type Details struct {
	BusinessID     uuid.UUID     `json:"business_id"`
	InventoryType  InventoryType `json:"inventory_type"`
	BasePriceMinor *int64        `json:"base_price_minor"`
	Amenities      []string      `json:"amenities"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type UpsertInput struct {
	InventoryType  InventoryType `json:"inventory_type"`
	BasePriceMinor *int64        `json:"base_price_minor"`
	Amenities      []string      `json:"amenities"`
}
