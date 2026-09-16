// Package application contains support ticket use cases and ports.
package application

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/domain"
)

var (
	ErrForbidden  = errors.New("forbidden")
	ErrNotFound   = errors.New("customer not found")
	ErrValidation = errors.New("validation failed")
)

const (
	defaultPageSize = 30
	maximumPageSize = 60
	maximumSubject  = 160
	maximumMessage  = 5000
)

type Actor struct {
	ID   string
	Role string
}

type Page struct {
	Items      []domain.Ticket
	NextCursor *string
}

type Service struct {
	commands CommandRepository
	queries  Queries
}

func NewService(commands CommandRepository, queries Queries) *Service {
	return &Service{commands: commands, queries: queries}
}

func (service *Service) Create(ctx context.Context, actor Actor, input domain.CreateInput) (domain.Ticket, error) {
	if err := authorizeCustomer(actor); err != nil {
		return domain.Ticket{}, err
	}
	input.Subject = strings.TrimSpace(input.Subject)
	input.Message = strings.TrimSpace(input.Message)
	if !input.Category.Valid() || input.Subject == "" || input.Message == "" || utf8.RuneCountInString(input.Subject) > maximumSubject || utf8.RuneCountInString(input.Message) > maximumMessage {
		return domain.Ticket{}, ErrValidation
	}
	return service.commands.Create(ctx, actor.ID, input)
}

func (service *Service) List(ctx context.Context, actor Actor, offset, pageSize int) (Page, error) {
	if err := authorizeCustomer(actor); err != nil {
		return Page{}, err
	}
	if offset < 0 {
		return Page{}, ErrValidation
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maximumPageSize {
		pageSize = maximumPageSize
	}
	items, err := service.queries.List(ctx, actor.ID, pageSize+1, offset)
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: items}
	if len(items) > pageSize {
		page.Items = items[:pageSize]
		next := strconv.Itoa(offset + pageSize)
		page.NextCursor = &next
	}
	return page, nil
}

func authorizeCustomer(actor Actor) error {
	if strings.TrimSpace(actor.ID) == "" || actor.Role != "customer" {
		return ErrForbidden
	}
	return nil
}
