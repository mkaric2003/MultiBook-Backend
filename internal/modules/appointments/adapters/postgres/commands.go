package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/appointments/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/appointments/domain"
	auditapp "github.com/mkaric2003/multibook-backend/internal/modules/audit/application"
	promotionsapp "github.com/mkaric2003/multibook-backend/internal/modules/promotions/application"
	promotionsdomain "github.com/mkaric2003/multibook-backend/internal/modules/promotions/domain"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

type CommandRepository struct {
	pool      *pgxpool.Pool
	audit     auditapp.Writer
	discounts promotionsapp.DiscountApplier
}

func NewCommandRepository(pool *pgxpool.Pool, audit auditapp.Writer, discounts promotionsapp.DiscountApplier) *CommandRepository {
	return &CommandRepository{pool: pool, audit: audit, discounts: discounts}
}

var _ application.AppointmentRepository = (*CommandRepository)(nil)

func (r *CommandRepository) SetStatus(ctx context.Context, actorID, role string, appointmentID uuid.UUID, status domain.Status) (domain.Appointment, error) {
	var where string
	switch status {
	case domain.StatusCancelled:
		if role != "customer" {
			return domain.Appointment{}, application.ErrForbidden
		}
		where = "customer_id=$2 AND status='confirmed'"
	case domain.StatusDeclined, domain.StatusCompleted:
		if role != "provider" {
			return domain.Appointment{}, application.ErrForbidden
		}
		where = "business_id IN (SELECT id FROM businesses WHERE owner_id=$2) AND status='confirmed'"
	case domain.StatusNoShow:
		if role != "provider" {
			return domain.Appointment{}, application.ErrForbidden
		}
		where = "business_id IN (SELECT id FROM businesses WHERE owner_id=$2) AND status IN ('confirmed','completed') AND payment_method='cash' AND scheduled_range <@ tstzrange(NULL,now(),'()')"
	}
	var appointment domain.Appointment
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		updated, err := scan(tx.QueryRow(ctx, `UPDATE service_appointments SET status=$3 WHERE id=$1 AND `+where+` RETURNING `+columns, appointmentID, actorID, status))
		appointment = updated
		if err != nil {
			return err
		}
		return r.audit.Write(ctx, tx, auditapp.Event{BusinessID: appointment.BusinessID, ActorUserID: actorID, Action: "service_appointment_status_updated", Metadata: map[string]any{"status": status}})
	})
	if errors.Is(err, application.ErrNotFound) {
		return domain.Appointment{}, application.ErrNotFound
	}
	if err != nil {
		return domain.Appointment{}, fmt.Errorf("update appointment status: %w", err)
	}
	return r.presentation(ctx, appointment.ID)
}

func (r *CommandRepository) Reschedule(ctx context.Context, actorID, role string, appointmentID uuid.UUID, input domain.RescheduleInput) (domain.Appointment, error) {
	var updated domain.Appointment
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		updated, err = r.rescheduleInTransaction(ctx, tx, actorID, role, appointmentID, input)
		return err
	})
	if err != nil {
		return domain.Appointment{}, err
	}
	return r.presentation(ctx, updated.ID)
}

