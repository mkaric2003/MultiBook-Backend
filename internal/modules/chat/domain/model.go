// Package domain contains chat models free from HTTP, Firebase, and PostgreSQL dependencies.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID                   uuid.UUID
	BusinessID           *uuid.UUID
	LegacyBusinessID     *string
	BusinessOwnerID      string
	CustomerID           string
	BusinessName         string
	BusinessImagePath    *string
	CustomerName         string
	CustomerImagePath    *string
	LastMessageID        *uuid.UUID
	LastMessageText      string
	LastMessageAt        *time.Time
	LastSenderID         *string
	LastReadAtCustomer   *time.Time
	LastReadAtBusiness   *time.Time
	TypingUserID         *string
	TypingExpiresAt      *time.Time
	ActiveParticipantIDs []string
	UnreadCustomerCount  int
	UnreadBusinessCount  int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Message struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	SenderID       string
	Text           string
	CreatedAt      time.Time
}

type PushJob struct {
	MessageID         uuid.UUID
	RecipientID       string
	Attempts          int
	ConversationID    uuid.UUID
	BusinessID        string
	BusinessOwnerID   string
	BusinessName      string
	BusinessImagePath *string
	CustomerID        string
	CustomerName      string
	CustomerImagePath *string
	SenderID          string
	MessageText       string
}
