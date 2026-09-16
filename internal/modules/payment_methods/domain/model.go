// Package domain contains saved payment method metadata models.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type CardBrand string

const (
	CardBrandVisa       CardBrand = "visa"
	CardBrandMastercard CardBrand = "mastercard"
	CardBrandAmex       CardBrand = "amex"
	CardBrandOther      CardBrand = "other"
)

func (brand CardBrand) Valid() bool {
	return brand == CardBrandVisa || brand == CardBrandMastercard || brand == CardBrandAmex || brand == CardBrandOther
}

type PaymentMethod struct {
	ID          uuid.UUID
	CustomerID  string
	Brand       CardBrand
	Last4       string
	ExpiryMonth int
	ExpiryYear  int
	HolderName  string
	IsDefault   bool
	CreatedAt   time.Time
}

type CreateInput struct {
	Brand       CardBrand
	Last4       string
	ExpiryMonth int
	ExpiryYear  int
	HolderName  string
}
