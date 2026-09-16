package application

import (
	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
)

// DashboardMetrics is the complete provider-dashboard metrics projection.
// Earnings-specific payment and staff breakdowns deliberately do not belong
// to this read model.
type DashboardMetrics struct {
	BusinessID             uuid.UUID
	BusinessType           domain.BusinessType
	Currency               string
	ActiveReservationCount int
	CurrentMonth           DashboardMonth
}

type DashboardMonth struct {
	MonthKey              string
	RevenueMinor          int64
	ReservationCount      int
	DailyRevenueMinor     map[string]int64
	DailyReservationCount map[string]int
}

type EarningsMetrics struct {
	BusinessID              uuid.UUID
	BusinessType            domain.BusinessType
	Currency                string
	StartDate               string
	EndDate                 string
	StaffID                 *uuid.UUID
	RevenueMinor            int64
	ReservationCount        int
	OnlineRevenueMinor      int64
	CashRevenueMinor        int64
	StaffEarningsMinor      int64
	DailyRevenueMinor       map[string]int64
	DailyReservationCount   map[string]int
	DailyOnlineRevenueMinor map[string]int64
	DailyCashRevenueMinor   map[string]int64
	DailyStaffEarningsMinor map[string]int64
}
