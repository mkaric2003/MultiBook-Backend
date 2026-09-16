package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	"github.com/mkaric2003/multibook-backend/internal/modules/metrics/application"
)

type Queries struct{ pool *pgxpool.Pool }

func NewQueries(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

var _ application.Queries = (*Queries)(nil)

func (q *Queries) GetDashboardMetrics(ctx context.Context, ownerID string, businessID uuid.UUID, monthStart time.Time) (application.DashboardMetrics, error) {
	var (
		businessOwner string
		businessType  domain.BusinessType
		currency      string
	)
	err := q.pool.QueryRow(ctx, `SELECT owner_id,type::text,currency FROM businesses
		WHERE id=$1 AND deleted_at IS NULL AND status<>'archived'`, businessID).
		Scan(&businessOwner, &businessType, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.DashboardMetrics{}, application.ErrBusinessNotFound
	}
	if err != nil {
		return application.DashboardMetrics{}, fmt.Errorf("read dashboard business: %w", err)
	}
	if businessOwner != ownerID {
		return application.DashboardMetrics{}, application.ErrForbidden
	}

	table, err := reservationTable(businessType)
	if err != nil {
		return application.DashboardMetrics{}, err
	}
	metrics := application.DashboardMetrics{
		BusinessID:   businessID,
		BusinessType: businessType,
		Currency:     currency,
		CurrentMonth: application.DashboardMonth{
			MonthKey:              monthStart.UTC().Format("2006-01"),
			DailyRevenueMinor:     make(map[string]int64),
			DailyReservationCount: make(map[string]int),
		},
	}
	if err := q.pool.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE business_id=$1 AND status='confirmed'`, businessID).
		Scan(&metrics.ActiveReservationCount); err != nil {
		return application.DashboardMetrics{}, fmt.Errorf("count active dashboard reservations: %w", err)
	}

	monthEnd := monthStart.AddDate(0, 1, 0)
	rows, err := q.pool.Query(ctx, `SELECT to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD'),sum(total_minor),count(*)
		FROM `+table+`
		WHERE business_id=$1 AND created_at >= $2 AND created_at < $3
		AND status IN ('confirmed','completed')
		AND (lower(payment_method)='cash' OR payment_status='paid')
		GROUP BY 1 ORDER BY 1`, businessID, monthStart, monthEnd)
	if err != nil {
		return application.DashboardMetrics{}, fmt.Errorf("read daily dashboard metrics: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var day string
		var revenue int64
		var count int
		if err := rows.Scan(&day, &revenue, &count); err != nil {
			return application.DashboardMetrics{}, fmt.Errorf("scan daily dashboard metrics: %w", err)
		}
		metrics.CurrentMonth.RevenueMinor += revenue
		metrics.CurrentMonth.ReservationCount += count
		metrics.CurrentMonth.DailyRevenueMinor[day] = revenue
		metrics.CurrentMonth.DailyReservationCount[day] = count
	}
	if err := rows.Err(); err != nil {
		return application.DashboardMetrics{}, fmt.Errorf("iterate daily dashboard metrics: %w", err)
	}
	return metrics, nil
}

func (q *Queries) GetEarningsMetrics(ctx context.Context, ownerID string, businessID uuid.UUID, filter application.EarningsFilter) (application.EarningsMetrics, error) {
	var (
		businessOwner string
		businessType  domain.BusinessType
		currency      string
	)
	err := q.pool.QueryRow(ctx, `SELECT owner_id,type::text,currency FROM businesses
		WHERE id=$1 AND deleted_at IS NULL AND status<>'archived'`, businessID).
		Scan(&businessOwner, &businessType, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.EarningsMetrics{}, application.ErrBusinessNotFound
	}
	if err != nil {
		return application.EarningsMetrics{}, fmt.Errorf("read earnings business: %w", err)
	}
	if businessOwner != ownerID {
		return application.EarningsMetrics{}, application.ErrForbidden
	}
	if filter.StaffID != nil {
		if businessType != domain.BusinessTypeService {
			return application.EarningsMetrics{}, application.ErrStaffNotFound
		}
		var exists bool
		if err := q.pool.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM service_staff WHERE id=$1 AND business_id=$2
		)`, *filter.StaffID, businessID).Scan(&exists); err != nil {
			return application.EarningsMetrics{}, fmt.Errorf("validate earnings staff: %w", err)
		}
		if !exists {
			return application.EarningsMetrics{}, application.ErrStaffNotFound
		}
	}

	table, err := reservationTable(businessType)
	if err != nil {
		return application.EarningsMetrics{}, err
	}
	metrics := application.EarningsMetrics{
		BusinessID:              businessID,
		BusinessType:            businessType,
		Currency:                currency,
		StartDate:               filter.StartDate.Format(time.DateOnly),
		EndDate:                 filter.EndDate.Format(time.DateOnly),
		StaffID:                 filter.StaffID,
		DailyRevenueMinor:       make(map[string]int64),
		DailyReservationCount:   make(map[string]int),
		DailyOnlineRevenueMinor: make(map[string]int64),
		DailyCashRevenueMinor:   make(map[string]int64),
		DailyStaffEarningsMinor: make(map[string]int64),
	}

	staffEarningsExpression := "0::bigint"
	staffPredicate := ""
	args := []any{businessID, filter.StartDate, filter.EndDate.AddDate(0, 0, 1), filter.UTCOffsetMinutes}
	if businessType == domain.BusinessTypeService && filter.StaffID != nil {
		staffEarningsExpression = "sum(provider_earnings_minor)"
		staffPredicate = " AND staff_id=$5"
		args = append(args, *filter.StaffID)
	}
	rows, err := q.pool.Query(ctx, `SELECT
		to_char(created_at AT TIME ZONE 'UTC' + $4 * interval '1 minute','YYYY-MM-DD'),
		sum(total_minor),
		count(*),
		coalesce(sum(total_minor) FILTER (WHERE lower(payment_method)<>'cash'),0),
		coalesce(sum(total_minor) FILTER (WHERE lower(payment_method)='cash'),0),
		`+staffEarningsExpression+`
		FROM `+table+`
		WHERE business_id=$1 AND created_at >= $2 AND created_at < $3
		AND status IN ('confirmed','completed')
		AND (lower(payment_method)='cash' OR payment_status='paid')`+staffPredicate+`
		GROUP BY 1 ORDER BY 1`, args...)
	if err != nil {
		return application.EarningsMetrics{}, fmt.Errorf("read earnings metrics: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			day           string
			revenue       int64
			count         int
			onlineRevenue int64
			cashRevenue   int64
			staffEarnings int64
		)
		if err := rows.Scan(&day, &revenue, &count, &onlineRevenue, &cashRevenue, &staffEarnings); err != nil {
			return application.EarningsMetrics{}, fmt.Errorf("scan earnings metrics: %w", err)
		}
		metrics.RevenueMinor += revenue
		metrics.ReservationCount += count
		metrics.OnlineRevenueMinor += onlineRevenue
		metrics.CashRevenueMinor += cashRevenue
		metrics.StaffEarningsMinor += staffEarnings
		metrics.DailyRevenueMinor[day] = revenue
		metrics.DailyReservationCount[day] = count
		metrics.DailyOnlineRevenueMinor[day] = onlineRevenue
		metrics.DailyCashRevenueMinor[day] = cashRevenue
		metrics.DailyStaffEarningsMinor[day] = staffEarnings
	}
	if err := rows.Err(); err != nil {
		return application.EarningsMetrics{}, fmt.Errorf("iterate earnings metrics: %w", err)
	}
	return metrics, nil
}

func reservationTable(businessType domain.BusinessType) (string, error) {
	switch businessType {
	case domain.BusinessTypeStay:
		return "stay_bookings", nil
	case domain.BusinessTypeService:
		return "service_appointments", nil
	default:
		return "", fmt.Errorf("unsupported business type %q", businessType)
	}
}
