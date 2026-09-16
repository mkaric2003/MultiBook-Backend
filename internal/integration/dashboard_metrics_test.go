//go:build integration

package integration

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	metricspostgres "github.com/mkaric2003/multibook-backend/internal/modules/metrics/adapters/postgres"
	metricsapp "github.com/mkaric2003/multibook-backend/internal/modules/metrics/application"
)

func TestDashboardMetricsQueryAndLiveInvalidation(t *testing.T) {
	databaseURL := os.Getenv("INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("INTEGRATION_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	owner, customer := "metrics-owner-"+uuid.NewString(), "metrics-customer-"+uuid.NewString()
	businessID := uuid.New()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM stay_bookings WHERE business_id=$1`, businessID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM businesses WHERE id=$1`, businessID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id=ANY($1)`, []string{owner, customer})
	}()
	exec(`INSERT INTO users(id,email,role,display_name,business_currency) VALUES
		($1,$1||'@example.test','provider','Metrics owner','BAM'),
		($2,$2||'@example.test','customer','Metrics customer','BAM')`, owner, customer)
	exec(`INSERT INTO businesses(id,owner_id,type,status,name,name_normalized,category_id,currency)
		VALUES($1,$2,'stay','active','Metrics stay','metrics stay','hotel','BAM')`, businessID, owner)

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	currentDay := monthStart.Add(24 * time.Hour)
	insertBooking := func(status, paymentStatus, paymentMethod string, total int64, createdAt time.Time) uuid.UUID {
		id := uuid.New()
		exec(`INSERT INTO stay_bookings(id,business_id,business_owner_id,customer_id,customer_name,customer_email,
			check_in,check_out,adults,price_per_night_minor,room_subtotal_minor,total_minor,original_total_minor,
			payment_status,payment_method,confirmation_code,currency,status,created_at)
			VALUES($1,$2,$3,$4,'Customer',$4||'@example.test','2099-01-01','2099-01-02',1,$5,$5,$5,$5,
			$6::stay_payment_status,$7,$8,'BAM',$9::stay_booking_status,$10)`,
			id, businessID, owner, customer, total, paymentStatus, paymentMethod, uuid.NewString(), status, createdAt)
		return id
	}
	insertBooking("confirmed", "pending", "cash", 1000, currentDay)
	insertBooking("completed", "paid", "card", 2000, currentDay)
	unpaidBooking := insertBooking("confirmed", "pending", "card", 3000, currentDay)
	insertBooking("cancelled", "paid", "cash", 4000, currentDay)
	insertBooking("completed", "paid", "card", 5000, monthStart.Add(-time.Hour))

	queries := metricspostgres.NewQueries(pool)
	metrics, err := queries.GetDashboardMetrics(ctx, owner, businessID, monthStart)
	if err != nil {
		t.Fatal(err)
	}
	dayKey := currentDay.Format("2006-01-02")
	if metrics.ActiveReservationCount != 2 || metrics.CurrentMonth.RevenueMinor != 3000 || metrics.CurrentMonth.ReservationCount != 2 {
		t.Fatalf("metrics counts = active:%d revenue:%d reservations:%d", metrics.ActiveReservationCount, metrics.CurrentMonth.RevenueMinor, metrics.CurrentMonth.ReservationCount)
	}
	if metrics.CurrentMonth.DailyRevenueMinor[dayKey] != 3000 || metrics.CurrentMonth.DailyReservationCount[dayKey] != 2 {
		t.Fatalf("daily metrics = %#v / %#v", metrics.CurrentMonth.DailyRevenueMinor, metrics.CurrentMonth.DailyReservationCount)
	}
	earnings, err := queries.GetEarningsMetrics(ctx, owner, businessID, metricsapp.EarningsFilter{
		StartDate: currentDay,
		EndDate:   currentDay,
	})
	if err != nil {
		t.Fatal(err)
	}
	if earnings.RevenueMinor != 3000 || earnings.OnlineRevenueMinor != 2000 || earnings.CashRevenueMinor != 1000 || earnings.ReservationCount != 2 {
		t.Fatalf("earnings = %#v", earnings)
	}

	listenerContext, stopListener := context.WithCancel(ctx)
	defer stopListener()
	listener := metricspostgres.NewMetricsUpdateListener(databaseURL, slog.New(slog.NewTextHandler(io.Discard, nil)))
	updates, unsubscribe := listener.SubscribeChanges(businessID)
	defer unsubscribe()
	listener.Run(listenerContext)
	select {
	case <-updates: // The listener publishes a resync signal after LISTEN succeeds.
	case <-time.After(5 * time.Second):
		t.Fatal("metrics listener did not connect")
	}
	exec(`UPDATE stay_bookings SET payment_status='paid' WHERE id=$1`, unpaidBooking)
	select {
	case <-updates:
	case <-time.After(5 * time.Second):
		t.Fatal("committed reservation change did not invalidate dashboard metrics")
	}

	metrics, err = queries.GetDashboardMetrics(ctx, owner, businessID, monthStart)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.CurrentMonth.RevenueMinor != 6000 || metrics.CurrentMonth.ReservationCount != 3 {
		t.Fatalf("live metrics revenue/count = %d/%d", metrics.CurrentMonth.RevenueMinor, metrics.CurrentMonth.ReservationCount)
	}
	earnings, err = queries.GetEarningsMetrics(ctx, owner, businessID, metricsapp.EarningsFilter{
		StartDate: currentDay,
		EndDate:   currentDay,
	})
	if err != nil {
		t.Fatal(err)
	}
	if earnings.RevenueMinor != 6000 || earnings.OnlineRevenueMinor != 5000 || earnings.CashRevenueMinor != 1000 || earnings.ReservationCount != 3 {
		t.Fatalf("updated earnings = %#v", earnings)
	}
}
