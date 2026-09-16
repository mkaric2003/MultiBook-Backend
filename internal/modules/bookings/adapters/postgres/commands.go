package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/bookings/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/bookings/domain"
	promotionsapp "github.com/mkaric2003/multibook-backend/internal/modules/promotions/application"
	promotionsdomain "github.com/mkaric2003/multibook-backend/internal/modules/promotions/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

type Repository struct {
	pool      *pgxpool.Pool
	discounts promotionsapp.DiscountApplier
}

func NewRepository(pool *pgxpool.Pool, discounts promotionsapp.DiscountApplier) *Repository {
	return &Repository{pool: pool, discounts: discounts}
}

var _ application.Repository = (*Repository)(nil)

func (r *Repository) CreateForCustomer(ctx context.Context, customerID string, businessID uuid.UUID, input domain.CreateInput) (domain.Booking, error) {
	var booking domain.Booking
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		var inventory, ownerID, name, city, image, currency string
		var basePrice int64
		err := tx.QueryRow(ctx, `SELECT d.inventory_type,b.owner_id,b.name,COALESCE(NULLIF(l.city,''),l.address,''),COALESCE((SELECT storage_path FROM business_media WHERE business_id=b.id AND media_type='cover'),''),b.currency,COALESCE(d.base_price_minor,0) FROM businesses b JOIN stay_details d ON d.business_id=b.id LEFT JOIN business_locations l ON l.business_id=b.id AND l.is_primary WHERE b.id=$1 AND b.type='stay' AND b.status='active' AND b.deleted_at IS NULL FOR UPDATE OF b`, businessID).Scan(&inventory, &ownerID, &name, &city, &image, &currency, &basePrice)
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("load stay: %w", err)
		}
		price := basePrice
		var roomName *string
		if inventory == "multiple_units" {
			if input.StayUnitTypeID == nil {
				return fmt.Errorf("%w: room_type_id is required", application.ErrValidation)
			}
			var quantity int16
			err = tx.QueryRow(ctx, `SELECT name,price_per_night_minor,quantity FROM stay_unit_types WHERE id=$1 AND business_id=$2 AND is_active AND deleted_at IS NULL FOR UPDATE`, *input.StayUnitTypeID, businessID).Scan(&roomName, &price, &quantity)
			if errors.Is(err, pgx.ErrNoRows) {
				return application.ErrNotFound
			}
			if err != nil {
				return fmt.Errorf("load room type: %w", err)
			}
			var count int16
			if err = tx.QueryRow(ctx, `SELECT count(*) FROM stay_bookings WHERE stay_unit_type_id=$1 AND status='confirmed' AND check_in < $3::date AND check_out > $2::date`, *input.StayUnitTypeID, input.CheckIn, input.CheckOut).Scan(&count); err != nil {
				return fmt.Errorf("check room availability: %w", err)
			}
			if count >= quantity {
				return application.ErrConflict
			}
		} else {
			if input.StayUnitTypeID != nil {
				return fmt.Errorf("%w: room_type_id is not valid for this stay", application.ErrValidation)
			}
			var exists bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM stay_bookings WHERE business_id=$1 AND status='confirmed' AND check_in < $3::date AND check_out > $2::date)`, businessID, input.CheckIn, input.CheckOut).Scan(&exists); err != nil {
				return fmt.Errorf("check stay availability: %w", err)
			}
			if exists {
				return application.ErrConflict
			}
		}
		types := make([]string, len(input.SelectedExtras))
		for i, e := range input.SelectedExtras {
			types[i] = e.Type
		}
		rows, err := tx.Query(ctx, `SELECT extra_type,price_minor,pricing_unit FROM stay_extras WHERE business_id=$1 AND extra_type=ANY($2) ORDER BY extra_type`, businessID, types)
		if err != nil {
			return fmt.Errorf("load extras: %w", err)
		}
		defer rows.Close()
		extras := make([]map[string]any, 0, len(types))
		var extrasTotal int64
		for rows.Next() {
			var typ, unit string
			var priceExtra int64
			if err = rows.Scan(&typ, &priceExtra, &unit); err != nil {
				return err
			}
			perNight := unit == "per_night"
			extras = append(extras, map[string]any{"type": typ, "price": priceExtra, "isPerNight": perNight, "isPerHour": unit == "per_hour"})
			mult := int64(1)
			if perNight {
				mult = int64(days(input.CheckIn, input.CheckOut))
			}
			extrasTotal += priceExtra * mult
		}
		if err = rows.Err(); err != nil {
			return err
		}
		if len(extras) != len(types) {
			return fmt.Errorf("%w: one or more extras are unavailable", application.ErrValidation)
		}
		nights := int64(days(input.CheckIn, input.CheckOut))
		subtotal := price * nights
		baseAmount := subtotal + extrasTotal
		var discount int64
		if applied, applyErr := r.discounts.Apply(ctx, tx, promotionsdomain.DiscountInput{BusinessID: businessID, PromoCode: input.PromoCode, SubtotalMinor: baseAmount, Nights: int(nights)}); applyErr != nil {
			return applyErr
		} else if applied != nil {
			discount = applied.AmountMinor
		}
		discountedBase := baseAmount - discount
		cleaning := int64(2500)
		service := int64(math.Round(float64(discountedBase) * .05))
		taxes := int64(math.Round(float64(discountedBase+cleaning+service) * .08))
		total := discountedBase + cleaning + service + taxes
		originalService := int64(math.Round(float64(baseAmount) * .05))
		originalTaxes := int64(math.Round(float64(baseAmount+cleaning+originalService) * .08))
		originalTotal := baseAmount + cleaning + originalService + originalTaxes
		paymentStatus := "paid"
		if input.PaymentMethod == "cash" {
			paymentStatus = "pending"
		}
		raw, _ := json.Marshal(extras)
		code := "BK-" + uuid.NewString()[:8]
		var id uuid.UUID
		var createdAt, updatedAt time.Time
		err = tx.QueryRow(ctx, `INSERT INTO stay_bookings(business_id,business_owner_id,customer_id,stay_unit_type_id,customer_name,customer_email,customer_avatar_path,check_in,check_out,adults,children,infants,price_per_night_minor,room_subtotal_minor,discount_minor,cleaning_fee_minor,service_fee_minor,taxes_minor,total_minor,original_total_minor,payment_status,payment_method,confirmation_code,currency,selected_extras) VALUES($1,$2,$3,$4,$5,$6,(SELECT avatar_storage_path FROM users WHERE id=$3),$7::date,$8::date,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20::stay_payment_status,$21,$22,$23,$24::jsonb) RETURNING id,created_at,updated_at`, businessID, ownerID, customerID, input.StayUnitTypeID, input.CustomerName, input.CustomerEmail, input.CheckIn, input.CheckOut, input.Adults, input.Children, input.Infants, price, subtotal, discount, cleaning, service, taxes, total, originalTotal, paymentStatus, input.PaymentMethod, code, currency, raw).Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			return fmt.Errorf("create booking: %w", err)
		}
		booking = domain.Booking{ID: id, BusinessID: businessID, BusinessOwnerID: ownerID, CustomerID: customerID, CustomerName: input.CustomerName, CustomerEmail: input.CustomerEmail, BusinessName: name, BusinessCity: city, BusinessImageURL: image, RoomType: roomName, RoomTypeID: input.StayUnitTypeID, CheckIn: input.CheckIn, CheckOut: input.CheckOut, Adults: input.Adults, Children: input.Children, Infants: input.Infants, PricePerNight: price, SelectedExtras: extras, RoomSubtotal: subtotal, DiscountAmount: discount, CleaningFee: cleaning, ServiceFee: service, Taxes: taxes, Total: total, OriginalTotal: originalTotal, PaymentStatus: paymentStatus, PaymentMethod: input.PaymentMethod, ConfirmationCode: code, Currency: currency, Status: "confirmed", CreatedAt: createdAt, UpdatedAt: updatedAt}
		return nil
	})
	if err != nil {
		return domain.Booking{}, err
	}
	return booking, nil
}
func days(checkIn, checkOut string) int {
	start, _ := time.Parse(time.DateOnly, checkIn)
	end, _ := time.Parse(time.DateOnly, checkOut)
	return int(end.Sub(start).Hours() / 24)
}
