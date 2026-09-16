package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/appointments/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/appointments/domain"
)

// Queries is the Postgres read-side adapter for appointment availability and
// appointment list screens.
type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.AppointmentQueries = (*Queries)(nil)

func (r *Queries) AvailableSlots(ctx context.Context, businessID, staffID uuid.UUID, input domain.AvailabilityInput) (domain.AvailableSlots, error) {
	return r.availableSlots(ctx, businessID, staffID, input)
}

func (r *Queries) ListForActor(ctx context.Context, actorID, role string, input domain.ListInput) (domain.Page, error) {
	return r.listForActor(ctx, actorID, role, input)
}

func (r *Queries) availableSlots(ctx context.Context, businessID, staffID uuid.UUID, input domain.AvailabilityInput) (domain.AvailableSlots, error) {
	var timeZone string
	err := r.pool.QueryRow(ctx, `SELECT d.time_zone FROM businesses b JOIN service_details d ON d.business_id=b.id JOIN service_staff s ON s.business_id=b.id WHERE b.id=$1 AND b.type='service' AND b.status='active' AND b.deleted_at IS NULL AND s.id=$2 AND s.is_active AND s.deleted_at IS NULL`, businessID, staffID).Scan(&timeZone)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AvailableSlots{}, application.ErrNotFound
	}
	if err != nil {
		return domain.AvailableSlots{}, fmt.Errorf("load service availability: %w", err)
	}
	offerings, err := loadOfferings(ctx, r.pool, businessID, staffID, input.OfferingIDs)
	if err != nil {
		return domain.AvailableSlots{}, err
	}
	if len(offerings) != len(input.OfferingIDs) {
		return domain.AvailableSlots{}, fmt.Errorf("%w: one or more offerings are unavailable for this staff member", application.ErrValidation)
	}
	var duration int16
	for _, offering := range offerings {
		duration += offering.DurationMinutes
	}
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		return domain.AvailableSlots{}, fmt.Errorf("load service time zone: %w", err)
	}
	date, _ := time.Parse(time.DateOnly, input.AppointmentDate)
	localStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, location)
	weekday := int16((int(localStart.Weekday()) + 6) % 7)
	rows, err := r.pool.Query(ctx, `
		SELECT minute FROM generate_series(0, 1410, 30) AS minute
		WHERE minute + $3 <= 1440
		  AND EXISTS(SELECT 1 FROM service_staff_weekly_availability w WHERE w.staff_id=$1 AND w.weekday=$2 AND w.start_minutes <= minute AND w.end_minutes >= minute+$3)
		  AND NOT EXISTS(SELECT 1 FROM service_staff_availability_blocks b WHERE b.staff_id=$1 AND b.blocked_range && tstzrange($4 + minute * interval '1 minute', $4 + (minute+$3) * interval '1 minute','[)'))
		  AND NOT EXISTS(SELECT 1 FROM service_appointments a WHERE a.staff_id=$1 AND a.status='confirmed' AND a.scheduled_range && tstzrange($4 + minute * interval '1 minute', $4 + (minute+$3) * interval '1 minute','[)'))
		ORDER BY minute`, staffID, weekday, duration, localStart)
	if err != nil {
		return domain.AvailableSlots{}, fmt.Errorf("query available slots: %w", err)
	}
	defer rows.Close()
	starts := []int16{}
	for rows.Next() {
		var minute int16
		if err = rows.Scan(&minute); err != nil {
			return domain.AvailableSlots{}, err
		}
		starts = append(starts, minute)
	}
	if err = rows.Err(); err != nil {
		return domain.AvailableSlots{}, err
	}
	return domain.AvailableSlots{StaffID: staffID, AppointmentDate: input.AppointmentDate, DurationMinutes: duration, StartMinutes: starts}, nil
}

func (r *Queries) listForActor(ctx context.Context, actorID, role string, input domain.ListInput) (domain.Page, error) {
	where := "a.customer_id=$1"
	args := []any{actorID}
	if role == "provider" {
		where = "b.owner_id=$1"
	}
	if input.BusinessID != uuid.Nil {
		args = append(args, input.BusinessID)
		where += fmt.Sprintf(" AND a.business_id=$%d", len(args))
	}
	if input.Status != "" {
		args = append(args, input.Status)
		where += fmt.Sprintf(" AND a.status=$%d", len(args))
	}
	args = append(args, input.Limit+1, input.Offset)
	rows, err := r.pool.Query(ctx, `SELECT `+presentationColumns+` FROM service_appointments a JOIN businesses b ON b.id=a.business_id WHERE `+where+fmt.Sprintf(" ORDER BY a.appointment_date DESC,a.start_minutes DESC,a.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return domain.Page{}, fmt.Errorf("list appointments: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Appointment, 0, input.Limit)
	for rows.Next() {
		item, err := scanPresentation(rows)
		if err != nil {
			return domain.Page{}, err
		}
		item.Offerings, err = appointmentOfferings(ctx, r.pool, item.ID)
		if err != nil {
			return domain.Page{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.Page{}, err
	}
	page := domain.Page{Items: items}
	if len(items) > input.Limit {
		page.Items = items[:input.Limit]
		next := strconv.Itoa(input.Offset + input.Limit)
		page.NextCursor = &next
	}
	return page, nil
}
