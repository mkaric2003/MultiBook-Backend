package domain

import "github.com/google/uuid"

type MediaType string

const (
	MediaTypeLogo    MediaType = "logo"
	MediaTypeCover   MediaType = "cover"
	MediaTypeGallery MediaType = "gallery"
)

type Media struct {
	ID          uuid.UUID `json:"id"`
	MediaType   MediaType `json:"media_type"`
	StoragePath string    `json:"storage_path"`
	Position    *int16    `json:"position"`
}
type ReplaceInput struct {
	Items []Input `json:"items"`
}
type Input struct {
	MediaType   MediaType `json:"media_type"`
	StoragePath string    `json:"storage_path"`
	Position    *int16    `json:"position"`
}
