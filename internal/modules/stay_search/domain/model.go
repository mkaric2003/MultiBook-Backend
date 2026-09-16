package domain

import "github.com/google/uuid"

type Input struct {
	City, CheckIn, CheckOut, InventoryType string
	Adults, Children                       int
	MinPriceMinor, MaxPriceMinor           int64
	MinimumRating                          float64
	CategoryIDs, CollectionIDs, Amenities  []string
	PageSize, Offset                       int
}

type Item struct {
	ID                                          uuid.UUID
	OwnerID, Name, CategoryID, Currency         string
	City, Address                               *string
	Latitude, Longitude, AverageRating          float64
	ReviewCount                                 int
	ShortDescription, LogoURL, CoverPhotoURL    *string
	PhotoURLs, FeaturedCollectionIDs, Amenities []string
	PricePerNight                               *int64
	InventoryType                               string
}
