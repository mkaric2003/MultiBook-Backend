// Package application contains notification use cases and their ports.
package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mkaric2003/multibook-backend/internal/modules/notifications/domain"
)

var (
	ErrNotFound        = errors.New("notification not found")
	ErrValidation      = errors.New("validation failed")
	ErrPushUnavailable = errors.New("push transport unavailable")
)

type Service struct {
	commands NotificationCommands
	queries  NotificationQueries
	push     PushSender
}

// DeliverPush sends a notification through FCM without persisting an in-app
// notification. Chat owns its durable outbox and calls this operation only
// after it has atomically committed the message.
func (s *Service) DeliverPush(ctx context.Context, event domain.Event) (bool, error) {
	if event.ID == "" || event.RecipientID == "" || event.Kind == "" {
		return false, ErrValidation
	}
	if s.push == nil {
		return false, ErrPushUnavailable
	}
	tokens, err := s.queries.DeviceTokens(ctx, event.RecipientID)
	if err != nil {
		return false, err
	}
	if len(tokens) == 0 {
		return false, nil
	}
	if err := s.push.Send(ctx, tokens, event); err != nil {
		return false, err
	}
	return true, nil
}

func NewService(commands NotificationCommands, queries NotificationQueries, push PushSender) *Service {
	return &Service{commands: commands, queries: queries, push: push}
}

func (s *Service) Publish(ctx context.Context, event domain.Event) {
	if event.ID == "" || event.RecipientID == "" {
		return
	}
	if err := s.commands.Create(ctx, event); err != nil {
		return
	}
	if s.push == nil {
		return
	}
	tokens, err := s.queries.DeviceTokens(ctx, event.RecipientID)
	if err == nil && len(tokens) > 0 {
		_ = s.push.Send(ctx, tokens, event)
	}
}

func (s *Service) List(ctx context.Context, userID string, offset, limit int) (Page, error) {
	if strings.TrimSpace(userID) == "" {
		return Page{}, ErrValidation
	}
	if offset < 0 {
		return Page{}, fmt.Errorf("%w: cursor is invalid", ErrValidation)
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	return s.queries.List(ctx, userID, ListInput{Offset: offset, Limit: limit})
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	if strings.TrimSpace(userID) == "" {
		return 0, ErrValidation
	}
	return s.queries.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(notificationID) == "" {
		return ErrValidation
	}
	return s.commands.MarkRead(ctx, userID, notificationID)
}

func (s *Service) UpsertDevice(ctx context.Context, userID, deviceID, token string) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(token) == "" {
		return ErrValidation
	}
	return s.commands.UpsertDevice(ctx, userID, deviceID, token)
}

func (s *Service) DeleteDevice(ctx context.Context, userID, deviceID string) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(deviceID) == "" {
		return ErrValidation
	}
	return s.commands.DeleteDevice(ctx, userID, deviceID)
}
