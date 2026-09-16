package domain

import (
	"time"

	"github.com/google/uuid"
)

type Details struct {
	BusinessID uuid.UUID `json:"business_id"`
	TimeZone   string    `json:"time_zone"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type UpsertInput struct {
	TimeZone string `json:"time_zone"`
}
