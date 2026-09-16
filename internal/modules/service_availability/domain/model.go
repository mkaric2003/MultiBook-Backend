// Package domain contains the service staff availability model.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type WeeklyAvailability struct {
	ID           uuid.UUID `json:"id"`
	StaffID      uuid.UUID `json:"staff_id"`
	Weekday      int16     `json:"weekday"`
	StartMinutes int16     `json:"start_minutes"`
	EndMinutes   int16     `json:"end_minutes"`
	CreatedAt    time.Time `json:"created_at"`
}

type AvailabilityBlock struct {
	ID        uuid.UUID `json:"id"`
	StaffID   uuid.UUID `json:"staff_id"`
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	Reason    *string   `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateWeeklyInput struct {
	Weekday      int16 `json:"weekday"`
	StartMinutes int16 `json:"start_minutes"`
	EndMinutes   int16 `json:"end_minutes"`
}

type UpdateWeeklyInput struct {
	Weekday      *int16 `json:"weekday"`
	StartMinutes *int16 `json:"start_minutes"`
	EndMinutes   *int16 `json:"end_minutes"`
}

type CreateBlockInput struct {
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
	Reason  *string   `json:"reason"`
}

type UpdateBlockInput struct {
	StartAt *time.Time `json:"start_at"`
	EndAt   *time.Time `json:"end_at"`
	Reason  *string    `json:"reason"`
}
