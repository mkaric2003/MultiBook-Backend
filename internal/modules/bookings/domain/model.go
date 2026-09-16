package domain

import (
	"time"

	"github.com/google/uuid"
)

type ExtraSelection struct {
	Type string `json:"type"`
}

type CreateInput struct {
	StayUnitTypeID *uuid.UUID       `json:"stayUnitTypeId"`
	CheckIn        string           `json:"checkIn"`
	CheckOut       string           `json:"checkOut"`
	Adults         int16            `json:"adults"`
	Children       int16            `json:"children"`
	Infants        int16            `json:"infants"`
	SelectedExtras []ExtraSelection `json:"selectedExtras"`
	CustomerName   string           `json:"customerName"`
	CustomerEmail  string           `json:"customerEmail"`
	PaymentMethod  string           `json:"paymentMethod"`
}

type Booking struct {
	ID               uuid.UUID  `json:"id"`
	BusinessID       uuid.UUID  `json:"businessId"`
	BusinessOwnerID  string     `json:"businessOwnerId"`
	CustomerID       string     `json:"customerId"`
	CustomerName     string     `json:"customerName"`
	CustomerEmail    string     `json:"customerEmail"`
	BusinessName     string     `json:"businessName"`
	BusinessCity     string     `json:"businessCity"`
	BusinessImageURL string     `json:"businessImageUrl"`
	RoomType         *string    `json:"roomType"`
	RoomTypeID       *uuid.UUID `json:"roomTypeId"`
	CheckIn          string     `json:"checkIn"`
	CheckOut         string     `json:"checkOut"`
	Adults           int16      `json:"adults"`
	Children         int16      `json:"children"`
	Infants          int16      `json:"infants"`
	PricePerNight    int64      `json:"pricePerNight"`
	SelectedExtras   any        `json:"selectedExtras"`
	RoomSubtotal     int64      `json:"roomSubtotal"`
	DiscountAmount   int64      `json:"discountAmount"`
	CleaningFee      int64      `json:"cleaningFee"`
	ServiceFee       int64      `json:"serviceFee"`
	Taxes            int64      `json:"taxes"`
	Total            int64      `json:"total"`
	OriginalTotal    int64      `json:"originalTotal"`
	PaymentStatus    string     `json:"paymentStatus"`
	PaymentMethod    string     `json:"paymentMethod"`
	ConfirmationCode string     `json:"confirmationCode"`
	Currency         string     `json:"currency"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type ListInput struct {
	BusinessID *uuid.UUID
	Status     string
	Offset     int
	Limit      int
}

type Page struct {
	Items      []Booking
	NextCursor *string
}

type UnavailableRange struct {
	CheckIn  string `json:"checkIn"`
	CheckOut string `json:"checkOut"`
}