func (r *CommandRepository) rescheduleInTransaction(ctx context.Context, tx pgx.Tx, actorID, role string, appointmentID uuid.UUID, input domain.RescheduleInput) (domain.Appointment, error) {
	var appointment domain.Appointment
	var timeZone string
	var ownerID string
	err := tx.QueryRow(ctx, `SELECT `+columns+`,d.time_zone,b.owner_id FROM service_appointments a JOIN businesses b ON b.id=a.business_id JOIN service_details d ON d.business_id=a.business_id WHERE a.id=$1 FOR UPDATE`, appointmentID).Scan(&appointment.ID, &appointment.BusinessID, &appointment.CustomerID, &appointment.StaffID, &appointment.BusinessName, &appointment.ProviderName, &appointment.AppointmentDate, &appointment.StartMinutes, &appointment.EndMinutes, &appointment.ServiceCostMinor, &appointment.OriginalServiceCostMinor, &appointment.DiscountMinor, &appointment.ServiceFeeMinor, &appointment.TaxesMinor, &appointment.TotalMinor, &appointment.ProviderCommissionRate, &appointment.ProviderEarningsMinor, &appointment.Currency, &appointment.PaymentStatus, &appointment.PaymentMethod, &appointment.ConfirmationCode, &appointment.Status, &appointment.CustomerRescheduleCount, &appointment.CreatedAt, &appointment.UpdatedAt, &timeZone, &ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Appointment{}, application.ErrNotFound
	}
	if err != nil {
		return domain.Appointment{}, err
	}
	if appointment.Status != domain.StatusConfirmed {
		return domain.Appointment{}, application.ErrConflict
	}
	if role == "customer" {
		if actorID != appointment.CustomerID || appointment.CustomerRescheduleCount >= 1 {
			return domain.Appointment{}, application.ErrForbidden
		}
	} else if role != "provider" || actorID != ownerID {
		return domain.Appointment{}, application.ErrForbidden
	}
	duration := appointment.EndMinutes - appointment.StartMinutes
	end := input.StartMinutes + duration
	if end > 1440 {
		return domain.Appointment{}, application.ErrValidation
	}
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		return domain.Appointment{}, err
	}
	date, _ := time.Parse(time.DateOnly, input.AppointmentDate)
	startAt := time.Date(date.Year(), date.Month(), date.Day(), 0, int(input.StartMinutes), 0, 0, location)
	endAt := startAt.Add(time.Duration(duration) * time.Minute)
	weekday := int16((int(startAt.Weekday()) + 6) % 7)
	var valid bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM service_staff_weekly_availability WHERE staff_id=$1 AND weekday=$2 AND start_minutes<=$3 AND end_minutes>=$4) AND NOT EXISTS(SELECT 1 FROM service_staff_availability_blocks WHERE staff_id=$1 AND blocked_range && tstzrange($5,$6,'[)'))`, appointment.StaffID, weekday, input.StartMinutes, end, startAt, endAt).Scan(&valid)
	if err != nil {
		return domain.Appointment{}, err
	}
	if !valid {
		return domain.Appointment{}, application.ErrConflict
	}
	increment := 0
	if role == "customer" {
		increment = 1
	}
	updated, err := scan(tx.QueryRow(ctx, `UPDATE service_appointments SET appointment_date=$2,start_minutes=$3,end_minutes=$4,scheduled_range=tstzrange($5,$6,'[)'),customer_reschedule_count=customer_reschedule_count+$7 WHERE id=$1 RETURNING `+columns, appointmentID, date, input.StartMinutes, end, startAt, endAt, increment))
	if err != nil {
		return domain.Appointment{}, mapError(err)
	}
	updated.Offerings, _ = appointmentOfferings(ctx, tx, updated.ID)
	if err = r.audit.Write(ctx, tx, auditapp.Event{BusinessID: updated.BusinessID, ActorUserID: actorID, Action: "service_appointment_rescheduled"}); err != nil {
		return domain.Appointment{}, fmt.Errorf("audit appointment reschedule: %w", err)
	}
	return updated, nil
}

func (r *CommandRepository) CreateForCustomer(ctx context.Context, customerID string, businessID uuid.UUID, input domain.CreateInput) (domain.Appointment, error) {
	var appointment domain.Appointment
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		appointment, err = r.createForCustomerInTransaction(ctx, tx, customerID, businessID, input)
		return err
	})
	if err != nil {
		return domain.Appointment{}, err
	}
	return appointment, nil
}

func (r *CommandRepository) createForCustomerInTransaction(ctx context.Context, tx pgx.Tx, customerID string, businessID uuid.UUID, input domain.CreateInput) (domain.Appointment, error) {
	var businessName, businessOwnerID, businessImageURL, currency, timeZone, providerName string
	var commission float64
	err := tx.QueryRow(ctx, `SELECT b.name,b.owner_id,COALESCE((SELECT storage_path FROM business_media WHERE business_id=b.id AND media_type='cover'),''),b.currency,d.time_zone,s.name,s.commission_rate FROM businesses b JOIN service_details d ON d.business_id=b.id JOIN service_staff s ON s.business_id=b.id WHERE b.id=$1 AND b.type='service' AND b.status='active' AND b.deleted_at IS NULL AND s.id=$2 AND s.is_active AND s.deleted_at IS NULL FOR UPDATE`, businessID, input.StaffID).Scan(&businessName, &businessOwnerID, &businessImageURL, &currency, &timeZone, &providerName, &commission)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Appointment{}, application.ErrNotFound
	}
	if err != nil {
		return domain.Appointment{}, fmt.Errorf("load appointment service: %w", err)
	}
	offerings, err := loadOfferings(ctx, tx, businessID, input.StaffID, input.OfferingIDs)
	if err != nil {
		return domain.Appointment{}, err
	}
	if len(offerings) != len(input.OfferingIDs) {
		return domain.Appointment{}, fmt.Errorf("%w: one or more offerings are unavailable for this staff member", application.ErrValidation)
	}
	var duration int16
	var originalCost int64
	for _, offering := range offerings {
		duration += offering.DurationMinutes
		originalCost += offering.PriceMinor
	}
	endMinutes := input.StartMinutes + duration
	if endMinutes > 1440 {
		return domain.Appointment{}, fmt.Errorf("%w: appointment ends after midnight", application.ErrValidation)
	}
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		return domain.Appointment{}, fmt.Errorf("load service time zone: %w", err)
	}
	date, _ := time.Parse(time.DateOnly, input.AppointmentDate)
	startAt := time.Date(date.Year(), date.Month(), date.Day(), 0, int(input.StartMinutes), 0, 0, location)
	endAt := startAt.Add(time.Duration(duration) * time.Minute)
	weekday := int16((int(startAt.Weekday()) + 6) % 7)
	var inHours bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM service_staff_weekly_availability WHERE staff_id=$1 AND weekday=$2 AND start_minutes <= $3 AND end_minutes >= $4)`, input.StaffID, weekday, input.StartMinutes, endMinutes).Scan(&inHours)
	if err != nil {
		return domain.Appointment{}, fmt.Errorf("check weekly availability: %w", err)
	}
	if !inHours {
		return domain.Appointment{}, fmt.Errorf("%w: selected time is outside staff availability", application.ErrConflict)
	}
	var blocked bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM service_staff_availability_blocks WHERE staff_id=$1 AND blocked_range && tstzrange($2,$3,'[)'))`, input.StaffID, startAt, endAt).Scan(&blocked)
	if err != nil {
		return domain.Appointment{}, fmt.Errorf("check availability blocks: %w", err)
	}
	if blocked {
		return domain.Appointment{}, application.ErrConflict
	}
	var discount int64
	if applied, applyErr := r.discounts.Apply(ctx, tx, promotionsdomain.DiscountInput{BusinessID: businessID, PromoCode: input.PromoCode, SubtotalMinor: originalCost, Nights: 0}); applyErr != nil {
		return domain.Appointment{}, applyErr
	} else if applied != nil {
		discount = applied.AmountMinor
	}
	serviceCost := originalCost - discount
	serviceFee := int64(math.Round(float64(serviceCost) * .085))
	taxes := int64(math.Round(float64(serviceCost+serviceFee) * .1))
	total := serviceCost + serviceFee + taxes
	earnings := int64(math.Round(float64(serviceCost) * commission / 100))
	paymentStatus := "paid"
	if input.PaymentMethod == "cash" {
		paymentStatus = "pending"
	}
	confirmationCode := fmt.Sprintf("AP-%s", uuid.NewString()[:8])
	appointment, err := scan(tx.QueryRow(ctx, `INSERT INTO service_appointments (business_id,customer_id,staff_id,customer_name,customer_email,customer_phone,customer_avatar_path,business_name,provider_name,appointment_date,start_minutes,end_minutes,scheduled_range,service_cost_minor,original_service_cost_minor,discount_minor,service_fee_minor,taxes_minor,total_minor,provider_commission_rate,provider_earnings_minor,currency,payment_status,payment_method,confirmation_code) VALUES ($1,$2,$3,$4,$5,$6,(SELECT avatar_storage_path FROM users WHERE id=$2),$7,$8,$9,$10,$11,tstzrange($12,$13,'[)'),$14,$15,$16,$17,$18,$19,$20,$21,$22,$23::appointment_payment_status,$24,$25) RETURNING `+columns, businessID, customerID, input.StaffID, input.CustomerName, input.CustomerEmail, input.CustomerPhone, businessName, providerName, date, input.StartMinutes, endMinutes, startAt, endAt, serviceCost, originalCost, discount, serviceFee, taxes, total, commission, earnings, currency, paymentStatus, input.PaymentMethod, confirmationCode))
	if err != nil {
		return domain.Appointment{}, mapError(err)
	}
	for _, offering := range offerings {
		if _, err = tx.Exec(ctx, `INSERT INTO service_appointment_offerings (appointment_id,offering_id,name,duration_minutes,price_minor) VALUES($1,$2,$3,$4,$5)`, appointment.ID, offering.ID, offering.Name, offering.DurationMinutes, offering.PriceMinor); err != nil {
			return domain.Appointment{}, fmt.Errorf("save appointment offering: %w", err)
		}
	}
	appointment.Offerings = offerings
	appointment.BusinessOwnerID = businessOwnerID
	appointment.BusinessImageURL = businessImageURL
	appointment.CustomerName = input.CustomerName
	appointment.CustomerEmail = input.CustomerEmail
	appointment.CustomerPhone = input.CustomerPhone
	if err = r.audit.Write(ctx, tx, auditapp.Event{BusinessID: businessID, ActorUserID: customerID, Action: "service_appointment_created"}); err != nil {
		return domain.Appointment{}, fmt.Errorf("audit appointment: %w", err)
	}
	return appointment, nil
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadOfferings(ctx context.Context, tx queryer, businessID, staffID uuid.UUID, ids []uuid.UUID) ([]domain.Offering, error) {
	rows, err := tx.Query(ctx, `SELECT o.id,o.name,o.duration_minutes,o.price_minor FROM service_offerings o JOIN service_staff_offerings so ON so.offering_id=o.id WHERE o.business_id=$1 AND so.staff_id=$2 AND o.id=ANY($3) AND o.is_active AND o.deleted_at IS NULL`, businessID, staffID, ids)
	if err != nil {
		return nil, fmt.Errorf("load appointment offerings: %w", err)
	}
	defer rows.Close()
	items := []domain.Offering{}
	for rows.Next() {
		var item domain.Offering
		if err = rows.Scan(&item.ID, &item.Name, &item.DurationMinutes, &item.PriceMinor); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func appointmentOfferings(ctx context.Context, db queryer, appointmentID uuid.UUID) ([]domain.Offering, error) {
	rows, err := db.Query(ctx, `SELECT offering_id,name,duration_minutes,price_minor FROM service_appointment_offerings WHERE appointment_id=$1 ORDER BY name`, appointmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Offering{}
	for rows.Next() {
		var item domain.Offering
		if err = rows.Scan(&item.ID, &item.Name, &item.DurationMinutes, &item.PriceMinor); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

const columns = `id,business_id,customer_id,staff_id,business_name,provider_name,appointment_date::text,start_minutes,end_minutes,service_cost_minor,original_service_cost_minor,discount_minor,service_fee_minor,taxes_minor,total_minor,provider_commission_rate,provider_earnings_minor,currency,payment_status::text,payment_method,confirmation_code,status::text,customer_reschedule_count,created_at,updated_at`

const presentationColumns = `a.id,a.business_id,a.customer_id,a.staff_id,a.business_name,a.provider_name,a.appointment_date::text,a.start_minutes,a.end_minutes,a.service_cost_minor,a.original_service_cost_minor,a.discount_minor,a.service_fee_minor,a.taxes_minor,a.total_minor,a.provider_commission_rate,a.provider_earnings_minor,a.currency,a.payment_status::text,a.payment_method,a.confirmation_code,a.status::text,a.customer_reschedule_count,a.created_at,a.updated_at,a.customer_name,a.customer_email,a.customer_phone,b.owner_id,COALESCE((SELECT storage_path FROM business_media WHERE business_id=b.id AND media_type='cover'),'')`

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.Appointment, error) {
	var a domain.Appointment
	err := row.Scan(&a.ID, &a.BusinessID, &a.CustomerID, &a.StaffID, &a.BusinessName, &a.ProviderName, &a.AppointmentDate, &a.StartMinutes, &a.EndMinutes, &a.ServiceCostMinor, &a.OriginalServiceCostMinor, &a.DiscountMinor, &a.ServiceFeeMinor, &a.TaxesMinor, &a.TotalMinor, &a.ProviderCommissionRate, &a.ProviderEarningsMinor, &a.Currency, &a.PaymentStatus, &a.PaymentMethod, &a.ConfirmationCode, &a.Status, &a.CustomerRescheduleCount, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, application.ErrNotFound
	}
	return a, err
}

func scanPresentation(row scanner) (domain.Appointment, error) {
	var a domain.Appointment
	err := row.Scan(&a.ID, &a.BusinessID, &a.CustomerID, &a.StaffID, &a.BusinessName, &a.ProviderName, &a.AppointmentDate, &a.StartMinutes, &a.EndMinutes, &a.ServiceCostMinor, &a.OriginalServiceCostMinor, &a.DiscountMinor, &a.ServiceFeeMinor, &a.TaxesMinor, &a.TotalMinor, &a.ProviderCommissionRate, &a.ProviderEarningsMinor, &a.Currency, &a.PaymentStatus, &a.PaymentMethod, &a.ConfirmationCode, &a.Status, &a.CustomerRescheduleCount, &a.CreatedAt, &a.UpdatedAt, &a.CustomerName, &a.CustomerEmail, &a.CustomerPhone, &a.BusinessOwnerID, &a.BusinessImageURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, application.ErrNotFound
	}
	return a, err
}

func (r *CommandRepository) presentation(ctx context.Context, appointmentID uuid.UUID) (domain.Appointment, error) {
	appointment, err := scanPresentation(r.pool.QueryRow(ctx, `SELECT `+presentationColumns+` FROM service_appointments a JOIN businesses b ON b.id=a.business_id WHERE a.id=$1`, appointmentID))
	if err != nil {
		return domain.Appointment{}, err
	}
	appointment.Offerings, err = appointmentOfferings(ctx, r.pool, appointment.ID)
	if err != nil {
		return domain.Appointment{}, err
	}
	return appointment, nil
}
func mapError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23P01" {
		return application.ErrConflict
	}
	return err
}
