package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mkaric2003/multibook-backend/internal/modules/bookings/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/bookings/domain"
)

const bookingColumns = `sb.id,sb.business_id,sb.business_owner_id,sb.customer_id,sb.customer_name,sb.customer_email,b.name,COALESCE(NULLIF(l.city,''),l.address,''),COALESCE((SELECT storage_path FROM business_media WHERE business_id=b.id AND media_type='cover'),''),rt.name,sb.stay_unit_type_id,sb.check_in::text,sb.check_out::text,sb.adults,sb.children,sb.infants,sb.price_per_night_minor,sb.selected_extras,sb.room_subtotal_minor,sb.discount_minor,sb.cleaning_fee_minor,sb.service_fee_minor,sb.taxes_minor,sb.total_minor,sb.original_total_minor,sb.payment_status::text,sb.payment_method,sb.confirmation_code,sb.currency,sb.status::text,sb.created_at,sb.updated_at`

const bookingFrom = ` FROM stay_bookings sb JOIN businesses b ON b.id=sb.business_id LEFT JOIN business_locations l ON l.business_id=b.id AND l.is_primary LEFT JOIN stay_unit_types rt ON rt.id=sb.stay_unit_type_id`

func (r *Repository) ListForCustomer(ctx context.Context, customerID string, input domain.ListInput) (domain.Page, error) {
	return r.list(ctx, "sb.customer_id=$1", customerID, input)
}

func (r *Repository) ListForProvider(ctx context.Context, providerID string, input domain.ListInput) (domain.Page, error) {
	return r.list(ctx, "b.owner_id=$1", providerID, input)
}

func (r *Repository) list(ctx context.Context, actorWhere string, actorID string, input domain.ListInput) (domain.Page, error) {
	args := []any{actorID}
	where := actorWhere
	if input.BusinessID != nil {
		args = append(args, *input.BusinessID)
		where += fmt.Sprintf(" AND sb.business_id=$%d", len(args))
	}
	if input.Status != "" {
		args = append(args, input.Status)
		where += fmt.Sprintf(" AND sb.status=$%d", len(args))
	}
	args = append(args, input.Limit+1, input.Offset)
	rows, err := r.pool.Query(ctx, `SELECT `+bookingColumns+bookingFrom+` WHERE `+where+fmt.Sprintf(" ORDER BY sb.created_at DESC,sb.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return domain.Page{}, fmt.Errorf("list customer bookings: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Booking, 0, input.Limit)
	for rows.Next() {
		item, err := scanBooking(rows)
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

func (r *Repository) UpdateStatusForProvider(ctx context.Context, providerID string, bookingID uuid.UUID, status string) (domain.Booking, error) {
	var condition string
	switch status {
	case "declined":
		condition = "sb.status='confirmed'"
	case "completed":
		condition = "sb.status='confirmed' AND sb.check_out <= current_date"
	case "no_show":
		condition = "sb.status IN ('confirmed','completed') AND (sb.payment_method='cash' OR sb.payment_status='pending') AND sb.check_out <= current_date"
	}
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `UPDATE stay_bookings sb SET status=$3 WHERE sb.id=$1 AND sb.business_owner_id=$2 AND `+condition+` RETURNING sb.id`, bookingID, providerID, status).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Booking{}, application.ErrNotFound
	}
	if err != nil {
		return domain.Booking{}, fmt.Errorf("update provider booking status: %w", err)
	}
	booking, err := scanBooking(r.pool.QueryRow(ctx, `SELECT `+bookingColumns+bookingFrom+` WHERE sb.id=$1 AND b.owner_id=$2`, id, providerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Booking{}, application.ErrNotFound
	}
	if err != nil {
		return domain.Booking{}, fmt.Errorf("load provider booking: %w", err)
	}
	return booking, nil
}

func (r *Repository) CancelForCustomer(ctx context.Context, customerID string, bookingID uuid.UUID) (domain.Booking, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `UPDATE stay_bookings SET status='cancelled' WHERE id=$1 AND customer_id=$2 AND status='confirmed' AND check_out >= current_date RETURNING id`, bookingID, customerID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Booking{}, application.ErrNotFound
	}
	if err != nil {
		return domain.Booking{}, fmt.Errorf("cancel booking: %w", err)
	}
	booking, err := r.bookingForCustomer(ctx, customerID, id)
	if err != nil {
		return domain.Booking{}, err
	}
	return booking, nil
}

func (r *Repository) UnavailableRanges(ctx context.Context, businessID uuid.UUID, roomTypeID *uuid.UUID) ([]domain.UnavailableRange, error) {
	var inventory string
	err := r.pool.QueryRow(ctx, `SELECT d.inventory_type FROM businesses b JOIN stay_details d ON d.business_id=b.id WHERE b.id=$1 AND b.type='stay' AND b.status='active' AND b.deleted_at IS NULL`, businessID).Scan(&inventory)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load stay availability: %w", err)
	}
	if inventory == "multiple_units" && roomTypeID == nil {
		return nil, fmt.Errorf("%w: room_type_id is required", application.ErrValidation)
	}
	if inventory == "single_unit" && roomTypeID != nil {
		return nil, fmt.Errorf("%w: room_type_id is not valid for this stay", application.ErrValidation)
	}
	args := []any{businessID}
	where := "business_id=$1 AND status='confirmed'"
	if roomTypeID != nil {
		var quantity int16
		err = r.pool.QueryRow(ctx, `SELECT quantity FROM stay_unit_types WHERE id=$1 AND business_id=$2 AND is_active AND deleted_at IS NULL`, *roomTypeID, businessID).Scan(&quantity)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("load room type availability: %w", err)
		}
		return r.fullRoomTypeDates(ctx, businessID, *roomTypeID, quantity)
	}
	rows, err := r.pool.Query(ctx, `SELECT check_in::text,check_out::text FROM stay_bookings WHERE `+where+` ORDER BY check_in,check_out`, args...)
	if err != nil {
		return nil, fmt.Errorf("list unavailable ranges: %w", err)
	}
	defer rows.Close()
	ranges := []domain.UnavailableRange{}
	for rows.Next() {
		var item domain.UnavailableRange
		if err = rows.Scan(&item.CheckIn, &item.CheckOut); err != nil {
			return nil, err
		}
		ranges = append(ranges, item)
	}
	return ranges, rows.Err()
}

