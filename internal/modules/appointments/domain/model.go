// Package domain contains service appointment models.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusConfirmed Status = "confirmed"
	StatusDeclined  Status = "declined"
	StatusCancelled Status = "cancelled"
	StatusCompleted Status = "completed"
	StatusNoShow    Status = "no_show"
)

type Offering struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	DurationMinutes int16     `json:"duration_minutes"`
	PriceMinor      int64     `json:"price_minor"`
}

type Appointment struct {
	ID                       uuid.UUID  `json:"id"`
	BusinessID               uuid.UUID  `json:"business_id"`
	BusinessOwnerID          string     `json:"business_owner_id"`
	CustomerID               string     `json:"customer_id"`
	StaffID                  uuid.UUID  `json:"staff_id"`
	BusinessName             string     `json:"business_name"`
	BusinessImageURL         string     `json:"business_image_url"`
	CustomerName             string     `json:"customer_name"`
	CustomerEmail            string     `json:"customer_email"`
	CustomerPhone            string     `json:"customer_phone"`
	ProviderName             string     `json:"provider_name"`
	AppointmentDate          string     `json:"appointment_date"`
	StartMinutes             int16      `json:"start_minutes"`
	EndMinutes               int16      `json:"end_minutes"`
	ServiceCostMinor         int64      `json:"service_cost_minor"`
	OriginalServiceCostMinor int64      `json:"original_service_cost_minor"`
	DiscountMinor            int64      `json:"discount_minor"`
	ServiceFeeMinor          int64      `json:"service_fee_minor"`
	TaxesMinor               int64      `json:"taxes_minor"`
	TotalMinor               int64      `json:"total_minor"`
	ProviderCommissionRate   float64    `json:"provider_commission_rate"`
	ProviderEarningsMinor    int64      `json:"provider_earnings_minor"`
	Currency                 string     `json:"currency"`
	PaymentStatus            string     `json:"payment_status"`
	PaymentMethod            string     `json:"payment_method"`
	ConfirmationCode         string     `json:"confirmation_code"`
	Status                   Status     `json:"status"`
	CustomerRescheduleCount  int16      `json:"customer_reschedule_count"`
	Offerings                []Offering `json:"offerings"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

type CreateInput struct {
	StaffID         uuid.UUID   `json:"staff_id"`
	AppointmentDate string      `json:"appointment_date"`
	StartMinutes    int16       `json:"start_minutes"`
	OfferingIDs     []uuid.UUID `json:"offering_ids"`
	CustomerName    string      `json:"customer_name"`
	CustomerEmail   string      `json:"customer_email"`
	CustomerPhone   string      `json:"customer_phone"`
	PaymentMethod   string      `json:"payment_method"`
}

type AvailabilityInput struct {
	AppointmentDate string      `json:"appointment_date"`
	OfferingIDs     []uuid.UUID `json:"offering_ids"`
}

type AvailableSlots struct {
	StaffID         uuid.UUID `json:"staff_id"`
	AppointmentDate string    `json:"appointment_date"`
	DurationMinutes int16     `json:"duration_minutes"`
	StartMinutes    []int16   `json:"start_minutes"`
}

type RescheduleInput struct {
	AppointmentDate string `json:"appointment_date"`
	StartMinutes    int16  `json:"start_minutes"`
}

type ListInput struct {
	BusinessID uuid.UUID
	Status     string
	Offset     int
	Limit      int
}

type Page struct {
	Items      []Appointment
	NextCursor *string
}
