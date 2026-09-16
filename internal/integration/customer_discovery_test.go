//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/modules/businesses/domain"
	discoverypostgres "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/adapters/postgres"
)

func TestCustomerDiscoveryPersistence(t *testing.T) {
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

	owner := "discovery-owner-" + uuid.NewString()
	city := "Discovery " + uuid.NewString()
	normalizedCity := strings.ToLower(city)
	firstID, secondID := uuid.New(), uuid.New()
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, owner)
		_, _ = pool.Exec(context.Background(), `DELETE FROM cities WHERE name=$1`, city)
	}()
	if _, err := pool.Exec(ctx, `INSERT INTO users(id,email,role,business_currency) VALUES($1,$2,'provider','BAM')`, owner, owner+"@example.test"); err != nil {
		t.Fatal(err)
	}
	insertBusiness := func(id uuid.UUID, name string, rating float64) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO businesses(id,owner_id,type,status,name,name_normalized,category_id,currency,average_rating)
			VALUES($1,$2,'stay','active',$3,lower($3),'hotel','BAM',$4)`, id, owner, name, rating); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO business_locations(business_id,city,city_normalized,address,country_code,coordinates,is_primary)
			VALUES($1,$2,$3,'Test','BA',ST_SetSRID(ST_MakePoint(18.42,43.86),4326)::geography,true)`, id, city, normalizedCity); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO stay_details(business_id,base_price_minor,inventory_type) VALUES($1,12000,'single_unit')`, id); err != nil {
			t.Fatal(err)
		}
	}
	insertBusiness(firstID, "Discovery lower", 3)
	insertBusiness(secondID, "Discovery higher", 5)

	queries := discoverypostgres.NewQueries(pool)
	popular, err := queries.ListPopular(ctx, domain.BusinessTypeStay, normalizedCity, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(popular) != 2 || popular[0].ID != secondID || popular[0].Stay == nil {
		t.Fatalf("unexpected popular summaries: %#v", popular)
	}

	ordered, err := queries.ListActiveByIDs(ctx, []uuid.UUID{firstID, secondID})
	if err != nil {
		t.Fatal(err)
	}
	if len(ordered) != 2 || ordered[0].ID != firstID || ordered[1].ID != secondID {
		t.Fatalf("input order was not preserved: %#v", ordered)
	}

	detail, err := queries.GetActiveDetail(ctx, firstID)
	if err != nil || detail.ID != firstID || detail.Stay == nil {
		t.Fatalf("unexpected active detail: %#v, %v", detail, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE businesses SET status='inactive' WHERE id=$1`, firstID); err != nil {
		t.Fatal(err)
	}
	ordered, err = queries.ListActiveByIDs(ctx, []uuid.UUID{firstID, secondID})
	if err != nil || len(ordered) != 1 || ordered[0].ID != secondID {
		t.Fatalf("inactive business was not omitted: %#v, %v", ordered, err)
	}
}
