package http

import (
	"time"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/domain"
)

type ticketResponse struct {
	ID            uuid.UUID       `json:"id"`
	CustomerID    string          `json:"customerId"`
	CustomerName  string          `json:"customerName"`
	CustomerEmail string          `json:"customerEmail"`
	Category      domain.Category `json:"category"`
	Status        domain.Status   `json:"status"`
	Subject       string          `json:"subject"`
	Message       string          `json:"message"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

type pageResponse struct {
	Items      []ticketResponse `json:"items"`
	NextCursor *string          `json:"nextCursor"`
}

func response(ticket domain.Ticket) ticketResponse {
	return ticketResponse{
		ID: ticket.ID, CustomerID: ticket.CustomerID, CustomerName: ticket.CustomerName,
		CustomerEmail: ticket.CustomerEmail, Category: ticket.Category, Status: ticket.Status,
		Subject: ticket.Subject, Message: ticket.Message, CreatedAt: ticket.CreatedAt, UpdatedAt: ticket.UpdatedAt,
	}
}

func pageResponseFrom(page application.Page) pageResponse {
	items := make([]ticketResponse, len(page.Items))
	for index, ticket := range page.Items {
		items[index] = response(ticket)
	}
	return pageResponse{Items: items, NextCursor: page.NextCursor}
}
