//go:build integration

package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	reviewspostgres "github.com/mkaric2003/multibook-backend/internal/modules/reviews/adapters/postgres"
	reviewsapp "github.com/mkaric2003/multibook-backend/internal/modules/reviews/application"
	"github.com/mkaric2003/multibook-backend/internal/modules/reviews/domain"
)

func TestReviewsPersistenceAndBusinessAggregate(t *testing.T) {
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

	owner := "review-owner-" + uuid.NewString()
	customer := "review-customer-" + uuid.NewString()
	otherCustomer := "review-other-" + uuid.NewString()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=ANY($1)`, []string{owner, customer, otherCustomer})
	}()
	exec(`INSERT INTO users(id,email,role,display_name,avatar_storage_path,business_currency) VALUES
		($1,$1||'@example.test','provider','Owner',NULL,'BAM'),
		($2,$2||'@example.test','customer','Current name','profiles/current.webp','BAM'),
		($3,$3||'@example.test','customer','Other',NULL,'BAM')`, owner, customer, otherCustomer)

	stayBusiness, serviceBusiness := uuid.New(), uuid.New()
	exec(`INSERT INTO businesses(id,owner_id,type,status,name,name_normalized,category_id,currency,average_rating,review_count) VALUES
		($1,$3,'stay','active','Review stay','review stay','hotel','BAM',4,2),
		($2,$3,'service','active','Review service','review service','salon','BAM',0,0)`, stayBusiness, serviceBusiness, owner)
	exec(`INSERT INTO stay_details(business_id,base_price_minor,inventory_type) VALUES($1,10000,'single_unit')`, stayBusiness)
	exec(`INSERT INTO service_details(business_id,time_zone) VALUES($1,'Europe/Sarajevo')`, serviceBusiness)
	staffID := uuid.New()
	exec(`INSERT INTO service_staff(id,business_id,name) VALUES($1,$2,'Staff')`, staffID, serviceBusiness)

	finishedStay, unfinishedStay := uuid.New(), uuid.New()
	insertStay := func(id uuid.UUID, customerID string, checkIn, checkOut string) {
		exec(`INSERT INTO stay_bookings(id,business_id,business_owner_id,customer_id,customer_name,customer_email,
			check_in,check_out,adults,price_per_night_minor,room_subtotal_minor,total_minor,original_total_minor,
			payment_status,payment_method,confirmation_code,currency)
			VALUES($1,$2,$3,$4,'Booking snapshot',$4||'@example.test',$5,$6,1,10000,10000,10000,10000,
			'paid','card',$7,'BAM')`, id, stayBusiness, owner, customerID, checkIn, checkOut, uuid.NewString())
	}
	insertStay(finishedStay, customer, "2025-01-01", "2025-01-02")
	insertStay(unfinishedStay, otherCustomer, "2099-01-01", "2099-01-02")

	commands := reviewspostgres.NewCommandRepository(pool)
	queries := reviewspostgres.NewQueries(pool)
	service := reviewsapp.NewService(commands, queries)
	actor := reviewsapp.Actor{ID: customer, Role: "customer"}
	comment := "  Excellent stay  "
	review, err := service.Create(ctx, actor, stayBusiness, domain.CreateInput{
		SourceID: finishedStay, Type: domain.SourceTypeStay, Rating: 2, Comment: &comment,
	})
	if err != nil {
		t.Fatal(err)
	}
	if review.CustomerName != "Booking snapshot" || review.CustomerAvatarPath == nil || *review.CustomerAvatarPath != "profiles/current.webp" || review.Comment == nil || *review.Comment != "Excellent stay" {
		t.Fatalf("review snapshot = %#v", review)
	}
	var average float64
	var count int
	if err := pool.QueryRow(ctx, `SELECT average_rating,review_count FROM businesses WHERE id=$1`, stayBusiness).Scan(&average, &count); err != nil {
		t.Fatal(err)
	}
	if average != 3.33 || count != 3 {
		t.Fatalf("aggregate = %.2f/%d, want 3.33/3", average, count)
	}
	has, err := queries.HasReview(ctx, customer, stayBusiness)
	if err != nil || !has {
		t.Fatalf("HasReview() = %v, %v", has, err)
	}
	items, err := queries.List(ctx, stayBusiness, 4, 0)
	if err != nil || len(items) != 1 || items[0].ID != review.ID {
		t.Fatalf("List() = %#v, %v", items, err)
	}
	_, err = service.Create(ctx, actor, stayBusiness, domain.CreateInput{SourceID: finishedStay, Type: domain.SourceTypeStay, Rating: 5})
	if !errors.Is(err, reviewsapp.ErrAlreadyReviewed) {
		t.Fatalf("duplicate error = %v", err)
	}

	_, err = service.Create(ctx, reviewsapp.Actor{ID: otherCustomer, Role: "customer"}, stayBusiness, domain.CreateInput{
		SourceID: unfinishedStay, Type: domain.SourceTypeStay, Rating: 5,
	})
	if !errors.Is(err, reviewsapp.ErrSourceNotFinished) {
		t.Fatalf("unfinished error = %v", err)
	}
	_, err = service.Create(ctx, reviewsapp.Actor{ID: otherCustomer, Role: "customer"}, stayBusiness, domain.CreateInput{
		SourceID: finishedStay, Type: domain.SourceTypeStay, Rating: 5,
	})
	if !errors.Is(err, reviewsapp.ErrForbidden) {
		t.Fatalf("foreign source error = %v", err)
	}

	appointmentID := uuid.New()
	exec(`INSERT INTO service_appointments(id,business_id,customer_id,staff_id,customer_name,customer_email,customer_phone,
		business_name,provider_name,appointment_date,start_minutes,end_minutes,scheduled_range,service_cost_minor,
		original_service_cost_minor,service_fee_minor,taxes_minor,total_minor,provider_commission_rate,
		provider_earnings_minor,currency,payment_status,payment_method,confirmation_code,status)
		VALUES($1,$2,$3,$4,'Service snapshot',$3||'@example.test','+38761111111','Review service','Staff',
		'2025-01-01',600,630,tstzrange('2025-01-01 10:00:00+01','2025-01-01 10:30:00+01','[)'),
		2500,2500,0,0,2500,100,2500,'BAM','paid','card',$5,'completed')`, appointmentID, serviceBusiness, otherCustomer, staffID, uuid.NewString())
	serviceReview, err := service.Create(ctx, reviewsapp.Actor{ID: otherCustomer, Role: "customer"}, serviceBusiness, domain.CreateInput{
		SourceID: appointmentID, Type: domain.SourceTypeService, Rating: 5,
	})
	if err != nil || serviceReview.SourceType != domain.SourceTypeService || serviceReview.BusinessOwnerID != owner {
		t.Fatalf("service review = %#v, %v", serviceReview, err)
	}
}
