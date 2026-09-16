// Package domain contains the Reviews domain vocabulary.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type SourceType string

const (
	SourceTypeStay    SourceType = "stay"
	SourceTypeService SourceType = "service"
)

func (t SourceType) Valid() bool { return t == SourceTypeStay || t == SourceTypeService }

type CreateInput struct {
	SourceID uuid.UUID
	Type     SourceType
	Rating   int16
	Comment  *string
}

type Review struct {
	ID                 uuid.UUID
	BusinessID         uuid.UUID
	BusinessOwnerID    string
	CustomerID         string
	CustomerName       string
	CustomerAvatarPath *string
	SourceID           uuid.UUID
	SourceType         SourceType
	Rating             int16
	Comment            *string
	CreatedAt          time.Time
}
