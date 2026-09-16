//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	discoverypostgres "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/adapters/postgres"
	recentpostgres "github.com/mkaric2003/multibook-backend/internal/modules/recently_viewed/adapters/postgres"
	recentapp "github.com/mkaric2003/multibook-backend/internal/modules/recently_viewed/application"
)

func TestRecentlyViewedPersistence(t *testing.T) {
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

	owner := "recent-owner-" + uuid.NewString()
	firstCustomer := "recent-a-" + uuid.NewString()
	secondCustomer := "recent-b-" + uuid.NewString()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=ANY($1)`, []string{owner, firstCustomer, secondCustomer})
	}()
	for _, user := range []string{owner, firstCustomer, secondCustomer} {
		role := "customer"
		if user == owner {
			role = "provider"
		}
		exec(`INSERT INTO users(id,email,role,business_currency) VALUES($1,$2,$3,'BAM')`, user, user+"@example.test", role)
	}

	insertBusiness := func(name string, businessType domain.BusinessType, status domain.BusinessStatus, withProjection bool) uuid.UUID {
		t.Helper()
		id := uuid.New()
		exec(`INSERT INTO businesses(id,owner_id,type,status,name,name_normalized,category_id,currency)
			VALUES($1,$2,$3,$4,$5,$6,'test','BAM')`, id, owner, businessType, status, name, name)
		if !withProjection {
			return id
		}
		exec(`INSERT INTO business_locations(business_id,city,city_normalized,address,country_code,coordinates,is_primary)
			VALUES($1,'Sarajevo','sarajevo','Test','BA',ST_SetSRID(ST_MakePoint(18.42,43.86),4326)::geography,true)`, id)
		if businessType == domain.BusinessTypeStay {
			exec(`INSERT INTO stay_details(business_id,base_price_minor,inventory_type) VALUES($1,12000,'single_unit')`, id)
		} else {
			exec(`INSERT INTO service_offerings(business_id,name,duration_minutes,price_minor,is_active) VALUES($1,'Test',30,2500,true)`, id)
		}
		return id
	}

	firstStay := insertBusiness("First stay", domain.BusinessTypeStay, domain.BusinessStatusActive, true)
	secondStay := insertBusiness("Second stay", domain.BusinessTypeStay, domain.BusinessStatusActive, true)
	serviceBusiness := insertBusiness("Service", domain.BusinessTypeService, domain.BusinessStatusActive, true)
	inactiveBusiness := insertBusiness("Inactive", domain.BusinessTypeStay, domain.BusinessStatusInactive, false)

	commands := recentpostgres.NewCommandRepository(pool)
	queries := recentpostgres.NewQueries(pool)
	service := recentapp.NewService(commands, queries, discoverypostgres.NewQueries(pool))
	actor := func(customerID string) recentapp.Actor { return recentapp.Actor{ID: customerID, Role: "customer"} }

	for _, businessID := range []uuid.UUID{uuid.New(), inactiveBusiness} {
		if err := commands.Record(ctx, firstCustomer, businessID); !errors.Is(err, recentapp.ErrBusinessNotFound) {
			t.Fatalf("record unavailable business %s: %v", businessID, err)
		}
	}
	if err := commands.Record(ctx, firstCustomer, firstStay); err != nil {
		t.Fatal(err)
	}
	exec(`UPDATE recently_viewed_businesses SET viewed_at=now()-interval '2 hours' WHERE customer_id=$1 AND business_id=$2`, firstCustomer, firstStay)
	for _, businessID := range []uuid.UUID{serviceBusiness, secondStay} {
		if err := commands.Record(ctx, firstCustomer, businessID); err != nil {
			t.Fatal(err)
		}
	}

	items, err := service.List(ctx, actor(secondCustomer), domain.BusinessTypeStay, 10)
	if err != nil || len(items) != 0 {
		t.Fatalf("other customer list: %v %v", items, err)
	}
	items, err = service.List(ctx, actor(firstCustomer), domain.BusinessTypeStay, 10)
	if err != nil || len(items) != 2 || items[0].ID != secondStay || items[1].ID != firstStay {
		t.Fatalf("ordered stay list: %v %v", items, err)
	}
	items, err = service.List(ctx, actor(firstCustomer), domain.BusinessTypeService, 10)
	if err != nil || len(items) != 1 || items[0].ID != serviceBusiness || items[0].Service == nil {
		t.Fatalf("service type list: %v %v", items, err)
	}
	exec(`UPDATE businesses SET name='Current name' WHERE id=$1`, firstStay)
	items, err = service.List(ctx, actor(firstCustomer), domain.BusinessTypeStay, 1)
	if err != nil || len(items) != 1 || items[0].ID != secondStay {
		t.Fatalf("limited list: %v %v", items, err)
	}
	exec(`UPDATE businesses SET status='inactive' WHERE id=$1`, secondStay)
	items, err = service.List(ctx, actor(firstCustomer), domain.BusinessTypeStay, 10)
	if err != nil || len(items) != 1 || items[0].ID != firstStay || items[0].Name != "Current name" {
		t.Fatalf("current active projection: %v %v", items, err)
	}

	for index := range 31 {
		businessID := insertBusiness(fmt.Sprintf("Retention %02d", index), domain.BusinessTypeService, domain.BusinessStatusActive, false)
		if err := commands.Record(ctx, secondCustomer, businessID); err != nil {
			t.Fatal(err)
		}
	}
	var relationCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM recently_viewed_businesses WHERE customer_id=$1`, secondCustomer).Scan(&relationCount); err != nil {
		t.Fatal(err)
	}
	if relationCount != 30 {
		t.Fatalf("retained %d relations, want 30", relationCount)
	}
}
