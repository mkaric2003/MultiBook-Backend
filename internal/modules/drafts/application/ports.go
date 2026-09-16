package application

import (
	"context"

	"github.com/mkaric2003/multibook-backend/internal/modules/drafts/domain"
)

type Commands interface {
	UpsertBooking(context.Context, string, domain.BookingInput) error
	DeleteBooking(context.Context, string) error
	UpsertAppointment(context.Context, string, domain.AppointmentInput) error
	DeleteAppointment(context.Context, string) error
}

type Queries interface {
	GetBooking(context.Context, string) (domain.BookingDraft, error)
	GetAppointment(context.Context, string) (domain.AppointmentDraft, error)
}
