package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/drafts/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/drafts/domain"
)

type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.Queries = (*Queries)(nil)

func (q *Queries) GetBooking(ctx context.Context, customerID string) (domain.BookingDraft, error) {
	var draft domain.BookingDraft
	err := q.pool.QueryRow(ctx, `SELECT d.business_id,b.name,COALESCE(NULLIF(l.city,''),l.address,''),COALESCE((SELECT storage_path FROM business_media WHERE business_id=b.id AND media_type='cover'),''),COALESCE(sd.base_price_minor,0),d.check_in::text,d.check_out::text,d.adults,d.children,d.infants,d.room_type_id,d.selected_extras,d.updated_at FROM booking_drafts d JOIN businesses b ON b.id=d.business_id JOIN stay_details sd ON sd.business_id=b.id LEFT JOIN business_locations l ON l.business_id=b.id AND l.is_primary WHERE d.customer_id=$1`, customerID).Scan(&draft.BusinessID, &draft.BusinessName, &draft.BusinessLocation, &draft.BusinessImageURL, &draft.PricePerNight, &draft.CheckIn, &draft.CheckOut, &draft.Adults, &draft.Children, &draft.Infants, &draft.RoomTypeID, &draft.SelectedExtras, &draft.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.BookingDraft{}, application.ErrNotFound
	}
	if err != nil {
		return domain.BookingDraft{}, fmt.Errorf("get booking draft: %w", err)
	}
	return draft, nil
}

func (q *Queries) GetAppointment(ctx context.Context, customerID string) (domain.AppointmentDraft, error) {
	var draft domain.AppointmentDraft
	err := q.pool.QueryRow(ctx, `SELECT d.business_id,b.name,COALESCE((SELECT storage_path FROM business_media WHERE business_id=b.id AND media_type='cover'),''),d.selected_offering_ids,d.selected_provider_id,s.name,d.appointment_date::text,d.start_minutes,d.selected_add_on_ids,d.updated_at FROM appointment_drafts d JOIN businesses b ON b.id=d.business_id LEFT JOIN service_staff s ON s.id=d.selected_provider_id WHERE d.customer_id=$1`, customerID).Scan(&draft.BusinessID, &draft.BusinessName, &draft.BusinessImageURL, &draft.SelectedOfferingIDs, &draft.SelectedProviderID, &draft.SelectedProviderName, &draft.AppointmentDate, &draft.StartMinutes, &draft.SelectedAddOnIDs, &draft.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AppointmentDraft{}, application.ErrNotFound
	}
	if err != nil {
		return domain.AppointmentDraft{}, fmt.Errorf("get appointment draft: %w", err)
	}
	return draft, nil
}
