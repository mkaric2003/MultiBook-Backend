package domain

import "github.com/google/uuid"

type Location struct {
	ID          uuid.UUID `json:"id"`
	City        string    `json:"city"`
	Address     string    `json:"address"`
	CountryCode *string   `json:"country_code"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
}

type UpsertInput struct {
	City        string  `json:"city"`
	Address     string  `json:"address"`
	CountryCode *string `json:"country_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}
