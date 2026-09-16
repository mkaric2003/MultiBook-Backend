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
	discoverypostgres "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/adapters/postgres"
	savedpostgres "github.com/mkaric2003/multibook-backend/internal/modules/saved_businesses/adapters/postgres"
	savedapp "github.com/mkaric2003/multibook-backend/internal/modules/saved_businesses/application"
)

func TestSavedBusinessesPersistence(t *testing.T) {
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
	owner, a, b := "saved-owner-"+uuid.NewString(), "saved-a-"+uuid.NewString(), "saved-b-"+uuid.NewString()
	id := uuid.New()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=ANY($1)`, []string{owner, a, b})
	}()
	for _, user := range []string{owner, a, b} {
		role := "customer"
		if user == owner {
			role = "provider"
		}
		exec(`INSERT INTO users(id,email,role,business_currency) VALUES($1,$2,$3,'BAM')`, user, user+"@example.test", role)
	}
	exec(`INSERT INTO businesses(id,owner_id,type,status,name,name_normalized,category_id,currency) VALUES($1,$2,'stay','active','Saved test','saved test','hotel','BAM')`, id, owner)
	exec(`INSERT INTO business_locations(business_id,city,city_normalized,address,country_code,coordinates,is_primary) VALUES($1,'Sarajevo','sarajevo','Test','BA',ST_SetSRID(ST_MakePoint(18.42,43.86),4326)::geography,true)`, id)
	exec(`INSERT INTO stay_details(business_id,base_price_minor,inventory_type) VALUES($1,12000,'single_unit')`, id)
	commands := savedpostgres.NewCommandRepository(pool)
	queries := savedpostgres.NewQueries(pool)
	service := savedapp.NewService(commands, queries, discoverypostgres.NewQueries(pool))
	actor := func(user string) savedapp.Actor { return savedapp.Actor{ID: user, Role: "customer"} }
	check := func(user string, want bool) {
		t.Helper()
		got, err := queries.IsSaved(ctx, user, id)
		if err != nil || got != want {
			t.Fatalf("saved %s = %v, %v", user, got, err)
		}
	}
	check(a, false)
	if err := commands.Save(ctx, a, id); err != nil {
		t.Fatal(err)
	}
	var before, after time.Time
	if err := pool.QueryRow(ctx, `SELECT saved_at FROM saved_businesses WHERE customer_id=$1 AND business_id=$2`, a, id).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err := commands.Save(ctx, a, id); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT saved_at FROM saved_businesses WHERE customer_id=$1 AND business_id=$2`, a, id).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if !before.Equal(after) {
		t.Fatal("idempotent save changed ordering")
	}
	check(a, true)
	check(b, false)
	if err := commands.Remove(ctx, b, id); err != nil {
		t.Fatal(err)
	}
	check(a, true)
	items, err := service.List(ctx, actor(b))
	if err != nil || len(items) != 0 {
		t.Fatalf("other customer list: %v %v", items, err)
	}
	exec(`UPDATE businesses SET name='Updated name' WHERE id=$1`, id)
	items, err = service.List(ctx, actor(a))
	if err != nil || len(items) != 1 || items[0].Name != "Updated name" || items[0].Stay == nil {
		t.Fatalf("current read model: %v %v", items, err)
	}
	for _, status := range []string{"inactive", "archived"} {
		exec(`UPDATE businesses SET status=$2 WHERE id=$1`, id, status)
		check(a, false)
		items, err = service.List(ctx, actor(a))
		if err != nil || len(items) != 0 {
			t.Fatalf("unavailable list: %v %v", items, err)
		}
		if err := commands.Save(ctx, a, id); !errors.Is(err, savedapp.ErrBusinessNotFound) {
			t.Fatalf("save unavailable: %v", err)
		}
	}
	exec(`UPDATE businesses SET status='active',deleted_at=now() WHERE id=$1`, id)
	check(a, false)
	if err := commands.Save(ctx, a, id); !errors.Is(err, savedapp.ErrBusinessNotFound) {
		t.Fatalf("save deleted: %v", err)
	}
	if err := commands.Save(ctx, a, uuid.New()); !errors.Is(err, savedapp.ErrBusinessNotFound) {
		t.Fatalf("save missing: %v", err)
	}
	for range 2 {
		if err := commands.Remove(ctx, a, id); err != nil {
			t.Fatal(err)
		}
	}
	check(a, false)
	exec(`UPDATE businesses SET deleted_at=NULL WHERE id=$1`, id)
	if err := commands.Save(ctx, a, id); err != nil {
		t.Fatal(err)
	}
	exec(`DELETE FROM businesses WHERE id=$1`, id)
	check(a, false)
}
