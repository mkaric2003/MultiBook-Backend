package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/mkaric2003/multibook-backend/internal/modules/service_search/domain"
)

func TestSearchUsesQueryPort(t *testing.T) {
	t.Parallel()
	queries := &fakeQueries{}
	if _, err := NewService(queries).Search(context.Background(), domain.Input{}); err != nil {
		t.Fatal(err)
	}
	if !queries.searched {
		t.Fatal("search should use query port")
	}
}

func TestSearchSortsAndUsesFirebaseCompatibleCursor(t *testing.T) {
	t.Parallel()
	first, second, third := uuid.New(), uuid.New(), uuid.New()
	queries := &fakeQueries{items: []domain.Item{
		{ID: first, Name: "Zeta", AverageRating: 4.5, ReviewCount: 5, LowestPriceMinor: 2000},
		{ID: second, Name: "Alpha", AverageRating: 4.8, ReviewCount: 1, LowestPriceMinor: 3000},
		{ID: third, Name: "Beta", AverageRating: 4.8, ReviewCount: 2, LowestPriceMinor: 1000},
	}}
	page, err := NewService(queries).Search(context.Background(), domain.Input{PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if page.Items[0].ID != third || page.Items[1].ID != second || page.NextCursor == nil || *page.NextCursor != second.String() {
		t.Fatalf("unexpected first page: %#v", page)
	}
	page, err = NewService(queries).Search(context.Background(), domain.Input{PageSize: 2, Cursor: *page.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != first || page.NextCursor != nil {
		t.Fatalf("unexpected cursor page: %#v", page)
	}
}

func TestSearchNormalizesCityLikeFirebase(t *testing.T) {
	t.Parallel()
	queries := &capturingQueries{}
	if _, err := NewService(queries).Search(context.Background(), domain.Input{City: "  Široki   Brijeg "}); err != nil {
		t.Fatal(err)
	}
	if queries.input.City != "siroki brijeg" {
		t.Fatalf("city = %q", queries.input.City)
	}
}

type fakeQueries struct {
	searched bool
	items    []domain.Item
}

func (q *fakeQueries) Search(context.Context, domain.Input) ([]domain.Item, error) {
	q.searched = true
	return append([]domain.Item(nil), q.items...), nil
}

type capturingQueries struct{ input domain.Input }

func (q *capturingQueries) Search(_ context.Context, input domain.Input) ([]domain.Item, error) {
	q.input = input
	return nil, nil
}
