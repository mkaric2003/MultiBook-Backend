// Package domain contains support ticket models without transport or persistence dependencies.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type Category string

const (
	CategoryAccount     Category = "account"
	CategoryBooking     Category = "booking"
	CategoryAppointment Category = "appointment"
	CategoryPayment     Category = "payment"
	CategoryTechnical   Category = "technical"
	CategoryOther       Category = "other"
)

func (category Category) Valid() bool {
	switch category {
	case CategoryAccount, CategoryBooking, CategoryAppointment, CategoryPayment, CategoryTechnical, CategoryOther:
		return true
	default:
		return false
	}
}

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "inProgress"
	StatusResolved   Status = "resolved"
)

type Ticket struct {
	ID            uuid.UUID
	CustomerID    string
	CustomerName  string
	CustomerEmail string
	Category      Category
	Status        Status
	Subject       string
	Message       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreateInput struct {
	Category Category
	Subject  string
	Message  string
}
