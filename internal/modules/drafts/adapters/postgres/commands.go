// Package postgres provides the PostgreSQL adapter for customer drafts.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/drafts/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/drafts/domain"
)

type CommandRepository struct{ pool *pgxpool.Pool }

func NewCommandRepository(pool *pgxpool.Pool) *CommandRepository {
	return &CommandRepository{pool: pool}
}

var _ application.Commands = (*CommandRepository)(nil)

func (r *CommandRepository) UpsertBooking(ctx context.Context, customerID string, input domain.BookingInput) error {
	var valid bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM businesses b JOIN stay_details d ON d.business_id=b.id WHERE b.id=$1 AND b.type='stay' AND b.status='active' AND b.deleted_at IS NULL AND ($2::uuid IS NULL OR EXISTS(SELECT 1 FROM stay_unit_types u WHERE u.id=$2 AND u.business_id=b.id AND u.is_active AND u.deleted_at IS NULL)))`, input.BusinessID, input.RoomTypeID).Scan(&valid)
	if err != nil {
		return fmt.Errorf("validate booking draft: %w", err)
	}
	if !valid {
		return application.ErrNotFound
	}
	extras := input.SelectedExtras
	if len(extras) == 0 {
		extras = json.RawMessage("[]")
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO booking_drafts(customer_id,business_id,check_in,check_out,adults,children,infants,room_type_id,selected_extras) VALUES($1,$2,$3::date,$4::date,$5,$6,$7,$8,$9::jsonb)
		ON CONFLICT(customer_id) DO UPDATE SET business_id=EXCLUDED.business_id,check_in=EXCLUDED.check_in,check_out=EXCLUDED.check_out,adults=EXCLUDED.adults,children=EXCLUDED.children,infants=EXCLUDED.infants,room_type_id=EXCLUDED.room_type_id,selected_extras=EXCLUDED.selected_extras`, customerID, input.BusinessID, input.CheckIn, input.CheckOut, input.Adults, input.Children, input.Infants, input.RoomTypeID, extras)
	if err != nil {
		return fmt.Errorf("upsert booking draft: %w", err)
	}
	return nil
}

func (r *CommandRepository) DeleteBooking(ctx context.Context, customerID string) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM booking_drafts WHERE customer_id=$1`, customerID); err != nil {
		return fmt.Errorf("delete booking draft: %w", err)
	}
	return nil
}

func (r *CommandRepository) UpsertAppointment(ctx context.Context, customerID string, input domain.AppointmentInput) error {
	var valid bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM businesses b JOIN service_details d ON d.business_id=b.id WHERE b.id=$1 AND b.type='service' AND b.status='active' AND b.deleted_at IS NULL AND ($2::uuid IS NULL OR EXISTS(SELECT 1 FROM service_staff s WHERE s.id=$2 AND s.business_id=b.id AND s.is_active AND s.deleted_at IS NULL)))`, input.BusinessID, input.SelectedProviderID).Scan(&valid)
	if err != nil {
		return fmt.Errorf("validate appointment draft: %w", err)
	}
	if !valid {
		return application.ErrNotFound
	}
	addOns := input.SelectedAddOnIDs
	if len(addOns) == 0 {
		addOns = json.RawMessage("[]")
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO appointment_drafts(customer_id,business_id,selected_offering_ids,selected_provider_id,appointment_date,start_minutes,selected_add_on_ids) VALUES($1,$2,$3,$4,$5::date,$6,$7::jsonb)
		ON CONFLICT(customer_id) DO UPDATE SET business_id=EXCLUDED.business_id,selected_offering_ids=EXCLUDED.selected_offering_ids,selected_provider_id=EXCLUDED.selected_provider_id,appointment_date=EXCLUDED.appointment_date,start_minutes=EXCLUDED.start_minutes,selected_add_on_ids=EXCLUDED.selected_add_on_ids`, customerID, input.BusinessID, input.SelectedOfferingIDs, input.SelectedProviderID, input.AppointmentDate, input.StartMinutes, addOns)
	if err != nil {
		return fmt.Errorf("upsert appointment draft: %w", err)
	}
	return nil
}

func (r *CommandRepository) DeleteAppointment(ctx context.Context, customerID string) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM appointment_drafts WHERE customer_id=$1`, customerID); err != nil {
		return fmt.Errorf("delete appointment draft: %w", err)
	}
	return nil
}
