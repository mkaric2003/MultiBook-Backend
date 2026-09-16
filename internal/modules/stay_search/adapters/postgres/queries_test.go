package postgres

import (
	"strings"
	"testing"

	"github.com/mkaric2003/multibook-backend/internal/modules/stay_search/domain"
)

func TestBuildSearchQueryWithoutDatesTreatsNilListsAsEmpty(t *testing.T) {
	t.Parallel()
	query, args := buildSearchQuery(domain.Input{Adults: 1, MinPriceMinor: 5000, MaxPriceMinor: 50000, PageSize: 8})

	if !strings.Contains(query, "COALESCE(cardinality($2::text[]), 0)=0") || !strings.Contains(query, "COALESCE(cardinality($6::text[]), 0)=0") {
		t.Fatal("nil list filters must be treated as empty lists")
	}
	if strings.Contains(query, "stay_bookings") {
		t.Fatal("availability tables must not be queried without both dates")
	}
	if got, want := len(args), 10; got != want {
		t.Fatalf("argument count = %d, want %d", got, want)
	}
}

func TestBuildSearchQueryWithDatesIncludesAvailability(t *testing.T) {
	t.Parallel()
	query, args := buildSearchQuery(domain.Input{Adults: 2, MinPriceMinor: 5000, MaxPriceMinor: 50000, CheckIn: "2026-09-01", CheckOut: "2026-09-03", PageSize: 8})

	if !strings.Contains(query, "FROM stay_bookings") {
		t.Fatal("availability query must be used when both dates are set")
	}
	if got, want := len(args), 13; got != want {
		t.Fatalf("argument count = %d, want %d", got, want)
	}
}
