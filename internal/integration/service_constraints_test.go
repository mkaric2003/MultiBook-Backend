//go:build integration

package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	availabilitypostgres "github.com/mkaric2003/multibook-backend/internal/modules/service_availability/adapters/postgres"
	availabilityapp "github.com/mkaric2003/multibook-backend/internal/modules/service_availability/application"
)

func TestServiceAvailabilityExclusionConstraints(t *testing.T) {
	databaseURL := os.Getenv("INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("INTEGRATION_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	defer connection.Close(ctx)

	tx, err := connection.Begin(ctx)
	if err != nil {
		t.Fatalf("begin fixture transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	userID := "integration-" + uuid.NewString()
	businessID := uuid.New()
	staffID := uuid.New()
	_, err = tx.Exec(ctx, `INSERT INTO users (id, email, role, business_currency) VALUES ($1, $2, 'provider', 'BAM')`, userID, userID+"@example.test")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO businesses (id, owner_id, type, name, name_normalized, category_id, currency) VALUES ($1, $2, 'service', 'Integration service', 'integration service', 'test', 'BAM')`, businessID, userID)
	if err != nil {
		t.Fatalf("create business: %v", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO service_details (business_id, time_zone) VALUES ($1, 'Europe/Sarajevo')`, businessID)
	if err != nil {
		t.Fatalf("create service details: %v", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO service_staff (id, business_id, name) VALUES ($1, $2, 'Integration staff')`, staffID, businessID)
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}

	_, err = tx.Exec(ctx, `INSERT INTO service_staff_weekly_availability (staff_id, weekday, start_minutes, end_minutes) VALUES ($1, 1, 480, 720)`, staffID)
	if err != nil {
		t.Fatalf("create weekly interval: %v", err)
	}
	if _, err = tx.Exec(ctx, "SAVEPOINT overlapping_weekly_interval"); err != nil {
		t.Fatalf("create weekly interval savepoint: %v", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO service_staff_weekly_availability (staff_id, weekday, start_minutes, end_minutes) VALUES ($1, 1, 600, 780)`, staffID)
	if err == nil {
		t.Fatal("expected overlapping weekly interval to violate exclusion constraint")
	}
	if _, err = tx.Exec(ctx, "ROLLBACK TO SAVEPOINT overlapping_weekly_interval"); err != nil {
		t.Fatalf("rollback weekly interval savepoint: %v", err)
	}

	_, err = tx.Exec(ctx, `INSERT INTO service_staff_availability_blocks (staff_id, blocked_range) VALUES ($1, tstzrange('2026-09-01 10:00:00+00', '2026-09-01 11:00:00+00', '[)'))`, staffID)
	if err != nil {
		t.Fatalf("create availability block: %v", err)
	}
	if _, err = tx.Exec(ctx, "SAVEPOINT overlapping_availability_block"); err != nil {
		t.Fatalf("create availability block savepoint: %v", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO service_staff_availability_blocks (staff_id, blocked_range) VALUES ($1, tstzrange('2026-09-01 10:30:00+00', '2026-09-01 11:30:00+00', '[)'))`, staffID)
	if err == nil {
		t.Fatal("expected overlapping availability block to violate exclusion constraint")
	}
	if _, err = tx.Exec(ctx, "ROLLBACK TO SAVEPOINT overlapping_availability_block"); err != nil {
		t.Fatalf("rollback availability block savepoint: %v", err)
	}
}

func TestServiceAvailabilityRepositoryEnforcesOwnership(t *testing.T) {
	databaseURL := os.Getenv("INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("INTEGRATION_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	ownerID := "integration-owner-" + uuid.NewString()
	otherID := "integration-other-" + uuid.NewString()
	businessID, staffID := uuid.New(), uuid.New()
	for _, userID := range []string{ownerID, otherID} {
		if _, err = pool.Exec(ctx, `INSERT INTO users (id, email, role, business_currency) VALUES ($1, $2, 'provider', 'BAM')`, userID, userID+"@example.test"); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM businesses WHERE id = $1`, businessID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = ANY($1)`, []string{ownerID, otherID})
	}()
	if _, err = pool.Exec(ctx, `INSERT INTO businesses (id, owner_id, type, name, name_normalized, category_id, currency) VALUES ($1, $2, 'service', 'Ownership service', 'ownership service', 'test', 'BAM')`, businessID, ownerID); err != nil {
		t.Fatalf("create business: %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO service_staff (id, business_id, name) VALUES ($1, $2, 'Ownership staff')`, staffID, businessID); err != nil {
		t.Fatalf("create staff: %v", err)
	}

	_, err = availabilitypostgres.NewQueries(pool).ListWeeklyOwned(ctx, otherID, businessID, staffID)
	if !errors.Is(err, availabilityapp.ErrNotFound) {
		t.Fatalf("expected not found for non-owner, got %v", err)
	}
}

func TestConfirmedAppointmentsCannotOverlap(t *testing.T) {
	databaseURL := os.Getenv("INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("INTEGRATION_DATABASE_URL is not set")
	}
	ctx := context.Background()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(ctx)
	tx, err := connection.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	owner, customer := "integration-owner-"+uuid.NewString(), "integration-customer-"+uuid.NewString()
	businessID, staffID := uuid.New(), uuid.New()
	for _, entry := range []struct{ id, role string }{{owner, "provider"}, {customer, "customer"}} {
		if _, err = tx.Exec(ctx, `INSERT INTO users(id,email,role,business_currency)VALUES($1,$2,$3,'BAM')`, entry.id, entry.id+"@example.test", entry.role); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO businesses(id,owner_id,type,name,name_normalized,category_id,currency)VALUES($1,$2,'service','Appointment test','appointment test','test','BAM')`, businessID, owner); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO service_staff(id,business_id,name)VALUES($1,$2,'Staff')`, staffID, businessID); err != nil {
		t.Fatal(err)
	}
	args := []any{businessID, customer, staffID, "Customer", "customer@example.test", "+38761123456", "Appointment test", "Staff", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), int16(600), int16(660), int64(1000), int64(85), int64(109), int64(1194), "BAM", "AP-test"}
	query := `INSERT INTO service_appointments(business_id,customer_id,staff_id,customer_name,customer_email,customer_phone,business_name,provider_name,appointment_date,start_minutes,end_minutes,scheduled_range,service_cost_minor,original_service_cost_minor,service_fee_minor,taxes_minor,total_minor,provider_commission_rate,provider_earnings_minor,currency,payment_status,payment_method,confirmation_code)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,tstzrange('2026-09-01 10:00:00+00','2026-09-01 11:00:00+00','[)'),$12,$12,$13,$14,$15,100,$12,$16,'pending','cash',$17)`
	if _, err = tx.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, "SAVEPOINT overlapping_appointment"); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, query, args...); err == nil {
		t.Fatal("expected overlapping appointment constraint violation")
	}
}

func TestHistoricalReferencesPreventPhysicalDeletion(t *testing.T) {
	databaseURL := os.Getenv("INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("INTEGRATION_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	defer connection.Close(ctx)

	tx, err := connection.Begin(ctx)
	if err != nil {
		t.Fatalf("begin fixture transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	ownerID, customerID := "integration-owner-"+uuid.NewString(), "integration-customer-"+uuid.NewString()
	serviceBusinessID, stayBusinessID := uuid.New(), uuid.New()
	staffID, offeringID, appointmentID := uuid.New(), uuid.New(), uuid.New()
	unitTypeID, unitID := uuid.New(), uuid.New()
	for _, user := range []struct{ id, role string }{{ownerID, "provider"}, {customerID, "customer"}} {
		if _, err := tx.Exec(ctx, `INSERT INTO users (id, email, role, business_currency) VALUES ($1, $2, $3, 'BAM')`, user.id, user.id+"@example.test", user.role); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO businesses (id, owner_id, type, name, name_normalized, category_id, currency) VALUES ($1, $2, 'service', 'Reference service', 'reference service', 'test', 'BAM')`, serviceBusinessID, ownerID); err != nil {
		t.Fatalf("create service business: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO service_staff (id, business_id, name) VALUES ($1, $2, 'Reference staff')`, staffID, serviceBusinessID); err != nil {
		t.Fatalf("create staff: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO service_offerings (id, business_id, name, duration_minutes, price_minor) VALUES ($1, $2, 'Reference offering', 30, 1000)`, offeringID, serviceBusinessID); err != nil {
		t.Fatalf("create offering: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO service_appointments (id, business_id, customer_id, staff_id, customer_name, customer_email, customer_phone, business_name, provider_name, appointment_date, start_minutes, end_minutes, scheduled_range, service_cost_minor, original_service_cost_minor, service_fee_minor, taxes_minor, total_minor, provider_commission_rate, provider_earnings_minor, currency, payment_status, payment_method, confirmation_code) VALUES ($1, $2, $3, $4, 'Customer', 'customer@example.test', '+38761123456', 'Reference service', 'Reference staff', '2026-09-02', 600, 660, tstzrange('2026-09-02 10:00:00+00', '2026-09-02 11:00:00+00', '[)'), 1000, 1000, 85, 109, 1194, 100, 1000, 'BAM', 'pending', 'cash', 'AP-reference')`, appointmentID, serviceBusinessID, customerID, staffID); err != nil {
		t.Fatalf("create appointment: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO service_appointment_offerings (appointment_id, offering_id, name, duration_minutes, price_minor) VALUES ($1, $2, 'Reference offering', 30, 1000)`, appointmentID, offeringID); err != nil {
		t.Fatalf("create appointment offering: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO businesses (id, owner_id, type, name, name_normalized, category_id, currency) VALUES ($1, $2, 'stay', 'Reference stay', 'reference stay', 'test', 'BAM')`, stayBusinessID, ownerID); err != nil {
		t.Fatalf("create stay business: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO stay_unit_types (id, business_id, name, max_guests, size_square_meters, price_per_night_minor, quantity) VALUES ($1, $2, 'Reference unit type', 2, 20, 12000, 1)`, unitTypeID, stayBusinessID); err != nil {
		t.Fatalf("create stay unit type: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO stay_units (id, business_id, stay_unit_type_id, sequence_number) VALUES ($1, $2, $3, 1)`, unitID, stayBusinessID, unitTypeID); err != nil {
		t.Fatalf("create physical stay unit: %v", err)
	}

	assertForeignKeyViolation(t, ctx, tx, "delete_referenced_staff", `DELETE FROM service_staff WHERE id = $1`, staffID)
	assertForeignKeyViolation(t, ctx, tx, "delete_referenced_offering", `DELETE FROM service_offerings WHERE id = $1`, offeringID)
	assertForeignKeyViolation(t, ctx, tx, "delete_referenced_unit_type", `DELETE FROM stay_unit_types WHERE id = $1`, unitTypeID)

	if _, err := tx.Exec(ctx, `UPDATE service_staff SET is_active = FALSE, deleted_at = now() WHERE id = $1`, staffID); err != nil {
		t.Fatalf("archive staff: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE service_offerings SET is_active = FALSE, deleted_at = now() WHERE id = $1`, offeringID); err != nil {
		t.Fatalf("archive offering: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE stay_unit_types SET is_active = FALSE, deleted_at = now() WHERE id = $1`, unitTypeID); err != nil {
		t.Fatalf("archive stay unit type: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE stay_units SET is_active = FALSE, deleted_at = now() WHERE id = $1`, unitID); err != nil {
		t.Fatalf("archive physical stay unit: %v", err)
	}
	var appointmentReferences, unitReferences int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM service_appointment_offerings WHERE appointment_id = $1 AND offering_id = $2`, appointmentID, offeringID).Scan(&appointmentReferences); err != nil {
		t.Fatalf("count appointment offering references: %v", err)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM stay_units WHERE id = $1 AND stay_unit_type_id = $2`, unitID, unitTypeID).Scan(&unitReferences); err != nil {
		t.Fatalf("count physical unit references: %v", err)
	}
	if appointmentReferences != 1 || unitReferences != 1 {
		t.Fatalf("archiving must preserve historical references: appointment=%d unit=%d", appointmentReferences, unitReferences)
	}
}

func assertForeignKeyViolation(t *testing.T, ctx context.Context, tx pgx.Tx, savepoint, query string, args ...any) {
	t.Helper()
	if _, err := tx.Exec(ctx, "SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("create savepoint %s: %v", savepoint, err)
	}
	_, err := tx.Exec(ctx, query, args...)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("expected foreign-key violation for %s, got %v", savepoint, err)
	}
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("rollback savepoint %s: %v", savepoint, err)
	}
}