func (r *Repository) fullRoomTypeDates(ctx context.Context, businessID, roomTypeID uuid.UUID, quantity int16) ([]domain.UnavailableRange, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT day::date::text,(day + interval '1 day')::date::text
		FROM generate_series(
			(SELECT min(check_in) FROM stay_bookings WHERE business_id=$1 AND stay_unit_type_id=$2 AND status='confirmed'),
			(SELECT max(check_out - 1) FROM stay_bookings WHERE business_id=$1 AND stay_unit_type_id=$2 AND status='confirmed'),
			interval '1 day'
		) AS day
		WHERE (
			SELECT count(*) FROM stay_bookings
			WHERE business_id=$1 AND stay_unit_type_id=$2 AND status='confirmed'
			  AND check_in <= day::date AND check_out > day::date
		) >= $3
		ORDER BY day`, businessID, roomTypeID, quantity)
	if err != nil {
		return nil, fmt.Errorf("list full room type dates: %w", err)
	}
	defer rows.Close()
	ranges := []domain.UnavailableRange{}
	for rows.Next() {
		var item domain.UnavailableRange
		if err = rows.Scan(&item.CheckIn, &item.CheckOut); err != nil {
			return nil, err
		}
		ranges = append(ranges, item)
	}
	return ranges, rows.Err()
}

func (r *Repository) bookingForCustomer(ctx context.Context, customerID string, bookingID uuid.UUID) (domain.Booking, error) {
	booking, err := scanBooking(r.pool.QueryRow(ctx, `SELECT `+bookingColumns+bookingFrom+` WHERE sb.id=$1 AND sb.customer_id=$2`, bookingID, customerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Booking{}, application.ErrNotFound
	}
	if err != nil {
		return domain.Booking{}, fmt.Errorf("load customer booking: %w", err)
	}
	return booking, nil
}

type rowScanner interface{ Scan(...any) error }

func scanBooking(row rowScanner) (domain.Booking, error) {
	var booking domain.Booking
	var extras json.RawMessage
	err := row.Scan(&booking.ID, &booking.BusinessID, &booking.BusinessOwnerID, &booking.CustomerID, &booking.CustomerName, &booking.CustomerEmail, &booking.BusinessName, &booking.BusinessCity, &booking.BusinessImageURL, &booking.RoomType, &booking.RoomTypeID, &booking.CheckIn, &booking.CheckOut, &booking.Adults, &booking.Children, &booking.Infants, &booking.PricePerNight, &extras, &booking.RoomSubtotal, &booking.DiscountAmount, &booking.CleaningFee, &booking.ServiceFee, &booking.Taxes, &booking.Total, &booking.OriginalTotal, &booking.PaymentStatus, &booking.PaymentMethod, &booking.ConfirmationCode, &booking.Currency, &booking.Status, &booking.CreatedAt, &booking.UpdatedAt)
	if err != nil {
		return domain.Booking{}, err
	}
	booking.SelectedExtras = []map[string]any{}
	if len(extras) > 0 {
		if err = json.Unmarshal(extras, &booking.SelectedExtras); err != nil {
			return domain.Booking{}, fmt.Errorf("decode selected extras: %w", err)
		}
	}
	return booking, nil
}
