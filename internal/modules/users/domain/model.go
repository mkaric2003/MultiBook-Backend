// Package domain contains the MultiBook users domain model. It has no HTTP,
// Firebase, or database dependencies.
package domain

import "time"

type UserRole string

const (
	UserRoleCustomer UserRole = "customer"
	UserRoleProvider UserRole = "provider"
	UserRoleAdmin    UserRole = "admin"
)

type User struct {
	ID                 string    `json:"id"`
	Email              *string   `json:"email"`
	FullName           *string   `json:"full_name"`
	FirstName          *string   `json:"first_name"`
	LastName           *string   `json:"last_name"`
	Role               *UserRole `json:"role"`
	PhoneE164          *string   `json:"phone_e164"`
	AvatarStoragePath  *string   `json:"avatar_storage_path"`
	CountryCode        *string   `json:"country_code"`
	DateOfBirth        *string   `json:"date_of_birth"`
	Address            *string   `json:"address"`
	City               *string   `json:"city"`
	BusinessCurrency   string    `json:"business_currency"`
	SelectedBusinessID *string   `json:"selected_business_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type UpdateProfileInput struct {
	FirstName         *string `json:"first_name"`
	LastName          *string `json:"last_name"`
	PhoneE164         *string `json:"phone_e164"`
	AvatarStoragePath *string `json:"avatar_storage_path"`
	CountryCode       *string `json:"country_code"`
	DateOfBirth       *string `json:"date_of_birth"`
	Address           *string `json:"address"`
	City              *string `json:"city"`
	BusinessCurrency  *string `json:"business_currency"`
}
