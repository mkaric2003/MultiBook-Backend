// Package domain contains customer-owned saved booking and appointment drafts.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type BookingDraft struct {
	BusinessID       uuid.UUID       `json:"business_id"`
	BusinessName     string          `json:"business_name"`
	BusinessLocation string          `json:"business_location"`
	BusinessImageURL string          `json:"business_image_url"`
	PricePerNight    int64           `json:"price_per_night"`
	CheckIn          string          `json:"check_in"`
	CheckOut         string          `json:"check_out"`
	Adults           int16           `json:"adults"`
	Children         int16           `json:"children"`
	Infants          int16           `json:"infants"`
	RoomTypeID       *uuid.UUID      `json:"room_type_id,omitempty"`
	SelectedExtras   json.RawMessage `json:"selected_extras"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type AppointmentDraft struct {
	BusinessID           uuid.UUID       `json:"business_id"`
	BusinessName         string          `json:"business_name"`
	BusinessImageURL     string          `json:"business_image_url"`
	SelectedOfferingIDs  []uuid.UUID     `json:"selected_offering_ids"`
	SelectedProviderID   *uuid.UUID      `json:"selected_provider_id,omitempty"`
	SelectedProviderName *string         `json:"selected_provider_name,omitempty"`
	AppointmentDate      string          `json:"appointment_date"`
	StartMinutes         *int16          `json:"start_minutes,omitempty"`
	SelectedAddOnIDs     json.RawMessage `json:"selected_add_on_ids"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type BookingInput struct {
	BusinessID     uuid.UUID
	CheckIn        string
	CheckOut       string
	Adults         int16
	Children       int16
	Infants        int16
	RoomTypeID     *uuid.UUID
	SelectedExtras json.RawMessage
}

type AppointmentInput struct {
	BusinessID          uuid.UUID
	SelectedOfferingIDs []uuid.UUID
	SelectedProviderID  *uuid.UUID
	AppointmentDate     string
	StartMinutes        *int16
	SelectedAddOnIDs    json.RawMessage
}
